use axum::{
    body::Body,
    extract::{
        ConnectInfo,
        Request,
        State,
        WebSocketUpgrade,
        ws::{Message, WebSocket},
    },
    http::StatusCode,
    response::{IntoResponse, Response},
    routing::get,
    Router,
};
use bytes::Bytes;
use futures_util::{SinkExt, StreamExt};
use std::{
    net::SocketAddr,
    sync::Arc,
    time::{Duration, Instant},
};
use tower_http::{cors::CorsLayer, compression::CompressionLayer, trace::TraceLayer};
use tracing::{info, warn, error};

mod rate_limiter;
mod container_router;

use rate_limiter::RateLimiter;
use container_router::ContainerRouter;

#[derive(Clone)]
pub struct AppState {
    pub rate_limiter: Arc<RateLimiter>,
    pub container_router: Arc<ContainerRouter>,
    pub client: reqwest::Client,
    pub start_time: Instant,
}

#[tokio::main]
async fn main() {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| "stackrun_proxy=info,tower_http=info".into()),
        )
        .init();

    let listen_addr = std::env::var("PROXY_PORT").unwrap_or_else(|_| "3080".into());

    let state = AppState {
        rate_limiter: Arc::new(RateLimiter::new()),
        container_router: Arc::new(ContainerRouter::new()),
        client: reqwest::Client::builder()
            .timeout(Duration::from_secs(30))
            .pool_max_idle_per_host(32)
            .build()
            .unwrap(),
        start_time: Instant::now(),
    };

    // Spawn periodic cleanup tasks
    let container_router = state.container_router.clone();
    tokio::spawn(async move {
        let mut interval = tokio::time::interval(Duration::from_secs(30));
        loop {
            interval.tick().await;
            container_router.refresh().await;
        }
    });

    let rate_limiter = state.rate_limiter.clone();
    tokio::spawn(async move {
        let mut interval = tokio::time::interval(Duration::from_secs(60));
        loop {
            interval.tick().await;
            rate_limiter.cleanup().await;
        }
    });

    let app = Router::new()
        .route("/health", get(health))
        .route("/proxy/metrics", get(proxy_metrics))
        .route("/ws", get(ws_handler))
        .fallback(proxy_handler)
        .layer(CorsLayer::permissive())
        .layer(CompressionLayer::new())
        .layer(TraceLayer::new_for_http())
        .with_state(state);

    let addr: SocketAddr = format!("0.0.0.0:{}", listen_addr).parse().unwrap();
    info!("StackRun Proxy starting on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app.into_make_service_with_connect_info::<SocketAddr>())
        .await
        .unwrap();
}

async fn health() -> impl IntoResponse {
    axum::Json(serde_json::json!({
        "status": "ok",
        "name": "stackrun-proxy",
        "version": "0.2.0",
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

async fn ws_handler(
    State(state): State<AppState>,
    ws: WebSocketUpgrade,
    ConnectInfo(addr): ConnectInfo<SocketAddr>,
    request: Request,
) -> impl IntoResponse {
    let host = request.headers()
        .get("host")
        .and_then(|h| h.to_str().ok())
        .unwrap_or("")
        .to_string();

    ws.on_upgrade(move |socket| handle_websocket(socket, state, addr, host))
}

async fn handle_websocket(
    mut client_ws: WebSocket,
    state: AppState,
    addr: SocketAddr,
    host: String,
) {

    let client_ip = addr.ip().to_string();

    // Rate limit check
    if !state.rate_limiter.check(&client_ip, 100, Duration::from_secs(60)).await {
        warn!("Rate limit exceeded for WebSocket {}", client_ip);
        let _ = client_ws.send(Message::Text("Rate limit exceeded".into())).await;
        return;
    }

    // Resolve container for host
    let container_info = match state.container_router.resolve(&host).await {
        Some(info) => info,
        None => {
            warn!("No container found for WebSocket host: {}", host);
            let _ = client_ws.send(Message::Text("Container not found".into())).await;
            return;
        }
    };

    // Build upstream WebSocket URL
    let upstream_ws_url = format!(
        "ws://{}:{}",
        container_info.host,
        container_info.port
    );

    info!("WebSocket proxy: {} -> {} ({})", client_ip, upstream_ws_url, container_info.container_name);

    // Connect to upstream WebSocket
    let upstream_ws = match tokio_tungstenite::connect_async(&upstream_ws_url).await {
        Ok((ws, _)) => ws,
        Err(e) => {
            error!("Failed to connect to upstream WebSocket: {}", e);
            let _ = client_ws.send(Message::Text(format!("Upstream error: {}", e).into())).await;
            return;
        }
    };

    let (mut upstream_write, mut upstream_read) = upstream_ws.split();
    let (mut client_write, mut client_read) = client_ws.split();

    // Forward messages from client to upstream
    let client_to_upstream = async {
        while let Some(msg) = client_read.next().await {
            match msg {
                Ok(Message::Text(text)) => {
                    if upstream_write.send(tokio_tungstenite::tungstenite::Message::Text(text.to_string())).await.is_err() {
                        break;
                    }
                }
                Ok(Message::Binary(data)) => {
                    if upstream_write.send(tokio_tungstenite::tungstenite::Message::Binary(data.to_vec())).await.is_err() {
                        break;
                    }
                }
                Ok(Message::Ping(data)) => {
                    if upstream_write.send(tokio_tungstenite::tungstenite::Message::Ping(data.to_vec())).await.is_err() {
                        break;
                    }
                }
                Ok(Message::Pong(data)) => {
                    if upstream_write.send(tokio_tungstenite::tungstenite::Message::Pong(data.to_vec())).await.is_err() {
                        break;
                    }
                }
                Ok(Message::Close(_)) => {
                    let _ = upstream_write.send(tokio_tungstenite::tungstenite::Message::Close(None)).await;
                    break;
                }
                _ => {}
            }
        }
    };

    // Forward messages from upstream to client
    let upstream_to_client = async {
        while let Some(msg) = upstream_read.next().await {
            match msg {
                Ok(tokio_tungstenite::tungstenite::Message::Text(text)) => {
                    if client_write.send(Message::Text(text.into())).await.is_err() {
                        break;
                    }
                }
                Ok(tokio_tungstenite::tungstenite::Message::Binary(data)) => {
                    if client_write.send(Message::Binary(Bytes::from(data))).await.is_err() {
                        break;
                    }
                }
                Ok(tokio_tungstenite::tungstenite::Message::Ping(data)) => {
                    if client_write.send(Message::Ping(Bytes::from(data))).await.is_err() {
                        break;
                    }
                }
                Ok(tokio_tungstenite::tungstenite::Message::Pong(data)) => {
                    if client_write.send(Message::Pong(Bytes::from(data))).await.is_err() {
                        break;
                    }
                }
                Ok(tokio_tungstenite::tungstenite::Message::Close(_)) => {
                    let _ = client_write.send(Message::Close(None)).await;
                    break;
                }
                _ => {}
            }
        }
    };

    tokio::select! {
        _ = client_to_upstream => {},
        _ = upstream_to_client => {},
    }
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

    // Decompose the incoming request into parts
    let (parts, body) = req.into_parts();

    // Collect the body bytes
    let body_bytes = match axum::body::to_bytes(body, usize::MAX).await {
        Ok(bytes) => bytes,
        Err(e) => {
            error!("Failed to read request body: {}", e);
            return Err(StatusCode::INTERNAL_SERVER_ERROR);
        }
    };

    // Build reqwest request
    let method = parts.method;
    let mut upstream_req = state.client.request(method.clone(), &upstream_url);

    // Copy headers (except host)
    for (key, value) in parts.headers.iter() {
        if key != "host" {
            upstream_req = upstream_req.header(key, value);
        }
    }

    // Add forwarded headers
    upstream_req = upstream_req
        .header("X-Forwarded-For", &client_ip)
        .header("X-Forwarded-Proto", "http")
        .header("X-Forwarded-Host", &host)
        .header("X-Real-IP", &client_ip)
        .header("Host", &host);

    // Add body
    if !body_bytes.is_empty() {
        upstream_req = upstream_req.body(body_bytes);
    }

    // Forward request to upstream
    match upstream_req.send().await {
        Ok(upstream_response) => {
            info!("Proxied {} to {} ({})", method, container_info.container_name, upstream_url);

            // Convert reqwest response to axum response
            let status = upstream_response.status();
            let headers = upstream_response.headers().clone();

            // Read response body
            let response_body = match upstream_response.bytes().await {
                Ok(bytes) => bytes,
                Err(e) => {
                    error!("Failed to read upstream response: {}", e);
                    return Err(StatusCode::BAD_GATEWAY);
                }
            };

            // Build axum response
            let mut response = Response::builder().status(status.as_u16());

            // Copy response headers
            for (key, value) in headers.iter() {
                response = response.header(key, value);
            }

            match response.body(Body::from(response_body)) {
                Ok(resp) => Ok(resp),
                Err(e) => {
                    error!("Failed to build response: {}", e);
                    Err(StatusCode::INTERNAL_SERVER_ERROR)
                }
            }
        }
        Err(e) => {
            error!("Failed to proxy request to {}: {}", upstream_url, e);
            Err(StatusCode::BAD_GATEWAY)
        }
    }
}
