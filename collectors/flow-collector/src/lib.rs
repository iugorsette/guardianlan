use std::{env, path::PathBuf, time::Duration};

use anyhow::Result;
use common::{load_events_from_path, publish_json_batch, FlowEvent, NatsPublisher, Publisher, FLOW_OBSERVED_SUBJECT};
use tracing::info;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FlowSource {
    Disabled,
    Fixture,
}

#[derive(Debug, Clone)]
pub struct Config {
    pub source: FlowSource,
    pub nats_url: String,
    pub fixture: PathBuf,
    pub interval_secs: u64,
    pub oneshot: bool,
}

impl Config {
    pub fn from_env() -> Self {
        Self {
            source: env::var("FLOW_SOURCE")
                .ok()
                .map(|value| {
                    if value.eq_ignore_ascii_case("fixture") {
                        FlowSource::Fixture
                    } else {
                        FlowSource::Disabled
                    }
                })
                .unwrap_or(FlowSource::Disabled),
            nats_url: env::var("NATS_URL").unwrap_or_else(|_| "nats://nats:4222".to_string()),
            fixture: env::var("FLOW_FIXTURE")
                .map(PathBuf::from)
                .unwrap_or_else(|_| PathBuf::from("/fixtures/events.json")),
            interval_secs: env::var("FLOW_INTERVAL_SECS")
                .ok()
                .and_then(|value| value.parse().ok())
                .unwrap_or(90),
            oneshot: env::var("FLOW_ONESHOT")
                .ok()
                .map(|value| value == "true")
                .unwrap_or(false),
        }
    }
}

pub async fn run_once<P: Publisher>(fixture: &PathBuf, publisher: &P) -> Result<usize> {
    let events = load_events_from_path::<FlowEvent>(fixture)?;
    publish_json_batch(publisher, FLOW_OBSERVED_SUBJECT, &events).await
}

pub async fn run(config: Config) -> Result<()> {
    let client = async_nats::connect(config.nats_url.clone()).await?;
    let publisher = NatsPublisher::new(client);
    run_with_publisher(config, publisher).await
}

pub async fn run_with_publisher<P: Publisher>(config: Config, publisher: P) -> Result<()> {
    loop {
        match config.source {
            FlowSource::Fixture => {
                let published = run_once(&config.fixture, &publisher).await?;
                info!(published, "published flow events");
            }
            FlowSource::Disabled => {
                info!("flow collector is disabled");
            }
        }

        if config.oneshot {
            break;
        }

        tokio::time::sleep(Duration::from_secs(config.interval_secs)).await;
    }

    Ok(())
}
