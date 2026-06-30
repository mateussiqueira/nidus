use hyper::body::Incoming;
use hyper::{Request, Response, StatusCode};
use http_body_util::Full;
use bytes::Bytes;

pub async fn forward(
    req: Request<Incoming>,
    port: u16,
) -> Result<Response<Full<Bytes>>, hyper::Error> {
    let client = reqwest::Client::new();
    let uri = format!(
        "http://127.0.0.1:{}{}",
        port,
        req.uri()
            .path_and_query()
            .map(|p| p.as_str())
            .unwrap_or("/")
    );

    match client.get(&uri).send().await {
        Ok(resp) => {
            let body = resp.bytes().await.unwrap_or_default();
            Ok(Response::new(Full::new(body)))
        }
        Err(_) => {
            let mut resp = Response::new(Full::new(Bytes::from("Bad Gateway")));
            *resp.status_mut() = StatusCode::BAD_GATEWAY;
            Ok(resp)
        }
    }
}
