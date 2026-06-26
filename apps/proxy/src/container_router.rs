use dashmap::DashMap;
use std::process::Command;
use tracing::{info, warn};

#[derive(Clone, Debug)]
pub struct ContainerInfo {
    pub host: String,
    pub port: u16,
    pub container_name: String,
}

pub struct ContainerRouter {
    routes: DashMap<String, ContainerInfo>,
}

impl ContainerRouter {
    pub fn new() -> Self {
        Self {
            routes: DashMap::new(),
        }
    }

    pub async fn resolve(&self, host: &str) -> Option<ContainerInfo> {
        // Check cache first
        if let Some(info) = self.routes.get(host) {
            return Some(info.clone());
        }

        // Try to discover container
        if let Some(info) = self.discover_container(host).await {
            self.routes.insert(host.to_string(), info.clone());
            Some(info)
        } else {
            None
        }
    }

    async fn discover_container(&self, host: &str) -> Option<ContainerInfo> {
        let slug = host
            .replace(".nidus.localhost", "")
            .replace(".nidus.com", "")
            .replace(".localhost", "")
            .replace(".local", "");

        if slug.is_empty() || slug == host {
            return None;
        }

        let container_name = format!("nidus-{}", slug);

        // Get container IP from Docker network
        let output = Command::new("docker")
            .args(["inspect", "-f", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", &container_name])
            .output()
            .ok()?;

        if !output.status.success() {
            return None;
        }

        let container_ip = String::from_utf8_lossy(&output.stdout).trim().to_string();
        if container_ip.is_empty() {
            return None;
        }

        // Get exposed port
        let output = Command::new("docker")
            .args(["port", &container_name])
            .output()
            .ok()?;

        if !output.status.success() {
            return None;
        }

        let port_str = String::from_utf8_lossy(&output.stdout);
        let port = port_str
            .lines()
            .next()?
            .split("->")
            .nth(1)?
            .split(':')
            .last()?
            .trim()
            .parse::<u16>()
            .ok()?;

        info!("Discovered container {} at {}:{}", container_name, container_ip, port);

        Some(ContainerInfo {
            host: container_ip,
            port,
            container_name,
        })
    }

    pub async fn count(&self) -> usize {
        self.routes.len()
    }

    pub async fn refresh(&self) {
        let output = match Command::new("docker")
            .args(["ps", "--format", "{{.Names}}\t{{.Ports}}"])
            .output()
        {
            Ok(o) => o,
            Err(_) => return,
        };

        if !output.status.success() {
            return;
        }

        let stdout = String::from_utf8_lossy(&output.stdout);
        for line in stdout.lines() {
            let parts: Vec<&str> = line.split('\t').collect();
            if parts.len() >= 2 {
                let name = parts[0].to_string();
                if name.starts_with("nidus-") {
                    let slug = name.strip_prefix("nidus-").unwrap_or("");

                    // Get container IP
                    if let Ok(ip_output) = Command::new("docker")
                        .args(["inspect", "-f", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}", &name])
                        .output()
                    {
                        let container_ip = String::from_utf8_lossy(&ip_output.stdout).trim().to_string();
                        if !container_ip.is_empty() {
                            if let Some(port) = parts[1]
                                .split("->")
                                .nth(1)
                                .and_then(|s| s.split(':').last())
                                .and_then(|s| s.trim().parse::<u16>().ok())
                            {
                                let host = format!("{}.nidus.localhost", slug);
                                self.routes.insert(host, ContainerInfo {
                                    host: container_ip,
                                    port,
                                    container_name: name,
                                });
                            }
                        }
                    }
                }
            }
        }
    }

    pub async fn remove(&self, host: &str) {
        self.routes.remove(host);
    }
}
