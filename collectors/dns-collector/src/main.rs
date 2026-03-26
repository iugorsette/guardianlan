use anyhow::Result;

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt::init();
    dns_collector::run(dns_collector::Config::from_env()).await
}

