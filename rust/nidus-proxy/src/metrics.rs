use std::sync::atomic::{AtomicU64, Ordering};

#[allow(dead_code)]
static REQUESTS: AtomicU64 = AtomicU64::new(0);

pub fn init() {}

#[allow(dead_code)]
pub fn metrics() -> String {
    let requests = REQUESTS.load(Ordering::Relaxed);
    format!(
        "# HELP nidus_proxy_requests_total Total requests\n\
         # TYPE nidus_proxy_requests_total counter\n\
         nidus_proxy_requests_total {}\n",
        requests
    )
}
