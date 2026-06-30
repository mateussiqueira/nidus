use criterion::{black_box, criterion_group, criterion_main, Criterion};

fn bench_proxy_simple(c: &mut Criterion) {
    c.bench_function("proxy_forward_local", |b| {
        let rt = tokio::runtime::Runtime::new().unwrap();
        b.iter(|| {
            rt.block_on(async {
                let client = reqwest::Client::new();
                let _ = client.get("http://127.0.0.1:3000/health").send().await;
            });
        })
    });
}

fn bench_rust_vs_reqwest(c: &mut Criterion) {
    let mut group = c.benchmark_group("http_clients");
    let rt = tokio::runtime::Runtime::new().unwrap();

    group.bench_function("reqwest_keepalive", |b| {
        let client = reqwest::Client::new();
        b.iter(|| {
            rt.block_on(async {
                black_box(client.get("http://127.0.0.1:3000/health").send().await);
            });
        })
    });

    group.bench_function("hyper_direct", |b| {
        b.iter(|| {
            rt.block_on(async {
                let stream = tokio::net::TcpStream::connect("127.0.0.1:3000").await.unwrap();
                let io = hyper_util::rt::TokioIo::new(stream);
                let (mut sender, conn) = hyper::client::conn::http1::handshake(io).await.unwrap();
                tokio::spawn(conn);
                let req = hyper::Request::builder()
                    .uri("/health")
                    .body(http_body_util::Empty::<bytes::Bytes>::new()).unwrap();
                black_box(sender.send_request(req).await);
            });
        })
    });
}

criterion_group!(benches, bench_proxy_simple, bench_rust_vs_reqwest);
criterion_main!(benches);
