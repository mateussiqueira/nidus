use dashmap::DashMap;
use std::time::{Duration, Instant};

pub struct RateLimiter {
    requests: DashMap<String, Vec<Instant>>,
}

impl RateLimiter {
    pub fn new() -> Self {
        Self {
            requests: DashMap::new(),
        }
    }

    pub async fn check(&self, key: &str, max_requests: u64, window: Duration) -> bool {
        let now = Instant::now();
        let cutoff = now - window;

        let mut entry = self.requests.entry(key.to_string()).or_insert_with(Vec::new);
        entry.retain(|t| *t > cutoff);

        if entry.len() >= max_requests as usize {
            return false;
        }

        entry.push(now);
        true
    }

    pub async fn stats(&self) -> serde_json::Value {
        let total = self.requests.len();
        let blocked = self.requests.iter()
            .filter(|e| e.value().len() >= 100)
            .count();

        serde_json::json!({
            "total_tracked": total,
            "blocked_ips": blocked,
        })
    }

    pub async fn cleanup(&self) {
        let cutoff = Instant::now() - Duration::from_secs(120);
        self.requests.retain(|_, times| {
            times.retain(|t| *t > cutoff);
            !times.is_empty()
        });
    }
}
