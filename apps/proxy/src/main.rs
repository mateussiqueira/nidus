use axum::{
    extract::{ConnectInfo, Request, State},
    http::StatusCode,
    response::{IntoResponse, Response},
    routing::get,
    Router,
};
use hyper_util::{
    client::legacy::{connect::HttpConnector, Client},
    rt::TokioExecutor,
};
use std::{
    net::SocketAddr,
    sync::Arc,
    time::{Duration, Instant},
};
use tower_http::{cors::CorsLayer, trace::TraceLayer};
use tracing::{info, warn};

mod rate_limiter;
mod container_router;

use rate_limiter::RateLimiter;
use container_router::ContainerRouter;

#[derive(Clone)]
pub struct AppState {
    pub rate_limiter: Arc<RateLimiter>,
    pub container_router: Arc<ContainerRouter>,
    pub client: Client<HttpConnector, hyper::body::Incoming>,
    pub start_time: Instant,
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "nidus_proxy=info,tower_http=info".into()),
        )
        .init();

    let listen_addr = std::env::var("PROXY_PORT").unwrap_or_else(|_| "3080".into());

    let state = AppState {
        rate_limiter: Arc::new(RateLimiter::new()),
        container_router: Arc::new(ContainerRouter::new()),
        client: Client::builder(TokioExecutor::new())
            .pool_idle_timeout(Duration::from_secs(30))
            .build_http(),
        start_time: Instant::now(),
    };

    let app = Router::new()
        .route("/health", get(health))
        .route("/proxy/metrics", get(proxy_metrics))
        .fallback(proxy_handler)
        .layer(CorsLayer::permissive())
        .layer(TraceLayer::new_for_http())
        .with_state(state);

    let addr: SocketAddr = format!("0.0.0.0:{}", listen_addr).parse().unwrap();
    info!("Nidus Proxy starting on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app.into_make_service_with_connect_info::<SocketAddr>())
        .await
        .unwrap();
}

async fn health() -> impl IntoResponse {
    axum::Json(serde_json::json!({
        "status": "ok",
        "name": "nidus-proxy",
        "version": "0.1.0",
        "timestamp": chrono::Utc::now().to_rfc3339(),
    }))
}

async fn proxy_metrics(State(state): State<AppState>) -> impl IntoResponse {
    let uptime = state.start_time.elapsed().as_secs();
    let container_count = state.container_router.count().await;

    axum::Json(serde_json::json!({
        "uptime": uptime,
        "containers": container_count,
        "rate_limiters": state.rate_limiter.stats().await,
    }))
}

async fn proxy_handler(
    State(state): State<AppState>,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    req: Request,
) -> Result<Response, StatusCode> {
    let host = req.headers()
        .get("host")
        .and_then(|h| h.to_str().ok())
        .unwrap_or("")
        .to_string();

    let client_ip = addr.ip().to_string();

    // Rate limit check
    if !state.rate_limiter.check(&client_ip, 100, Duration::from_secs(60)).await {
        warn!("Rate limit exceeded for {}", client_ip);
        return Err(StatusCode::TOO_MANY_REQUESTS);
    }

    // Resolve container for host
    let container_info = match state.container_router.resolve(&host).await {
        Some(info) => info,
        None => {
            warn!("No container found for host: {}", host);
            return Err(StatusCode::NOT_FOUND);
        }
    };

    // Build upstream URL
    let path = req.uri().path_and_query()
        .map(|pq| pq.as_str())
        .unwrap_or("/");

    let upstream_url = format!("http://{}:{}{}", container_info.host, container_info.port, path);

    // Return proxy info
    let body = serde_json::json!({
        "proxy": true,
        "upstream": upstream_url,
        "container": container_info.container_name,
        "client_ip": client_ip,
    });

    Ok((
        StatusCode::OK,
        [("content-type", "application/json")],
        body.to_string(),
    ).into_response())
}
