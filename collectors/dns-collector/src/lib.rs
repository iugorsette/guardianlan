use std::{env, path::PathBuf, time::Duration};

use anyhow::Result;
use common::{load_events_from_path, publish_json_batch, DnsEvent, NatsPublisher, Publisher, DNS_OBSERVED_SUBJECT};
use tracing::info;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DnsSource {
    Disabled,
    Fixture,
}

#[derive(Debug, Clone)]
pub struct Config {
    pub source: DnsSource,
    pub nats_url: String,
    pub fixture: PathBuf,
    pub interval_secs: u64,
    pub oneshot: bool,
}

impl Config {
    pub fn from_env() -> Self {
        Self {
            source: env::var("DNS_SOURCE")
                .ok()
                .map(|value| {
                    if value.eq_ignore_ascii_case("fixture") {
                        DnsSource::Fixture
                    } else {
                        DnsSource::Disabled
                    }
                })
                .unwrap_or(DnsSource::Disabled),
            nats_url: env::var("NATS_URL").unwrap_or_else(|_| "nats://nats:4222".to_string()),
            fixture: env::var("DNS_FIXTURE")
                .map(PathBuf::from)
                .unwrap_or_else(|_| PathBuf::from("/fixtures/queries.json")),
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
