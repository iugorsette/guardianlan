use std::{collections::HashMap, fs, path::Path};

use tracing::warn;

const FALLBACK_OUIS: &[(&str, &str)] = &[
    ("000C43", "Roku"),
    ("001132", "Synology"),
    ("00155D", "Microsoft"),
    ("00163E", "Xiaomi"),
    ("0017F2", "Apple"),
    ("001A11", "Google"),
    ("001B63", "Apple"),
    ("001CF0", "Samsung"),
    ("001DD8", "Netgear"),
    ("00212E", "Intelbras"),
    ("0023CD", "Hikvision"),
    ("00265A", "Dell"),
    ("0026B9", "Apple"),
    ("0026F2", "Netgear"),
    ("00269E", "Espressif"),
    ("0026F4", "Apple"),
    ("0026FE", "Apple"),
    ("00271E", "Xiaomi"),
    ("0023A7", "Sony"),
    ("00259C", "Apple"),
    ("00265B", "Samsung"),
    ("00A040", "Canon"),
    ("00B052", "TP-Link"),
    ("00D02D", "Cisco"),
    ("04E8B9", "Acer"),
    ("08EA44", "Samsung"),
    ("0C8BFD", "Apple"),
    ("10C37B", "TP-Link"),
    ("18FE34", "Espressif"),
    ("1C5CF2", "LG"),
    ("205B2A", "HP"),
    ("2405F5", "Nintendo"),
    ("28E347", "Xiaomi"),
    ("2C3AE8", "Amazon"),
    ("347E5C", "Tuya"),
    ("3C2E5C", "Google"),
    ("40B034", "Apple"),
    ("44E4D9", "Dahua"),
    ("4C5E0C", "Samsung"),
    ("50C7BF", "Amazon"),
    ("5C494F", "Xiaomi"),
    ("607D09", "Intelbras"),
    ("689E29", "TP-Link"),
    ("74DA38", "Google"),
    ("7CD95C", "Apple"),
    ("80B989", "Amazon"),
    ("84D6D0", "Hikvision"),
    ("98F4AB", "Xiaomi"),
    ("98502E", "Intelbras"),
    ("9CC7A6", "Amazon"),
    ("A0A3B3", "Apple"),
    ("A42BB0", "Samsung"),
    ("A4CF12", "Intelbras"),
    ("AC84C6", "Sony"),
    ("B81EA4", "Amazon"),
    ("B827EB", "Raspberry Pi"),
    ("C0F853", "Intelbras"),
    ("D8E844", "Amazon"),
    ("E4956E", "Intelbras"),
    ("F4032A", "Intelbras"),
];

pub fn load_vendor_db(path: Option<&Path>) -> HashMap<String, String> {
    let mut vendors = fallback_vendor_db();

    let Some(path) = path else {
        return vendors;
    };

    let contents = match fs::read_to_string(path) {
        Ok(contents) => contents,
        Err(error) => {
            warn!(path = %path.display(), %error, "could not load OUI database, using fallback vendor map");
            return vendors;
        }
    };

    for line in contents.lines() {
        if let Some((prefix, vendor)) = parse_oui_line(line) {
            vendors.insert(prefix, vendor);
        }
    }

    vendors
}

pub fn lookup_vendor(vendors: &HashMap<String, String>, mac: &str) -> String {
    let prefix = mac
        .chars()
        .filter(|ch| ch.is_ascii_hexdigit())
        .take(6)
        .collect::<String>()
        .to_ascii_uppercase();

    vendors.get(&prefix).cloned().unwrap_or_default()
}

fn fallback_vendor_db() -> HashMap<String, String> {
    FALLBACK_OUIS
        .iter()
        .map(|(prefix, vendor)| ((*prefix).to_string(), (*vendor).to_string()))
        .collect()
}

fn parse_oui_line(line: &str) -> Option<(String, String)> {
    if let Some((prefix, vendor)) = parse_with_marker(line, "(hex)") {
        return Some((prefix, vendor));
    }

    parse_with_marker(line, "(base 16)")
}

fn parse_with_marker(line: &str, marker: &str) -> Option<(String, String)> {
    let (prefix, vendor) = line.split_once(marker)?;
    let normalized = prefix
        .chars()
        .filter(|ch| ch.is_ascii_hexdigit())
        .collect::<String>()
        .to_ascii_uppercase();

    if normalized.len() != 6 {
        return None;
    }

    let vendor = vendor.trim();
    if vendor.is_empty() {
        return None;
    }

    Some((normalized, vendor.to_string()))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_ieee_hex_line() {
        let parsed = parse_oui_line("68-9E-29   (hex)\tTP-Link Systems Inc.");
        assert_eq!(
            parsed,
            Some(("689E29".to_string(), "TP-Link Systems Inc.".to_string()))
        );
    }

    #[test]
    fn uses_fallback_vendor_when_known() {
        let vendors = load_vendor_db(None);
        assert_eq!(lookup_vendor(&vendors, "68:9E:29:C4:C2:95"), "TP-Link");
    }
}
