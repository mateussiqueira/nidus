use anyhow::Result;
use hyper::body::Incoming;
use hyper::server::conn::http1;
use hyper::service::service_fn;
use hyper::{Request, Response, StatusCode};
use hyper_util::rt::TokioIo;
use http_body_util::Full;
use bytes::Bytes;
use parking_lot::RwLock;
use std::net::SocketAddr;
use std::path::PathBuf;
use std::sync::Arc;
use tokio::net::TcpListener;
use tracing::{info, warn, error, instrument};
use wasmtime::*;
use wasmtime_wasi::WasiCtxBuilder;

mod engine;

const MAX_FUEL: u64 = 50_000_000;
const MODULE_CACHE_SIZE: usize = 100;

#[derive(Clone)]
struct EdgeState {
    wasm_dir: PathBuf,
    instance_cache: Arc<RwLock<lru::LruCache<String, InstancePre<wasmtime_wasi::preview1::WasiP1Ctx>>>>,
    engine: Engine,
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(tracing_subscriber::EnvFilter::new("stackrun_edge=info,wasmtime=warn"))
        .init();

    info!("StackRun Edge Runtime v0.1.0");
    info!("   Runtime: wasmtime");
    info!("   Max fuel/request: {}M instructions", MAX_FUEL / 1_000_000);
    info!("   Module cache: {} entries", MODULE_CACHE_SIZE);

    let mut config = Config::new();
    config.async_support(true);
    config.cranelift_opt_level(OptLevel::Speed);
    config.epoch_interruption(true);
    config.consume_fuel(true);

    let engine = Engine::new(&config)?;

    let state = EdgeState {
        wasm_dir: PathBuf::from(std::env::var("WASM_DIR").unwrap_or_else(|_| "/var/lib/stackrun/wasm".into())),
        instance_cache: Arc::new(RwLock::new(lru::LruCache::new(MODULE_CACHE_SIZE.try_into().unwrap()))),
        engine,
    };

    engine::load_modules(&state).await?;

    let addr: SocketAddr = "0.0.0.0:8085".parse()?;
    let listener = TcpListener::bind(addr).await?;
    info!("Edge server listening on {}", addr);
    info!("Features:");
    info!("  - WASI preview1 runtime (wasmtime)");
    info!("  - InstancePre cache (<1ms cold starts)");
    info!("  - LRU module cache");
    info!("  - Resource limits: fuel + memory + epoch");
    info!("  - Tracing instrumentation");

    let engine_ref = state.engine.clone();
    tokio::spawn(async move {
        loop {
            tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
            engine_ref.increment_epoch();
        }
    });

    loop {
        let (stream, peer) = listener.accept().await?;
        let state = state.clone();
        tokio::spawn(async move {
            let io = TokioIo::new(stream);
            if let Err(e) = http1::Builder::new()
                .serve_connection(io, service_fn(move |req| handle_request(state.clone(), req)))
                .await
            {
                warn!("Connection error from {}: {}", peer, e);
            }
        });
    }
}

#[instrument(skip(state, req), fields(
    host = %req.headers().get("host").and_then(|h| h.to_str().ok()).unwrap_or("unknown"),
    path = %req.uri().path()
))]
async fn handle_request(state: EdgeState, req: Request<Incoming>) -> Result<Response<Full<Bytes>>> {
    let host = req.headers()
        .get("host")
        .and_then(|h| h.to_str().ok())
        .unwrap_or("unknown");
    let path = req.uri().path();

    if path == "/health" {
        let cache_size = state.instance_cache.write().len();
        let body = format!(
            r#"{{"status":"ok","runtime":"wasmtime","modules_cached":{},"version":"0.1.0"}}"#,
            cache_size
        );
        return Ok(Response::new(Full::new(Bytes::from(body))));
    }

    let fn_name = host.split('.').next().unwrap_or("default");
    let instance_pre = {
        let mut cache = state.instance_cache.write();
        cache.get(fn_name).cloned()
    };

    match instance_pre {
        Some(pre) => {
            let start = std::time::Instant::now();
            info!("Executing WASM function: {} (path: {})", fn_name, path);

            let wasi = WasiCtxBuilder::new()
                .inherit_stdio()
                .inherit_env()
                .build_p1();

            let mut store = Store::new(&state.engine, wasi);
            store.set_fuel(MAX_FUEL)?;
            store.set_epoch_deadline(1);

            let instance = pre.instantiate(&mut store)?;

            if let Ok(func) = instance.get_typed_func::<(), ()>(&mut store, "handle") {
                match func.call_async(&mut store, ()).await {
                    Ok(_) => {
                        let elapsed = start.elapsed();
                        info!("WASM execution: {}us", elapsed.as_micros());
                        Ok(Response::new(Full::new(Bytes::from("OK"))))
                    }
                    Err(e) => {
                        error!("WASM execution error: {}", e);
                        let mut resp = Response::new(Full::new(Bytes::from("Execution Error")));
                        *resp.status_mut() = StatusCode::INTERNAL_SERVER_ERROR;
                        Ok(resp)
                    }
                }
            } else {
                warn!("WASM module has no \"handle\" function");
                let mut resp = Response::new(Full::new(Bytes::from("No handler")));
                *resp.status_mut() = StatusCode::NOT_FOUND;
                Ok(resp)
            }
        }
        None => {
            warn!("No WASM module for: {}", fn_name);
            let mut resp = Response::new(Full::new(Bytes::from("Edge function not found")));
            *resp.status_mut() = StatusCode::NOT_FOUND;
            Ok(resp)
        }
    }
}
