use std::{
    collections::{HashMap, HashSet},
    env,
    net::{IpAddr, Ipv4Addr, SocketAddr, TcpStream},
    path::PathBuf,
    process::Command,
    str::FromStr,
    time::Duration,
};

use anyhow::{Context, Result};
use common::{
    load_events_from_path, publish_json_batch, DeviceEvidence, DeviceEvent, NatsPublisher, Publisher,
    DEVICE_DISCOVERED_SUBJECT, DEVICE_UPDATED_SUBJECT,
};
use serde::Deserialize;
use time::{format_description::well_known::Rfc3339, OffsetDateTime};
use tracing::{info, warn};

mod vendor_oui;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DiscoverySource {
    Fixture,
    Live,
}

#[derive(Debug, Clone)]
pub struct Config {
    pub source: DiscoverySource,
    pub nats_url: String,
    pub fixture: PathBuf,
    pub interval_secs: u64,
    pub max_hosts: u32,
    pub interface_allowlist: Vec<String>,
    pub ping_enabled: bool,
    pub fingerprint_enabled: bool,
    pub fingerprint_timeout_ms: u64,
    pub vendor_db_path: Option<PathBuf>,
    pub oneshot: bool,
}

impl Config {
    pub fn from_env() -> Self {
        Self {
            source: env::var("DISCOVERY_SOURCE")
                .ok()
                .map(|value| {
                    if value.eq_ignore_ascii_case("fixture") {
                        DiscoverySource::Fixture
                    } else {
                        DiscoverySource::Live
                    }
                })
                .unwrap_or(DiscoverySource::Live),
            nats_url: env::var("NATS_URL").unwrap_or_else(|_| "nats://nats:4222".to_string()),
            fixture: env::var("DISCOVERY_FIXTURE")
                .map(PathBuf::from)
                .unwrap_or_else(|_| PathBuf::from("/fixtures/devices.json")),
            interval_secs: env::var("DISCOVERY_INTERVAL_SECS")
                .ok()
                .and_then(|value| value.parse().ok())
                .unwrap_or(60),
            max_hosts: env::var("DISCOVERY_MAX_HOSTS")
                .ok()
                .and_then(|value| value.parse().ok())
                .unwrap_or(256),
            interface_allowlist: env::var("DISCOVERY_INTERFACE_ALLOWLIST")
                .ok()
                .map(|value| {
                    value
                        .split(',')
                        .map(str::trim)
                        .filter(|name| !name.is_empty())
                        .map(ToOwned::to_owned)
                        .collect()
                })
                .unwrap_or_default(),
            ping_enabled: env::var("DISCOVERY_PING_ENABLED")
                .ok()
                .map(|value| value != "false")
                .unwrap_or(true),
            fingerprint_enabled: env::var("DISCOVERY_FINGERPRINT_ENABLED")
                .ok()
                .map(|value| value != "false")
                .unwrap_or(true),
            fingerprint_timeout_ms: env::var("DISCOVERY_FINGERPRINT_TIMEOUT_MS")
                .ok()
                .and_then(|value| value.parse().ok())
                .unwrap_or(180),
            vendor_db_path: env::var("DISCOVERY_VENDOR_DB")
                .ok()
                .filter(|value| !value.trim().is_empty())
                .map(PathBuf::from)
                .or_else(|| Some(PathBuf::from("/usr/share/ieee-data/oui.txt"))),
            oneshot: env::var("DISCOVERY_ONESHOT")
                .ok()
                .map(|value| value == "true")
                .unwrap_or(false),
        }
    }
}

pub async fn run_once<P: Publisher>(fixture: &PathBuf, publisher: &P, subject: &str) -> Result<usize> {
    let events = load_events_from_path::<DeviceEvent>(fixture)?;
    publish_json_batch(publisher, subject, &events).await
}

pub fn discover_live_devices(config: &Config) -> Result<Vec<DeviceEvent>> {
    let now = OffsetDateTime::now_utc().format(&Rfc3339)?;
    let interfaces = local_interfaces(config)?;
    let gateways = default_gateways()?;
    let vendors = vendor_oui::load_vendor_db(config.vendor_db_path.as_deref());
    let fingerprint_timeout = Duration::from_millis(config.fingerprint_timeout_ms);

    if interfaces.is_empty() {
        warn!("no candidate LAN interfaces found for live discovery");
        return Ok(Vec::new());
    }

    for interface in &interfaces {
        if config.ping_enabled {
            trigger_ping_sweep(interface);
        }
    }

    let mut devices: HashMap<String, DeviceEvent> = HashMap::new();
    for interface in &interfaces {
        for neighbor in neighbours_for(interface)? {
            let device_id = make_device_id(neighbor.lladdr.as_deref(), &neighbor.dst);
            let mac = normalize_mac(neighbor.lladdr.as_deref());
            let vendor = vendor_oui::lookup_vendor(&vendors, &mac);
            let hostname = reverse_dns_name(&neighbor.dst)
                .unwrap_or_else(|| fallback_hostname(&neighbor.dst));
            let open_ports = if config.fingerprint_enabled {
                fingerprint_host(&neighbor.dst, fingerprint_timeout)
            } else {
                Vec::new()
            };
            let device_type =
                classify_device(&hostname, &neighbor.dst, &vendor, &open_ports, &gateways);
            let profile_id = match device_type.as_str() {
                "camera" | "iot" | "router" | "printer" | "tv" | "nas" => "iot",
                _ => "guest",
            }
            .to_string();

            let entry = devices.entry(device_id.clone()).or_insert_with(|| DeviceEvent {
                id: device_id.clone(),
                mac: mac.clone(),
                ips: Vec::new(),
                hostname: hostname.clone(),
                vendor: vendor.clone(),
                device_type: device_type.clone(),
                profile_id: profile_id.clone(),
                managed: false,
                evidence: build_device_evidence(&neighbor.dst, &device_type, &open_ports),
                observed_at: now.clone(),
            });

            if !entry.ips.iter().any(|ip| ip == &neighbor.dst) {
                entry.ips.push(neighbor.dst.clone());
            }

            if entry.hostname.is_empty() && !hostname.is_empty() {
                entry.hostname = hostname.clone();
            }

            if entry.vendor.is_empty() && !vendor.is_empty() {
                entry.vendor = vendor.clone();
            }

            if entry.device_type == "unknown" && device_type != "unknown" {
                entry.device_type = device_type;
                entry.profile_id = profile_id;
            }

            if entry.evidence.open_ports.is_empty() && !open_ports.is_empty() {
                entry.evidence = build_device_evidence(&neighbor.dst, &entry.device_type, &open_ports);
            }

            entry.observed_at = now.clone();
        }
    }

    let mut collected = devices.into_values().collect::<Vec<_>>();
    collected.sort_by(|left, right| left.hostname.cmp(&right.hostname));
    Ok(collected)
}

pub async fn run(config: Config) -> Result<()> {
    let client = async_nats::connect(config.nats_url.clone()).await?;
    let publisher = NatsPublisher::new(client);
    run_with_publisher(config, publisher).await
}

pub async fn run_with_publisher<P: Publisher>(config: Config, publisher: P) -> Result<()> {
    let mut first_batch = true;

    loop {
        let subject = if first_batch {
            DEVICE_DISCOVERED_SUBJECT
        } else {
            DEVICE_UPDATED_SUBJECT
        };

        let events = match config.source {
            DiscoverySource::Fixture => load_events_from_path::<DeviceEvent>(&config.fixture)?,
            DiscoverySource::Live => discover_live_devices(&config)?,
        };
        let published = publish_json_batch(&publisher, subject, &events).await?;
        info!(subject, published, "published discovery events");

        if config.oneshot {
            break;
        }

        first_batch = false;
        tokio::time::sleep(Duration::from_secs(config.interval_secs)).await;
    }

    Ok(())
}

#[derive(Debug, Deserialize)]
struct IpLinkRecord {
    ifname: String,
    #[serde(default)]
    operstate: Option<String>,
    #[serde(default)]
    addr_info: Vec<IpAddrInfo>,
}

#[derive(Debug, Deserialize)]
struct IpAddrInfo {
    family: String,
    local: String,
    prefixlen: u8,
}

#[derive(Debug, Clone)]
struct LocalInterface {
    ifname: String,
    ipv4: Ipv4Addr,
    prefixlen: u8,
}

#[derive(Debug, Deserialize)]
struct NeighbourRecord {
    dst: String,
    #[serde(default)]
    lladdr: Option<String>,
    #[serde(default)]
    state: Option<Vec<String>>,
}

#[derive(Debug, Clone)]
struct DeviceNeighbour {
    dst: String,
    lladdr: Option<String>,
}

#[derive(Debug, Deserialize)]
struct RouteRecord {
    #[serde(default)]
    gateway: Option<String>,
}

fn local_interfaces(config: &Config) -> Result<Vec<LocalInterface>> {
    let output = Command::new("ip")
        .args(["-j", "-4", "addr", "show", "up"])
        .output()
        .context("run ip addr")?;

    let links: Vec<IpLinkRecord> =
        serde_json::from_slice(&output.stdout).context("parse ip addr json")?;

    let allowlist: HashSet<&str> = config.interface_allowlist.iter().map(String::as_str).collect();
    let mut interfaces = Vec::new();

    for link in links {
        if !allowlist.is_empty() && !allowlist.contains(link.ifname.as_str()) {
            continue;
        }
        if !is_candidate_interface(&link.ifname, link.operstate.as_deref()) {
            continue;
        }

        for addr in link.addr_info {
            if addr.family != "inet" {
                continue;
            }

            let ipv4 = match Ipv4Addr::from_str(&addr.local) {
                Ok(ipv4) if is_private_ipv4(&ipv4) => ipv4,
                _ => continue,
            };

            if host_capacity(addr.prefixlen) > config.max_hosts {
                warn!(
                    interface = link.ifname,
                    prefixlen = addr.prefixlen,
                    max_hosts = config.max_hosts,
                    "skipping interface because subnet is larger than configured max hosts"
                );
                continue;
            }

            interfaces.push(LocalInterface {
                ifname: link.ifname.clone(),
                ipv4,
                prefixlen: addr.prefixlen,
            });
        }
    }

    Ok(interfaces)
}

fn default_gateways() -> Result<HashSet<String>> {
    let output = Command::new("ip")
        .args(["-j", "route", "show", "default"])
        .output()
        .context("run ip route")?;

    let routes: Vec<RouteRecord> =
        serde_json::from_slice(&output.stdout).context("parse ip route json")?;

    Ok(routes
        .into_iter()
        .filter_map(|route| route.gateway)
        .collect())
}

fn trigger_ping_sweep(interface: &LocalInterface) {
    let cidr = format!("{}/{}", network_address(interface.ipv4, interface.prefixlen), interface.prefixlen);
    let _ = Command::new("fping")
        .args(["-a", "-q", "-g", &cidr, "-I", &interface.ifname])
        .output();
}

fn neighbours_for(interface: &LocalInterface) -> Result<Vec<DeviceNeighbour>> {
    let output = Command::new("ip")
        .args(["-j", "neigh", "show", "dev", &interface.ifname])
        .output()
        .with_context(|| format!("read neighbour table for {}", interface.ifname))?;

    let neighbours: Vec<NeighbourRecord> =
        serde_json::from_slice(&output.stdout).context("parse neighbour json")?;

    let devices = neighbours
        .into_iter()
        .filter(|entry| {
            entry
                .state
                .as_ref()
                .map(|state| {
                    !state.iter().any(|item| {
                        matches!(item.as_str(), "FAILED" | "INCOMPLETE" | "NOARP")
                    })
                })
                .unwrap_or(true)
        })
        .filter_map(|entry| {
            let ip = Ipv4Addr::from_str(&entry.dst).ok()?;
            if !is_private_ipv4(&ip) {
                return None;
            }

            Some(DeviceNeighbour {
                dst: entry.dst,
                lladdr: entry.lladdr,
            })
        })
        .collect();

    Ok(devices)
}

fn reverse_dns_name(ip: &str) -> Option<String> {
    let output = Command::new("getent").args(["hosts", ip]).output().ok()?;
    if !output.status.success() {
        return None;
    }

    let stdout = String::from_utf8(output.stdout).ok()?;
    let hostname = stdout.split_whitespace().nth(1)?;
    if hostname == ip {
        return None;
    }

    Some(hostname.to_string())
}

fn fallback_hostname(ip: &str) -> String {
    format!("device-{}", ip.replace('.', "-"))
}

fn classify_device(
    hostname: &str,
    ip: &str,
    vendor: &str,
    open_ports: &[u16],
    gateways: &HashSet<String>,
) -> String {
    let lower = hostname.to_ascii_lowercase();
    let vendor_lower = vendor.to_ascii_lowercase();

    if gateways.contains(ip) || lower.contains("router") || lower.contains("gateway") {
        return "router".to_string();
    }
    if ["cam", "camera", "baby", "monitor", "tapo", "rtsp"]
        .iter()
        .any(|needle| lower.contains(needle))
    {
        return "camera".to_string();
    }
    if has_any_port(open_ports, &[554, 8554, 37777]) {
        return "camera".to_string();
    }
    if ["ipad", "tablet", "tab"].iter().any(|needle| lower.contains(needle)) {
        return "tablet".to_string();
    }
    if ["iphone", "android", "phone", "pixel", "galaxy"]
        .iter()
        .any(|needle| lower.contains(needle))
    {
        return "phone".to_string();
    }
    if ["macbook", "laptop", "notebook", "predator", "thinkpad", "xps", "desktop", "pc"]
        .iter()
        .any(|needle| lower.contains(needle))
    {
        return "computer".to_string();
    }
    if ["tv", "chromecast", "roku", "firetv"].iter().any(|needle| lower.contains(needle)) {
        return "tv".to_string();
    }
    if has_any_port(open_ports, &[8008, 8009, 7000]) {
        return "tv".to_string();
    }
    if ["printer", "epson", "hp-", "canon"].iter().any(|needle| lower.contains(needle)) {
        return "printer".to_string();
    }
    if has_any_port(open_ports, &[631, 9100, 515]) {
        return "printer".to_string();
    }
    if ["echo", "alexa", "speaker", "nest"].iter().any(|needle| lower.contains(needle)) {
        return "iot".to_string();
    }
    if has_any_port(open_ports, &[5000, 5001, 445]) {
        return "nas".to_string();
    }
    if matches_vendor(&vendor_lower, CAMERA_VENDORS) {
        return "camera".to_string();
    }
    if matches_vendor(&vendor_lower, PRINTER_VENDORS) {
        return "printer".to_string();
    }
    if matches_vendor(&vendor_lower, TV_VENDORS) {
        return "tv".to_string();
    }
    if matches_vendor(&vendor_lower, ROUTER_VENDORS) {
        return "router".to_string();
    }
    if matches_vendor(&vendor_lower, IOT_VENDORS) {
        return "iot".to_string();
    }

    "unknown".to_string()
}

fn fingerprint_host(ip: &str, timeout: Duration) -> Vec<u16> {
    let Ok(ipv4) = Ipv4Addr::from_str(ip) else {
        return Vec::new();
    };

    COMMON_PROBE_PORTS
        .iter()
        .copied()
        .filter(|port| tcp_port_open(ipv4, *port, timeout))
        .collect()
}

fn build_device_evidence(ip: &str, device_type: &str, open_ports: &[u16]) -> DeviceEvidence {
    let services = infer_services(open_ports);
    let candidate_snapshot_urls = snapshot_urls(ip, open_ports, device_type);
    let candidate_stream_urls = stream_urls(ip, open_ports, device_type);
    let preview_supported = !candidate_snapshot_urls.is_empty() || !candidate_stream_urls.is_empty();
    let preview_requires_auth = preview_supported;
    let confidence = match device_type {
        "camera" => {
            if has_any_port(open_ports, &[554, 8554, 80, 443, 8080]) {
                "high"
            } else {
                "medium"
            }
        }
        "router" | "computer" => "high",
        "iot" | "tv" | "printer" => "medium",
        _ => "low",
    }
    .to_string();

    DeviceEvidence {
        open_ports: open_ports.to_vec(),
        services,
        candidate_snapshot_urls,
        candidate_stream_urls,
        preview_supported,
        preview_requires_auth,
        confidence,
    }
}

fn tcp_port_open(ip: Ipv4Addr, port: u16, timeout: Duration) -> bool {
    let address = SocketAddr::new(IpAddr::V4(ip), port);
    TcpStream::connect_timeout(&address, timeout).is_ok()
}

fn has_any_port(open_ports: &[u16], candidates: &[u16]) -> bool {
    candidates.iter().any(|port| open_ports.contains(port))
}

fn matches_vendor(vendor: &str, patterns: &[&str]) -> bool {
    !vendor.is_empty() && patterns.iter().any(|pattern| vendor.contains(pattern))
}

fn infer_services(open_ports: &[u16]) -> Vec<String> {
    let mut services = Vec::new();
    for port in open_ports {
        let service = match port {
            53 => Some("dns"),
            80 => Some("http"),
            139 => Some("netbios"),
            443 => Some("https"),
            445 => Some("smb"),
            515 => Some("lpd"),
            554 | 8554 => Some("rtsp"),
            631 => Some("ipp"),
            7000 => Some("airplay"),
            8008 | 8009 => Some("cast"),
            8080 => Some("http-alt"),
            8443 => Some("https-alt"),
            8883 => Some("mqtts"),
            9100 => Some("jetdirect"),
            1883 => Some("mqtt"),
            37777 => Some("dvr"),
            5000 | 5001 => Some("nas"),
            _ => None,
        };

        if let Some(service) = service {
            if !services.iter().any(|item| item == service) {
                services.push(service.to_string());
            }
        }
    }

    services
}

fn snapshot_urls(ip: &str, open_ports: &[u16], device_type: &str) -> Vec<String> {
    if device_type != "camera" {
        return Vec::new();
    }

    let mut urls = Vec::new();
    let ports = if open_ports.is_empty() {
        vec![80, 8080, 443]
    } else {
        open_ports.to_vec()
    };

    for port in ports {
        match port {
            80 | 8080 => {
                let scheme = "http";
                for path in [
                    "/snapshot.jpg",
                    "/cgi-bin/snapshot.cgi",
                    "/ISAPI/Streaming/channels/101/picture",
                    "/webcapture.jpg?command=snap&channel=1",
                ] {
                    urls.push(format!("{scheme}://{ip}:{port}{path}"));
                }
            }
            443 | 8443 => {
                let scheme = "https";
                for path in [
                    "/snapshot.jpg",
                    "/cgi-bin/snapshot.cgi",
                    "/ISAPI/Streaming/channels/101/picture",
                ] {
                    urls.push(format!("{scheme}://{ip}:{port}{path}"));
                }
            }
            _ => {}
        }
    }

    urls
}

fn stream_urls(ip: &str, open_ports: &[u16], device_type: &str) -> Vec<String> {
    if device_type != "camera" {
        return Vec::new();
    }

    let mut urls = Vec::new();
    let ports = if open_ports.is_empty() {
        vec![554, 8554]
    } else {
        open_ports.to_vec()
    };

    for port in ports {
        match port {
            554 | 8554 => {
                for path in [
                    "/",
                    "/stream1",
                    "/h264Preview_01_main",
                    "/cam/realmonitor?channel=1&subtype=0",
                    "/live/ch0",
                ] {
                    urls.push(format!("rtsp://{ip}:{port}{path}"));
                }
            }
            80 | 8080 => {
                urls.push(format!("http://{ip}:{port}/video"));
                urls.push(format!("http://{ip}:{port}/mjpeg"));
            }
            _ => {}
        }
    }

    urls
}

fn make_device_id(mac: Option<&str>, ip: &str) -> String {
    match mac {
        Some(mac) if !mac.is_empty() => format!("device-{}", mac.replace(':', "-").to_ascii_lowercase()),
        _ => format!("device-{}", ip.replace('.', "-")),
    }
}

fn normalize_mac(mac: Option<&str>) -> String {
    mac.unwrap_or_default().to_ascii_uppercase()
}

fn host_capacity(prefixlen: u8) -> u32 {
    if prefixlen >= 32 {
        return 1;
    }

    let hosts = 1u128 << (32 - prefixlen as u32);
    hosts.min(u128::from(u32::MAX)) as u32
}

fn network_address(ip: Ipv4Addr, prefixlen: u8) -> Ipv4Addr {
    let ip_value = u32::from(ip);
    let mask = if prefixlen == 0 {
        0
    } else {
        u32::MAX << (32 - prefixlen)
    };

    Ipv4Addr::from(ip_value & mask)
}

fn is_private_ipv4(ip: &Ipv4Addr) -> bool {
    let octets = ip.octets();
    if octets[0] == 10 {
        return true;
    }
    if octets[0] == 192 && octets[1] == 168 {
        return true;
    }

    octets[0] == 172 && (16..=31).contains(&octets[1])
}

fn is_candidate_interface(name: &str, operstate: Option<&str>) -> bool {
    if matches!(operstate, Some("DOWN")) {
        return false;
    }

    !matches!(
        name,
        "lo"
    ) && !["docker", "br-", "veth", "virbr", "podman", "tailscale", "tun", "tap"]
        .iter()
        .any(|prefix| name.starts_with(prefix))
}

const COMMON_PROBE_PORTS: &[u16] = &[
    53, 80, 139, 443, 445, 515, 554, 631, 7000, 8000, 8008, 8009, 8080, 8443, 8554, 8883, 9100,
    1883, 37777, 5000, 5001,
];

const CAMERA_VENDORS: &[&str] = &[
    "hikvision",
    "dahua",
    "reolink",
    "ezviz",
    "imou",
    "amcrest",
    "foscam",
    "wyze",
    "eufy",
    "arlo",
    "intelbras",
];

const PRINTER_VENDORS: &[&str] = &["epson", "brother", "canon", "lexmark", "xerox", "hewlett", "hp"];

const TV_VENDORS: &[&str] = &["roku", "samsung", "lg", "hisense", "sony", "tcl", "chromecast"];

const ROUTER_VENDORS: &[&str] = &[
    "tp-link",
    "netgear",
    "mikrotik",
    "ubiquiti",
    "arris",
    "technicolor",
    "tenda",
    "mercusys",
    "d-link",
    "intelbras",
    "xiaomi",
];

const IOT_VENDORS: &[&str] = &["amazon", "google", "tuya", "espressif", "raspberry pi"];

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn host_capacity_limits_small_networks() {
        assert_eq!(host_capacity(24), 256);
        assert_eq!(host_capacity(30), 4);
    }

    #[test]
    fn network_address_uses_prefix() {
        let network = network_address(Ipv4Addr::new(192, 168, 1, 42), 24);
        assert_eq!(network, Ipv4Addr::new(192, 168, 1, 0));
    }

    #[test]
    fn interface_filter_skips_virtual_links() {
        assert!(is_candidate_interface("enp67s0", Some("UP")));
        assert!(!is_candidate_interface("docker0", Some("UP")));
        assert!(!is_candidate_interface("lo", Some("UNKNOWN")));
    }

    #[test]
    fn camera_keywords_are_classified() {
        let gateways = HashSet::new();
        assert_eq!(
            classify_device("baby-cam.local", "192.168.1.10", "", &[], &gateways),
            "camera"
        );
    }

    #[test]
    fn camera_ports_are_classified_even_without_hostname() {
        let gateways = HashSet::new();
        assert_eq!(
            classify_device("device-192-168-1-20", "192.168.1.20", "", &[554], &gateways),
            "camera"
        );
    }

    #[test]
    fn printer_vendor_is_classified() {
        let gateways = HashSet::new();
        assert_eq!(
            classify_device(
                "device-192-168-1-30",
                "192.168.1.30",
                "HP",
                &[],
                &gateways
            ),
            "printer"
        );
    }

    #[test]
    fn laptop_keywords_are_classified_as_computer() {
        let gateways = HashSet::new();
        assert_eq!(
            classify_device(
                "sette-Predator-PHN16-72",
                "192.168.1.50",
                "Acer",
                &[],
                &gateways
            ),
            "computer"
        );
    }

    #[test]
    fn camera_evidence_exposes_preview_candidates() {
        let evidence = build_device_evidence("192.168.1.21", "camera", &[80, 554]);
        assert!(evidence.preview_supported);
        assert!(evidence.preview_requires_auth);
        assert!(evidence
            .candidate_snapshot_urls
            .iter()
            .any(|url| url.contains("snapshot")));
        assert!(evidence
            .candidate_stream_urls
            .iter()
            .any(|url| url.starts_with("rtsp://192.168.1.21:554")));
    }

    #[test]
    fn camera_evidence_falls_back_to_default_preview_paths() {
        let evidence = build_device_evidence("192.168.1.21", "camera", &[]);
        assert!(evidence.preview_supported);
        assert!(!evidence.candidate_snapshot_urls.is_empty());
        assert!(!evidence.candidate_stream_urls.is_empty());
    }
}
