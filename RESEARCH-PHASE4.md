# StackRun Phase 4 — Service Mesh, Zero-Downtime, Security

## 1. gRPC Service Mesh — Arquitetura

### Por que gRPC e não REST?
```
REST/JSON:   HTTP/1.1, texto, ~500B overhead por request
gRPC/protobuf: HTTP/2, binário, ~20B overhead por request
gRPC streaming: Server push, bidirectional, flow control
```
**Ganho**: 25x menos overhead, streaming nativo, contratos tipados.

### Mesh topology
```
┌──────────────────────────────────────────────────┐
│                 stackrun-edge :8085                   │
│  ┌────────────────────────────────────────────┐   │
│  │ gRPC client → stackrun-proxy (resolve slug)   │   │
│  │ gRPC client → stackrun-builder (trigger build) │   │
│  └────────────────────────────────────────────┘   │
├──────────────────────────────────────────────────┤
│                 stackrun-proxy :8080                  │
│  ┌────────────────────────────────────────────┐   │
│  │ gRPC server ← edge (slug resolution)       │   │
│  │ gRPC server ← builder (port registration)  │   │
│  │ gRPC client → stackrun-api (project lookup)   │   │
│  └────────────────────────────────────────────┘   │
├──────────────────────────────────────────────────┤
│                 stackrun-builder :8083                │
│  ┌────────────────────────────────────────────┐   │
│  │ gRPC server ← edge (trigger builds)        │   │
│  │ gRPC client → stackrun-proxy (register port)  │   │
│  │ gRPC client → stackrun-api (update deploy)    │   │
│  └────────────────────────────────────────────┘   │
├──────────────────────────────────────────────────┤
│                 stackrun-api (Go) :3001               │
│  ┌────────────────────────────────────────────┐   │
│  │ gRPC server (tonic-reflection)             │   │
│  │ └─ ProjectService, DeployService, etc      │   │
│  └────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────┘
```

### Protobuf service definitions
```protobuf
service ProjectService {
  rpc GetProject(GetProjectRequest) returns (Project);
  rpc ListProjects(ListProjectsRequest) returns (ListProjectsResponse);
  rpc ResolveSlug(ResolveSlugRequest) returns (ResolveSlugResponse);
}

service DeployService {
  rpc TriggerBuild(TriggerBuildRequest) returns (TriggerBuildResponse);
  rpc StreamBuildLogs(BuildLogsRequest) returns (stream BuildLogEntry);
  rpc RegisterPort(RegisterPortRequest) returns (RegisterPortResponse);
}

service HealthService {
  rpc Check(HealthCheckRequest) returns (HealthCheckResponse);
  rpc Watch(HealthWatchRequest) returns (stream HealthStatus);
}
```

## 2. Zero-Downtime Deploys

### Blue/Green Strategy
```
┌─────────────────────────────────────────────┐
│                 Load Balancer                │
│  ┌───────────────────────────────────────┐  │
│  │ Health check: GET /health             │  │
│  │ Routing: blue (active) / green (new)  │  │
│  └───────────────────────────────────────┘  │
├─────────────────────────────────────────────┤
│  Blue (v1.0.0)          Green (v1.1.0)      │
│  ┌─────────────────┐   ┌─────────────────┐  │
│  │ Port: 8095       │   │ Port: 8096       │  │
│  │ Status: ACTIVE   │   │ Status: WARMING  │  │
│  │ Traffic: 100%    │   │ Traffic: 0%      │  │
│  └─────────────────┘   └─────────────────┘  │
├─────────────────────────────────────────────┤
│  Deploy Flow:                               │
│  1. Build new image                         │
│  2. Start green container (port 8096)       │
│  3. Health check green (10 attempts)        │
│  4. If healthy: switch traffic to green     │
│  5. Wait 30s (drain connections)            │
│  6. Stop blue container                     │
│  7. If unhealthy: keep blue, alert          │
└─────────────────────────────────────────────┘
```

### Canary Deploy (advanced)
```
Traffic split over time:
t=0:   Green 0%,   Blue 100%
t=30:  Green 5%,   Blue 95%   (monitor errors)
t=60:  Green 25%,  Blue 75%   (monitor latency)
t=90:  Green 50%,  Blue 50%   (monitor full metrics)
t=120: Green 100%, Blue 0%    (complete, drain blue)
IF errors > 1%: auto-rollback to Blue 100%
```

### Health Gating
```
Readiness probe: GET /health → 200 (ready for traffic)
Liveness probe:  GET /health → 200 (still alive)
Startup probe:   GET /health → 200 (initialized)

Gate conditions:
- readiness: 3 consecutive 200s, wait 2s between
- liveness: 1 failure → restart after 10s
- startup: max 60s to pass, else fail deploy
```

## 3. Security Hardening

### Secrets Management
```
┌──────────────────────────────────────────┐
│            HashiCorp Vault (ou SOPS)      │
│  ┌────────────────────────────────────┐  │
│  │ Transit Engine: encrypt/decrypt    │  │
│  │ KV v2: secrets at rest             │  │
│  │ Dynamic DB creds: TTL 1h           │  │
│  └────────────────────────────────────┘  │
├──────────────────────────────────────────┤
│  Integration:                             │
│  - stackrun-api: vault client (transit)      │
│  - env vars: vault://secret/data/nidus    │
│  - DB password: dynamic, rotated hourly   │
└──────────────────────────────────────────┘
```

### SBOM (Software Bill of Materials)
```bash
cargo cyclonedx    # Generate CycloneDX SBOM
cargo audit        # Check for known vulnerabilities
cargo deny check   # License compliance
trivy fs .         # Filesystem vulnerability scan
trivy image nidus  # Container image scan
```

### TLS Everywhere
```
- Rust services: tonic + rustls (mTLS)
- Go ↔ Rust: gRPC over TLS 1.3
- PostgreSQL: sslmode=verify-full
- Redis: TLS + ACL
- Caddy → backend: HTTPS (not HTTP)
```

### Audit Logging
```
Event schema:
{
  "timestamp": "2026-06-30T17:00:00Z",
  "actor": "user-id|service-name",
  "action": "deploy.start|deploy.complete|secret.access",
  "resource": "project:slug|secret:name",
  "result": "success|failure",
  "metadata": { "branch": "main", "commit": "abc123" }
}
```

## 4. Rust crate: stackrun-mesh

```rust
// Shared protobuf definitions for inter-service communication
// Compiled to Rust types via tonic-build
```

