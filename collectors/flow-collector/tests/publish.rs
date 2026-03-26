use std::sync::{Arc, Mutex};

use anyhow::Result;
use async_trait::async_trait;
use common::{Publisher, FLOW_OBSERVED_SUBJECT};
use flow_collector::run_once;
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
async fn publishes_flow_fixture_entries() {
    let file = NamedTempFile::new().unwrap();
    std::fs::write(
        file.path(),
        r#"[{"device_id":"cam","src_ip":"192.168.1.20","dst_ip":"203.0.113.10","dst_port":554,"protocol":"tcp","bytes_in":100,"bytes_out":200,"observed_at":"2026-03-26T18:00:00Z"}]"#,
    )
    .unwrap();

    let subjects = Arc::new(Mutex::new(Vec::new()));
    let publisher = MockPublisher {
        subjects: subjects.clone(),
    };

    let published = run_once(&file.path().to_path_buf(), &publisher).await.unwrap();

    assert_eq!(published, 1);
    assert_eq!(subjects.lock().unwrap().as_slice(), &[FLOW_OBSERVED_SUBJECT.to_string()]);
}

