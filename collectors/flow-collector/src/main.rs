use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt::init();
    flow_collector::run(flow_collector::Config::from_env()).await
}

