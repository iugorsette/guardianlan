use std::sync::{Arc, Mutex};

use anyhow::Result;
use async_trait::async_trait;
use common::{Publisher, DNS_OBSERVED_SUBJECT};
use dns_collector::run_once;
use tempfile::NamedTempFile;

struct MockPublisher {
    subjects: Arc<Mutex<Vec<String>>>,
}

#[async_trait]
impl Publisher for MockPublisher {
    async fn publish(&self, subject: &str, _payload: Vec<u8>) -> Result<()> {
        self.subjects.lock().unwrap().push(subject.to_string());
        Ok(())
    }
}

#[tokio::test]
async fn publishes_dns_fixture_entries() {
    let file = NamedTempFile::new().unwrap();
    std::fs::write(
        file.path(),
        r#"[{"device_id":"tablet","query":"example.com","domain":"example.com","category":"general","resolver":"adguardhome","blocked":false,"observed_at":"2026-03-26T18:00:00Z"}]"#,
    )
    .unwrap();

    let subjects = Arc::new(Mutex::new(Vec::new()));
    let publisher = MockPublisher {
        subjects: subjects.clone(),
    };

    let published = run_once(&file.path().to_path_buf(), &publisher).await.unwrap();

    assert_eq!(published, 1);
    assert_eq!(subjects.lock().unwrap().as_slice(), &[DNS_OBSERVED_SUBJECT.to_string()]);
}

