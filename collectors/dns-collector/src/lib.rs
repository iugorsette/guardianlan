use std::{env, fs, path::PathBuf, time::Duration};

use anyhow::{Context, Result};
use common::{load_events_from_path, publish_json_batch, DnsEvent, NatsPublisher, Publisher, DNS_OBSERVED_SUBJECT};
use serde_json::Value;
use tracing::{info, warn};

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DnsSource {
    Disabled,
    Fixture,
    AdGuardFile,
}

#[derive(Debug, Clone)]
pub struct Config {
    pub source: DnsSource,
    pub nats_url: String,
    pub fixture: PathBuf,
    pub adguard_querylog: PathBuf,
    pub resolver_name: String,
    pub interval_secs: u64,
    pub oneshot: bool,
}

impl Config {
    pub fn from_env() -> Self {
        Self {
            source: env::var("DNS_SOURCE")
                .ok()
                .map(|value| match value.to_ascii_lowercase().as_str() {
                    "fixture" => DnsSource::Fixture,
                    "adguard_file" => DnsSource::AdGuardFile,
                    _ => DnsSource::Disabled,
                })
                .unwrap_or(DnsSource::Disabled),
            nats_url: env::var("NATS_URL").unwrap_or_else(|_| "nats://nats:4222".to_string()),
            fixture: env::var("DNS_FIXTURE")
                .map(PathBuf::from)
                .unwrap_or_else(|_| PathBuf::from("/fixtures/queries.json")),
            adguard_querylog: env::var("DNS_ADGUARD_QUERYLOG")
                .map(PathBuf::from)
                .unwrap_or_else(|_| PathBuf::from("/adguard-work/querylog.json")),
            resolver_name: env::var("DNS_RESOLVER_NAME")
                .unwrap_or_else(|_| "adguardhome".to_string()),
            interval_secs: env::var("DNS_INTERVAL_SECS")
                .ok()
                .and_then(|value| value.parse().ok())
                .unwrap_or(45),
            oneshot: env::var("DNS_ONESHOT")
                .ok()
                .map(|value| value == "true")
                .unwrap_or(false),
        }
    }
}

pub async fn run_once<P: Publisher>(fixture: &PathBuf, publisher: &P) -> Result<usize> {
    let events = load_events_from_path::<DnsEvent>(fixture)?;
    publish_json_batch(publisher, DNS_OBSERVED_SUBJECT, &events).await
}

pub async fn run_adguard_once<P: Publisher>(querylog: &PathBuf, resolver_name: &str, publisher: &P) -> Result<usize> {
    let events = load_adguard_events(querylog, resolver_name)?;
    publish_json_batch(publisher, DNS_OBSERVED_SUBJECT, &events).await
}

pub async fn run(config: Config) -> Result<()> {
    let client = async_nats::connect(config.nats_url.clone()).await?;
    let publisher = NatsPublisher::new(client);
    run_with_publisher(config, publisher).await
}

pub async fn run_with_publisher<P: Publisher>(config: Config, publisher: P) -> Result<()> {
    loop {
        match config.source {
            DnsSource::Fixture => {
                let published = run_once(&config.fixture, &publisher).await?;
                info!(published, "published dns events");
            }
            DnsSource::AdGuardFile => {
                let published = run_adguard_once(&config.adguard_querylog, &config.resolver_name, &publisher).await?;
                info!(published, querylog = %config.adguard_querylog.display(), "published dns events from adguard querylog");
            }
            DnsSource::Disabled => {
                info!("dns collector is disabled");
            }
        }

        if config.oneshot {
            break;
        }

        tokio::time::sleep(Duration::from_secs(config.interval_secs)).await;
    }

    Ok(())
}

fn load_adguard_events(path: &PathBuf, resolver_name: &str) -> Result<Vec<DnsEvent>> {
    let contents = match fs::read_to_string(path) {
        Ok(contents) => contents,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
            warn!(querylog = %path.display(), "adguard querylog file not found yet");
            return Ok(Vec::new());
        }
        Err(error) => return Err(error).with_context(|| format!("read adguard querylog {}", path.display())),
    };

    parse_adguard_events(&contents, resolver_name)
}

fn parse_adguard_events(contents: &str, resolver_name: &str) -> Result<Vec<DnsEvent>> {
    let trimmed = contents.trim();
    if trimmed.is_empty() {
        return Ok(Vec::new());
    }

    let entries = if trimmed.starts_with('[') {
        serde_json::from_str::<Vec<Value>>(trimmed).context("parse adguard querylog array")?
    } else {
        trimmed
            .lines()
            .filter(|line| !line.trim().is_empty())
            .map(|line| serde_json::from_str::<Value>(line).context("parse adguard querylog line"))
            .collect::<Result<Vec<_>>>()?
    };

    Ok(entries
        .into_iter()
        .filter_map(|entry| normalize_adguard_entry(&entry, resolver_name))
        .collect())
}

fn normalize_adguard_entry(entry: &Value, resolver_name: &str) -> Option<DnsEvent> {
    let observed_at = first_string(entry, &[
        &["time"],
        &["timestamp"],
        &["ts"],
        &["t"],
    ])?;

    let domain = first_string(entry, &[
        &["question", "name"],
        &["question", "host"],
        &["qhost"],
        &["host"],
        &["domain"],
        &["query"],
    ])?;

    let query = first_string(entry, &[
        &["query"],
        &["question", "name"],
        &["question", "host"],
        &["domain"],
        &["qhost"],
    ])
    .unwrap_or_else(|| domain.clone());

    let client_hint = first_string(entry, &[
        &["client"],
        &["client_ip"],
        &["clientIP"],
        &["src_ip"],
        &["ip"],
    ])
    .unwrap_or_default();
    let client_name = first_string(entry, &[
        &["client_name"],
        &["clientName"],
        &["device_name"],
        &["name"],
    ])
    .unwrap_or_default();

    let (client_ip, inferred_name) = split_client_hint(&client_hint);
    let category = first_string(entry, &[
        &["category"],
        &["reason"],
        &["result", "reason"],
    ])
    .unwrap_or_else(|| infer_category(&domain));
    let blocked = first_bool(entry, &[
        &["blocked"],
        &["is_filtered"],
        &["result", "is_filtered"],
        &["result", "blocked"],
        &["status", "blocked"],
    ])
    .unwrap_or_else(|| infer_blocked(entry));

    Some(DnsEvent {
        device_id: String::new(),
        client_ip,
        client_name: if client_name.is_empty() { inferred_name } else { client_name },
        query,
        domain: domain.to_ascii_lowercase(),
        category,
        resolver: resolver_name.to_string(),
        blocked,
        observed_at,
    })
}

fn split_client_hint(value: &str) -> (String, String) {
    let trimmed = value.trim();
    if trimmed.is_empty() {
        return (String::new(), String::new());
    }

    if looks_like_ip(trimmed) {
        return (trimmed.to_string(), String::new());
    }

    if let Some((name, ip)) = trimmed.rsplit_once(' ') {
        if looks_like_ip(ip) {
            return (ip.to_string(), name.trim().to_string());
        }
    }

    (String::new(), trimmed.to_string())
}

fn first_string(entry: &Value, paths: &[&[&str]]) -> Option<String> {
    for path in paths {
        if let Some(value) = get_path(entry, path).and_then(|value| value.as_str()) {
            let trimmed = value.trim();
            if !trimmed.is_empty() {
                return Some(trimmed.to_string());
            }
        }
    }

    None
}

fn first_bool(entry: &Value, paths: &[&[&str]]) -> Option<bool> {
    for path in paths {
        if let Some(value) = get_path(entry, path) {
            if let Some(boolean) = value.as_bool() {
                return Some(boolean);
            }
            if let Some(text) = value.as_str() {
                match text.to_ascii_lowercase().as_str() {
                    "true" | "blocked" | "filtered" => return Some(true),
                    "false" | "allowed" => return Some(false),
                    _ => {}
                }
            }
        }
    }

    None
}

fn get_path<'a>(entry: &'a Value, path: &[&str]) -> Option<&'a Value> {
    let mut current = entry;
    for segment in path {
        current = current.get(*segment)?;
    }
    Some(current)
}

fn infer_blocked(entry: &Value) -> bool {
    first_string(entry, &[&["status"], &["result", "status"]])
        .map(|value| matches!(value.to_ascii_lowercase().as_str(), "blocked" | "filtered"))
        .unwrap_or(false)
}

fn infer_category(domain: &str) -> String {
    if domain.contains("xvideos") || domain.contains("porn") || domain.contains("adult") {
        return "adult".to_string();
    }
    "unknown".to_string()
}

fn looks_like_ip(value: &str) -> bool {
    value.parse::<std::net::IpAddr>().is_ok()
}

#[cfg(test)]
mod tests {
    use super::parse_adguard_events;

    #[test]
    fn parses_adguard_json_lines() {
        let events = parse_adguard_events(
            r#"{"time":"2026-03-26T18:00:00Z","client":"192.168.1.25","question":{"name":"www.xvideos.com"},"blocked":true,"category":"adult"}
{"time":"2026-03-26T18:01:00Z","client_name":"Kid Tablet","client_ip":"192.168.1.25","question":{"name":"escola.local"},"blocked":false}"#,
            "adguardhome",
        )
        .unwrap();

        assert_eq!(events.len(), 2);
        assert_eq!(events[0].client_ip, "192.168.1.25");
        assert_eq!(events[0].domain, "www.xvideos.com");
        assert!(events[0].blocked);
        assert_eq!(events[1].client_name, "Kid Tablet");
        assert_eq!(events[1].resolver, "adguardhome");
    }
}
