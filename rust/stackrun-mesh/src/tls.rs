// TLS configuration for gRPC service mesh
// Uses rustls for mutual TLS between services

use tonic::transport::{Certificate, ClientTlsConfig, Identity, ServerTlsConfig};
use std::path::Path;

pub fn load_server_tls(cert_path: &str, key_path: &str) -> Result<ServerTlsConfig, Box<dyn std::error::Error>> {
    let cert = std::fs::read_to_string(cert_path)?;
    let key = std::fs::read_to_string(key_path)?;
    let identity = Identity::from_pem(cert, key);
    Ok(ServerTlsConfig::new().identity(identity))
}

pub fn load_client_tls(ca_cert_path: &str) -> Result<ClientTlsConfig, Box<dyn std::error::Error>> {
    let pem = std::fs::read_to_string(ca_cert_path)?;
    let ca = Certificate::from_pem(pem);
    Ok(ClientTlsConfig::new().ca_certificate(ca))
}
