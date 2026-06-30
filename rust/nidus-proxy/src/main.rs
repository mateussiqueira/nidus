use hyper::body::Incoming;
use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper::{Request, Response, StatusCode};
use hyper_util::rt::TokioIo;
use http_body_util::Full;
use bytes::Bytes;
use std::collections::HashMap;
use std::net::SocketAddr;
use std::sync::Arc;
use tokio::net::TcpListener;
use tokio::sync::RwLock;
use tokio_postgres::{Client, NoTls};
use tracing::{info, warn, error};
use nidus_mesh::project_service_client::ProjectServiceClient;
use nidus_mesh::ResolveSlugRequest;

struct ProxyState {
    db: Client,
    cache: RwLock<HashMap<String, u16>>,
    dashboard_port: u16,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    tracing_subscriber::fmt()
        .with_env_filter("nidus_proxy=info")
        .init();

    info!("🦀 Nidus Proxy v0.2.0 starting...");

    let db_url = std::env::var("DATABASE_URL")
        .unwrap_or_else(|_| "host=localhost user=nidus password=nidus_dev_2026 dbname=nidus".into());

    let (client, conn) = tokio_postgres::connect(&db_url, NoTls).await?;
    tokio::spawn(async move {
        if let Err(e) = conn.await {
            error!("Database connection error: {}", e);
        }
    });

    let state = Arc::new(ProxyState {
        db: client,
        cache: RwLock::new(HashMap::new()),
        dashboard_port: 3000,
    });

    warm_cache(&state).await;

    let addr: SocketAddr = "0.0.0.0:8081".parse()?;
    let listener = TcpListener::bind(addr).await?;
    info!("Listening on {} (Rust/hyper)", addr);
    info!("Dashboard: localhost:{}, Cache size: {}", state.dashboard_port, state.cache.read().await.len());

    loop {
        let (stream, peer) = listener.accept().await?;
        let state = state.clone();
        tokio::spawn(async move {
            let io = TokioIo::new(stream);
            if let Err(e) = http1::Builder::new()
                .serve_connection(io, service_fn(move |req| handle(state.clone(), req)))
                .await
            {
                warn!("Connection error from {}: {}", peer, e);
            }
        });
    }
}

async fn resolve_via_grpc(slug: &str) -> Option<u16> {
    let mut client = ProjectServiceClient::connect("http://127.0.0.1:3001").await.ok()?;
    let req = tonic::Request::new(ResolveSlugRequest { slug: slug.to_string() });
    let resp = client.resolve_slug(req).await.ok()?;
    Some(resp.into_inner().port as u16)
}

async fn warm_cache(state: &Arc<ProxyState>) {
    let rows = state.db.query(
        "SELECT slug, port FROM projects WHERE status = $1 AND port > 0",
        &[&"ACTIVE"]
    ).await;

    match rows {
        Ok(rows) => {
            let mut cache = state.cache.write().await;
            for row in rows {
                let slug: String = row.get(0);
                let port: i32 = row.get(1);
                if port > 0 {
                    cache.insert(slug, port as u16);
                }
            }
            info!("Cache warmed: {} routes", cache.len());
        }
        Err(e) => warn!("Cache warm failed: {}", e),
    }
}

async fn handle(state: Arc<ProxyState>, req: Request<Incoming>) -> Result<Response<Full<Bytes>>, hyper::Error> {
    let host = req.headers()
        .get("host")
        .and_then(|h| h.to_str().ok())
        .unwrap_or("unknown");

    let path = req.uri().path();

    if path == "/health" {
        let cache_size = state.cache.read().await.len();
        let body = format!(r#"{{"status":"ok","proxy":"rust-hyper","version":"0.2.0","cache_routes":{cache_size}}}"#);
        return Ok(Response::new(Full::new(Bytes::from(body))));
    }

    if path == "/metrics" {
        let body = format!("# HELP nidus_proxy_cache_routes Total cached routes\n# TYPE nidus_proxy_cache_routes gauge\nnidus_proxy_cache_routes {}\n",
            state.cache.read().await.len());
        return Ok(Response::new(Full::new(Bytes::from(body))));
    }

    let slug = host.split('.').next().unwrap_or("unknown");

    let system: &[&str] = &["app", "api", "docs", "metrics", "nidus", "localhost", "127"];
    if system.contains(&slug) || !host.contains('.') {
        return forward(req, state.dashboard_port).await;
    }

    {
        let cache = state.cache.read().await;
        if let Some(port) = cache.get(slug) {
            return forward_to_port(req, *port).await;
        }
    }

    let row = state.db.query_one(
        "SELECT port FROM projects WHERE slug = $1 AND status = ACTIVE AND port > 0 LIMIT 1",
        &[&slug]
    ).await;

    match row {
        Ok(r) => {
            let port: i32 = r.get(0);
            if port > 0 {
                let mut cache = state.cache.write().await;
                cache.insert(slug.to_string(), port as u16);
                return forward_to_port(req, port as u16).await;
            }
        }
        Err(_) => {}
    }

    forward(req, state.dashboard_port).await
}

async fn forward(req: Request<Incoming>, port: u16) -> Result<Response<Full<Bytes>>, hyper::Error> {
    forward_to_port(req, port).await
}

async fn forward_to_port(req: Request<Incoming>, port: u16) -> Result<Response<Full<Bytes>>, hyper::Error> {
    let client = reqwest::Client::new();
    let path = req.uri().path_and_query()
        .map(|p| p.as_str())
        .unwrap_or("/");
    let uri = format!("http://127.0.0.1:{}{}", port, path);

    match client.get(&uri).timeout(std::time::Duration::from_secs(30)).send().await {
        Ok(resp) => {
            let status = resp.status();
            let body = resp.bytes().await.unwrap_or_default();
            let mut response = Response::new(Full::new(body));
            *response.status_mut() = status;
            Ok(response)
        }
        Err(e) => {
            warn!("Upstream {} error: {}", uri, e);
            let mut resp = Response::new(Full::new(Bytes::from("Bad Gateway")));
            *resp.status_mut() = StatusCode::BAD_GATEWAY;
            Ok(resp)
        }
    }
}
