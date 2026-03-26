use std::sync::{Arc, Mutex};

use anyhow::Result;
use async_trait::async_trait;
use common::{Publisher, DEVICE_DISCOVERED_SUBJECT};
use discovery_collector::run_once;
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
async fn publishes_fixture_entries_as_discovered_events() {
    let file = NamedTempFile::new().unwrap();
    std::fs::write(
        file.path(),
        r#"[{"id":"device-1","mac":"AA:BB","ips":["192.168.1.10"],"hostname":"cam","vendor":"Generic","device_type":"camera","profile_id":"iot","managed":false,"observed_at":"2026-03-26T18:00:00Z"}]"#,
    )
    .unwrap();

    let subjects = Arc::new(Mutex::new(Vec::new()));
    let publisher = MockPublisher {
        subjects: subjects.clone(),
    };

    let published = run_once(&file.path().to_path_buf(), &publisher, DEVICE_DISCOVERED_SUBJECT)
        .await
        .unwrap();

    assert_eq!(published, 1);
    assert_eq!(subjects.lock().unwrap().as_slice(), &[DEVICE_DISCOVERED_SUBJECT.to_string()]);
}

