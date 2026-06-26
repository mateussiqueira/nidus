use dashmap::DashMap;
use std::process::Command;

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
        if let Some(info) = self.routes.get(host) {
            return Some(info.clone());
        }

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
            .replace(".localhost", "")
            .replace(".local", "");

        if slug.is_empty() || slug == host {
            return None;
        }

        let container_name = format!("nidus-{}", slug);

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

        Some(ContainerInfo {
            host: "127.0.0.1".to_string(),
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
                    let host = format!("{}.nidus.localhost", slug);

                    if let Some(port) = parts[1]
                        .split("->")
                        .nth(1)
                        .and_then(|s| s.split(':').last())
                        .and_then(|s| s.trim().parse::<u16>().ok())
                    {
                        self.routes.insert(host, ContainerInfo {
                            host: "127.0.0.1".to_string(),
                            port,
                            container_name: name,
                        });
                    }
                }
            }
        }
    }
}
