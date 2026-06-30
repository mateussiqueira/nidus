use hyper::body::Incoming;
use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper::{Request, Response};
use hyper_util::rt::TokioIo;
use http_body_util::Full;
use bytes::Bytes;
use std::net::SocketAddr;
use tokio::net::TcpListener;
use tracing::{info, warn};

mod proxy;
mod router;
mod metrics;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    tracing_subscriber::fmt::init();

    metrics::init();

    let addr: SocketAddr = "0.0.0.0:8080".parse()?;
    let listener = TcpListener::bind(addr).await?;
    info!("Nidus Proxy v2.0 (Rust) listening on {}", addr);

    loop {
        let (stream, peer) = listener.accept().await?;
        tokio::spawn(async move {
            let io = TokioIo::new(stream);
            if let Err(e) = http1::Builder::new()
                .serve_connection(io, service_fn(|req| handle(req)))
                .await
            {
                warn!("Connection error from {}: {}", peer, e);
            }
        });
    }
}

async fn handle(req: Request<Incoming>) -> Result<Response<Full<Bytes>>, hyper::Error> {
    let _host = req
        .headers()
        .get("host")
        .and_then(|h| h.to_str().ok())
        .unwrap_or("unknown");

    // Health check
    if req.uri().path() == "/health" {
        return Ok(Response::new(Full::new(Bytes::from(
            r#"{"status":"ok","proxy":"rust","version":"0.1.0"}"#,
        ))));
    }

    // Phase 1: forward everything to dashboard on port 3000
    // Phase 2: DB-backed per-project routing
    proxy::forward(req, 3000).await
}
