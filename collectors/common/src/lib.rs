use std::{fs, path::Path};

use anyhow::Result;
use async_trait::async_trait;
use serde::{de::DeserializeOwned, Deserialize, Serialize};

pub const DEVICE_DISCOVERED_SUBJECT: &str = "network.device.discovered";
pub const DEVICE_UPDATED_SUBJECT: &str = "network.device.updated";
pub const DNS_OBSERVED_SUBJECT: &str = "network.dns.query_observed";
pub const FLOW_OBSERVED_SUBJECT: &str = "network.flow.observed";
pub const ALERT_RAISED_SUBJECT: &str = "network.alert.raised";

#[derive(Debug, Clone, Default, Serialize, Deserialize, PartialEq, Eq)]
pub struct DeviceEvidence {
    #[serde(default)]
    pub open_ports: Vec<u16>,
    #[serde(default)]
    pub services: Vec<String>,
    #[serde(default)]
    pub candidate_snapshot_urls: Vec<String>,
    #[serde(default)]
    pub candidate_stream_urls: Vec<String>,
    #[serde(default)]
    pub preview_supported: bool,
    #[serde(default)]
    pub preview_requires_auth: bool,
    #[serde(default)]
    pub confidence: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct DeviceEvent {
    pub id: String,
    pub mac: String,
    pub ips: Vec<String>,
    pub hostname: String,
    pub vendor: String,
    pub device_type: String,
    pub profile_id: String,
    pub managed: bool,
    #[serde(default)]
    pub evidence: DeviceEvidence,
    pub observed_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct DnsEvent {
    pub device_id: String,
    pub query: String,
    pub domain: String,
    pub category: String,
    pub resolver: String,
    pub blocked: bool,
    pub observed_at: String,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq, Eq)]
pub struct FlowEvent {
    pub device_id: String,
    pub src_ip: String,
    pub dst_ip: String,
    pub dst_port: u16,
    pub protocol: String,
    pub bytes_in: u64,
    pub bytes_out: u64,
    pub observed_at: String,
}

pub fn load_events_from_path<T>(path: &Path) -> Result<Vec<T>>
where
    T: DeserializeOwned,
{
    let contents = fs::read_to_string(path)?;
    Ok(serde_json::from_str(&contents)?)
}

#[async_trait]
pub trait Publisher: Send + Sync {
    async fn publish(&self, subject: &str, payload: Vec<u8>) -> Result<()>;
}

pub struct NatsPublisher {
    client: async_nats::Client,
}

impl NatsPublisher {
    pub fn new(client: async_nats::Client) -> Self {
        Self { client }
    }
}

#[async_trait]
impl Publisher for NatsPublisher {
    async fn publish(&self, subject: &str, payload: Vec<u8>) -> Result<()> {
        self.client.publish(subject.to_string(), payload.into()).await?;
        Ok(())
    }
}

pub async fn publish_json_batch<P, T>(publisher: &P, subject: &str, events: &[T]) -> Result<usize>
where
    P: Publisher,
    T: Serialize + Send + Sync,
{
    for event in events {
        publisher
            .publish(subject, serde_json::to_vec(event)?)
            .await?;
    }

    Ok(events.len())
}
