# 🚀 Roadmap de Otimização - Nidus

**Status:** Active Development  
**Última Atualização:** 28 de junho de 2026  
**Target:** 512MB (Lite) até 2GB (Completa) | Compétir com Vercel/Commit.com

---

## 📊 Visão Geral - Arquitetura Atual

```
┌─────────────────────────────────────────────────────────────┐
│                   CADDY (10-50MB)                           │
│              HTTPS automático + Load Balance                │
└──────────────────────┬────────────────────────────────────┘
                       │
        ┌──────────────┼──────────────┬────────────────┐
        ▼              ▼              ▼                ▼
   Dashboard       API Go/NestJS   Proxy Rust    Deploy Worker
   Next.js         30-50MB         10-50MB       Go 20-50MB
   150-300MB                                     
        │              │              │                │
        └──────────────┼──────────────┴────────────────┘
                       │
     ┌─────────────────┼──────────────┐
     ▼                 ▼              ▼
PostgreSQL 16      Redis 7         Worker Pool
200-500MB          256MB           (Docker SDK)
```

**Consumo Total Estimado:**
- 🟢 **Lite (512MB):** API Go (64MB) + Worker Go (64MB) + SQLite (5MB) + Sistema (379MB)
- 🟡 **Médio (1.5GB):** PostgreSQL (200MB) + Redis (256MB) + API (30MB) + Worker (50MB) + Dashboard (150MB) + Caddy (20MB) + Buffer
- 🔴 **Completo (2GB+):** Acima + Margem para picos e build paralelo

---

## 🎯 Fases de Desenvolvimento

### **FASE 1: Deploy Worker Go [ATUAL] ✅**

**Duração:** Semanas 1-2  
**Status:** Em Implementação  
**RAM Target:** -80% vs Node.js (100MB → 20MB)

#### 1.1 Docker SDK Nativo (Semana 1)
- [x] Implementar docker-go SDK (não CLI exec)
- [x] Streaming de logs em tempo real
- [x] Timeout handling e retry logic
- [ ] BuildKit support (parallelização)
- [ ] Image layer caching (Nidus-side)

**Ficheiros:**
- `workers/deploy/main.go` - Core worker
- `workers/deploy/Dockerfile` - Build otimizado (multi-stage)
- `workers/deploy/go.mod` - Dependências (minimizar)

**Benchmark Esperado:**
```
Métrica              Node.js    Go        Melhoria
─────────────────────────────────────────────────
Git clone            ~5s        ~0.5s     ⚡ 10x
Docker build         ~30s       ~25s      ⚡ 1.2x
Memory idle          ~100MB     ~15MB     ⚡ 6.7x
Memory building      ~300MB     ~50MB     ⚡ 6x
Startup time         ~2s        ~0.2s     ⚡ 10x
```

#### 1.2 Concorrência de Build (Semana 2)
- [ ] Implementar job queue (BullMQ → native channels)
- [ ] WORKER_CONCURRENCY = 10 (padrão)
- [ ] Suportar até 50 builds simultâneos (em 2GB)
- [ ] Graceful shutdown + job persistence

**Código:**
```go
// workers/deploy/main.go
const MaxConcurrentBuilds = 50

func (w *Worker) Start(ctx context.Context) {
    semaphore := make(chan struct{}, MaxConcurrentBuilds)
    for job := range w.queue {
        semaphore <- struct{}{}        // Acquire
        go func() {
            defer func() { <-semaphore }() // Release
            w.processBuild(ctx, job)
        }()
    }
}
```

**Arquivo de config:**
```toml
# workers/deploy/.env.production
MAX_CONCURRENT_BUILDS=50
DOCKER_TIMEOUT_SECONDS=120
LOG_BUFFER_SIZE=8192
MEMORY_LIMIT_MB=256
```

---

### **FASE 2: Control Plane - Migração Go [PRÓXIMO] ⏳**

**Duração:** Semanas 3-5  
**Status:** Planejamento  
**RAM Target:** -70% vs NestJS (150MB → 45MB)

#### 2.1 Framework & Setup (Semana 3)
- [ ] Avaliar frameworks Go: **Fiber** (velocidade) vs **Chi** (padrão)
- [ ] Estrutura de projeto: `cmd/api`, `internal/handlers`, `internal/services`
- [ ] Database driver: `pgx/v5` (performance, prepared statements)
- [ ] Dependency injection: `wire` (compile-time segurança)

**Recomendação:** Fiber para máxima performance

```
go-fiber/fiber/v3          → 50-80x mais rápido que Node.js
lib/pq                     → Prepared statements
google/wire                → DI compile-time
uber/zap                   → Logging de alta performance
prometheus/client_golang   → Métricas
```

#### 2.2 Endpoints & Services (Semana 4)
Migrar 20 endpoints de `apps/control-plane/src`:

| Módulo | Endpoints | Prioridade | RAM Economia |
|--------|-----------|-----------|-------------|
| **auth** | POST /login, POST /register, POST /verify | 🔴 Alta | -15% |
| **projects** | CRUD projects + deployments | 🔴 Alta | -20% |
| **deployments** | List, create, logs, stream | 🟡 Média | -15% |
| **webhook** | GitHub webhooks, trigger | 🟡 Média | -10% |
| **health** | GET /health, readiness | 🟢 Baixa | N/A |

**Estrutura:**
```
apps/api-go/
├── cmd/main.go                    # Entrypoint
├── internal/
│   ├── handlers/
│   │   ├── auth.go
│   │   ├── projects.go
│   │   ├── deployments.go
│   │   └── webhook.go
│   ├── services/
│   │   ├── auth_service.go
│   │   ├── project_service.go
│   │   └── deploy_service.go
│   ├── database/
│   │   ├── postgres.go
│   │   └── migrations/
│   └── middleware/
│       ├── auth.go
│       ├── logger.go
│       └── rate_limit.go
├── Dockerfile                     # Multi-stage build
├── go.mod
└── go.sum
```

#### 2.3 Migration & Rollout (Semana 5)
- [ ] API Gateway approach: Node.js → Go gradualmente
- [ ] Feature flags: `API_GO_ENABLED=false` default
- [ ] Canary deployment: 10% → 50% → 100%
- [ ] Monitoramento de latência vs NestJS

**Health Check:**
```bash
# Validar performance antes de 100%
curl -w "@curl-format.txt" -o /dev/null -s http://api:3001/health
# Comparar latência Go vs NestJS
```

---

### **FASE 3: Data Plane - Proxy Rust [PLANEJADO] 🔮**

**Duração:** Semanas 6-8  
**Status:** Backlog  
**RAM Target:** +0MB (substituir Caddy, mesmos recursos)

#### 3.1 Arquitetura Rust
- [ ] Framework: **Axum** (tokio-based, production-ready)
- [ ] Objetivo: Substituir Caddy com melhor throughput
- [ ] Recursos: TLS termination, load balancing, rate limiting

```rust
// apps/proxy-rust/src/main.rs
use axum::{
    extract::Path,
    http::{StatusCode, Request},
    middleware,
    response::Response,
    routing::get,
    Router,
};
use tokio::net::TcpListener;
use tower_http::cors::CorsLayer;

#[tokio::main]
async fn main() {
    let app = Router::new()
        .route("/:project_id/*path", get(proxy_handler))
        .layer(middleware::from_fn(rate_limit))
        .layer(CorsLayer::permissive());
        
    let listener = TcpListener::bind("0.0.0.0:3080").await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

async fn proxy_handler(
    Path(project_id): Path<String>,
) -> Response {
    // Zero-copy proxying para melhor performance
    todo!()
}
```

#### 3.2 Performance Improvements
- [ ] **Zero-copy proxying:** Buffer reutilizável
- [ ] **Connection pooling:** HTTP keep-alive automático
- [ ] **Load balancing:** Round-robin com health checks
- [ ] **Rate limiting:** Token bucket algorithm

**Benchmark Esperado:**
```
Métrica              Caddy      Rust       Melhoria
───────────────────────────────────────────────────
Throughput (RPS)     10.000     30.000     ⚡ 3x
Latency p99          50ms       15ms       ⚡ 3.3x
Memory per conn      ~1MB       ~100KB     ⚡ 10x
```

#### 3.3 Deployment
- [ ] `apps/proxy-rust/Dockerfile` (Alpine base)
- [ ] docker-compose.yml: substituir Caddy
- [ ] Canary: Router 10% → Rust, 90% → Caddy
- [ ] Rollback automático se latência aumentar

---

### **FASE 4: Banco de Dados & Cache [PARALELO] 🔄**

**Duração:** Ongoing (em paralelo com Fases 1-2)  
**Status:** Active  
**RAM Target:** -20% otimização de queries

#### 4.1 PostgreSQL Optimization
- [ ] **Índices Missing:** Analisar EXPLAIN ANALYZE
  ```sql
  -- Adicionar índices críticos
  CREATE INDEX CONCURRENTLY idx_projects_owner_id 
    ON projects(owner_id);
  CREATE INDEX CONCURRENTLY idx_deployments_project_created 
    ON deployments(project_id, created_at DESC);
  ```

- [ ] **Connection Pooling:** pgx/v5 (25-50 connections)
- [ ] **Prepared Statements:** Reescrever queries dinâmicas
- [ ] **Query Optimization:** N+1 queries detection
  ```go
  // Evitar N+1 em deployments list
  // ❌ Ruim:
  for _, project := range projects {
      deployments := db.GetDeploymentsByProjectID(project.ID) // N queries!
  }
  
  // ✅ Bom:
  projectIDs := getProjectIDs(projects)
  allDeployments := db.GetDeploymentsByProjectIDs(projectIDs) // 1 query
  ```

#### 4.2 Redis Optimization
- [ ] **Eviction Policy:** LRU (máximo 256MB)
- [ ] **Persistence:** AOF (Append-Only File) configurado
  ```conf
  # redis.conf
  maxmemory 256mb
  maxmemory-policy allkeys-lru
  appendonly yes
  appendfsync everysec
  ```

- [ ] **Cache Strategy:**
  ```go
  // TTL por tipo
  const (
      ProjectCacheTTL     = 60 * time.Second  // Read-heavy
      DeploymentCacheTTL  = 10 * time.Second  // Muda frequente
      UserSessionTTL      = 24 * time.Hour    // Long-lived
  )
  ```

#### 4.3 SQLite para Lite Edition
- [ ] Adaptar schema para SQLite (AUTOINCREMENT, tipos)
- [ ] Implementar migration runner (não Prisma)
- [ ] Testing em Raspberry Pi Zero 2W

---

## 📈 Métricas & KPIs

### **Antes de Otimização (Baseline)**
```
Métrica                Medição          Alvo
──────────────────────────────────────────────
Total RAM (completo)   ~2.5GB          ↓ 2GB
API Latency p95        ~150ms          < 100ms
Deploy throughput      5 deploys/min   50 deploys/min
Dashboard load time    ~3.5s           < 2s
Worker concurrency     10              50
Database queries/s     ~500            < 100 (com cache)
```

### **Depois de Otimização (Target)**
```
Métrica                Otimizado        Melhoria
──────────────────────────────────────────────
Total RAM (completo)   ~1.5-1.8GB       ↓ 25%
API Latency p95        ~80ms            ↓ 47%
Deploy throughput      50 deploys/min   ⚡ 10x
Dashboard load time    ~1.8s            ↓ 49%
Worker concurrency     50               ⚡ 5x
Database queries/s     ~150             ↓ 70%
```

---

## 🛠️ Implementação - Quick Start

### **Checklist - Fase 1 (Semanas 1-2)**

#### Semana 1:
- [ ] `workers/deploy/main.go` - Docker SDK implementation
  - [ ] docker.NewClient()
  - [ ] docker.BuildImage() com streaming
  - [ ] Error handling + timeout
- [ ] Tests: `workers/deploy/main_test.go`
  - [ ] Unit tests (mocks)
  - [ ] Integration tests (Docker)
- [ ] Dockerfile: multi-stage build
  ```dockerfile
  # Estágio 1: Build
  FROM golang:1.25-alpine as builder
  WORKDIR /build
  COPY go.mod go.sum .
  RUN go mod download
  COPY . .
  RUN CGO_ENABLED=0 go build -o worker main.go
  
  # Estágio 2: Runtime
  FROM alpine:latest
  COPY --from=builder /build/worker /usr/local/bin/
  ENTRYPOINT ["/usr/local/bin/worker"]
  ```

#### Semana 2:
- [ ] Job queue implementation
  - [ ] Channel-based distribution
  - [ ] Persistence (RabbitMQ ou Redis)
- [ ] `workers/deploy/.env.production`
  - [ ] MAX_CONCURRENT_BUILDS=50
  - [ ] DOCKER_TIMEOUT_SECONDS=120
- [ ] Load testing
  - [ ] k6: simulação de 50 builds paralelos
  - [ ] Medir CPU, memória, latência

---

### **Checklist - Fase 2 (Semanas 3-5)**

#### Semana 3:
- [ ] Setup novo projeto Go
  ```bash
  cd apps && mkdir api-go && cd api-go
  go mod init github.com/mateussiqueira/nidus-api
  go get github.com/gofiber/fiber/v3
  go get github.com/jackc/pgx/v5
  ```
- [ ] Estrutura de diretórios
- [ ] Main.go com Fiber
- [ ] PostgreSQL connection
- [ ] Health check endpoint

#### Semana 4:
- [ ] Handlers (auth, projects, deployments)
- [ ] Services layer
- [ ] Database layer (queries otimizadas)
- [ ] Middleware (JWT, logger, rate limit)
- [ ] Tests (testes de performance)

#### Semana 5:
- [ ] Feature flag system
- [ ] Canary deployment strategy
- [ ] Monitoring (Prometheus)
- [ ] Rollback procedure
- [ ] Full integration testing

---

## 📋 Configurações de Ambiente

### **.env.production (Completo - 2GB)**
```env
# API
API_PORT=3001
API_MEMORY_LIMIT=200
API_CONCURRENCY=4

# Deploy Worker
WORKER_PORT=8081
MAX_CONCURRENT_BUILDS=50
WORKER_MEMORY_LIMIT=500
DOCKER_TIMEOUT_SECONDS=120

# Proxy
PROXY_PORT=3080
PROXY_MEMORY_LIMIT=100

# Database
DATABASE_URL=postgresql://user:pass@postgres:5432/nidus
DB_POOL_SIZE=25
DB_STATEMENT_CACHE_SIZE=100

# Redis
REDIS_URL=redis://redis:6379
REDIS_MAX_MEMORY=256mb

# Dashboard
NEXT_PUBLIC_API_URL=https://api.nidus.dev
NEXT_PUBLIC_WS_URL=wss://ws.nidus.dev

# Caddy/Proxy
CADDY_AUTO_HTTPS=on
CADDY_PLUGINS=rate-limit,gzip
```

### **.env.lite (512MB - Raspberry Pi)**
```env
# API
API_PORT=3001
API_MEMORY_LIMIT=64
API_CONCURRENCY=1

# Deploy Worker
WORKER_PORT=8081
MAX_CONCURRENT_BUILDS=2
WORKER_MEMORY_LIMIT=64
DOCKER_TIMEOUT_SECONDS=60

# Database
DATABASE_URL=sqlite:///data/nidus.db
DB_POOL_SIZE=3

# Skip
SKIP_DASHBOARD=true
SKIP_PROXY=true
```

---

## 📊 Comparação vs Concorrentes

### **vs Vercel**
| Feature | Nidus | Vercel | Status |
|---------|-------|--------|--------|
| Build parallelization | 50 | Unlimited | 🟡 Próxima |
| Edge locations | 1 | 34+ | 🔴 Futuro |
| Cold start time | <1s | <100ms | 🟡 Otimizando |
| RAM mínima | 512MB | N/A | ✅ Vantagem |
| Deploy time | ~30s | ~60s | ✅ Vantagem |

### **vs Commit.com (Competitor local)**
| Feature | Nidus | Commit | Status |
|---------|-------|--------|--------|
| Memory footprint | 2GB | ~3GB | ✅ Vantagem |
| Deploy concurrency | 50 | 20 | ✅ Vantagem |
| Startup time (Lite) | ~5s | ~20s | ✅ Vantagem |
| Open source | ✅ | ❌ | ✅ Vantagem |

---

## 🚨 Riscos & Mitigação

| Risco | Impacto | Mitigação |
|-------|--------|-----------|
| Migration Go falha | Rollback 2 semanas | Canary deploy, feature flags |
| Memory leak Go | Crash produção | Profiling com pprof, tests |
| Database performance degrade | Queries lentes | Índices pré-validados, EXPLAIN |
| Redis eviction errors | Deploy jobs perdem | Persistent queue com database |
| Rust proxy breaking changes | Downtime | Gradual rollout, monitoring |

---

## 📞 Responsáveis & Timeline

| Fase | Owner | Start | End | Status |
|------|-------|-------|-----|--------|
| **1 - Worker Go** | DevOps | Jun 24 | Jul 8 | 🟡 Em Progresso |
| **2 - API Go** | Backend | Jul 1 | Jul 22 | ⏳ Planejado |
| **3 - Proxy Rust** | Infrastructure | Jul 15 | Aug 5 | 🔮 Backlog |
| **4 - DB Optimization** | DBA | Jun 24 | Jul 31 | 🟡 Paralelo |

---

## 📚 Documentação de Referência

- [PLAN-MIGRATION.md](./PLAN-MIGRATION.md) - Estratégia detalhada de migração
- [PERFORMANCE.md](./PERFORMANCE.md) - Benchmarks e testes
- [docker-compose.lite.yml](./docker-compose.lite.yml) - Configuração Lite (512MB)
- [docker-compose.prod.yml](./docker-compose.prod.yml) - Configuração Produção
- [SECURITY.md](./SECURITY.md) - Considerações de segurança

---

## 🎉 Resultado Final (ETA: Agosto 2026)

```
┌─────────────────────────────────────────────────────────┐
│            NIDUS - OTIMIZADO E PRONTO                  │
├─────────────────────────────────────────────────────────┤
│ ✅ Deploy Worker em Go (6.7x RAM savings)              │
│ ✅ API em Go (1.5-2GB total)                           │
│ ✅ 50 builds simultâneos                               │
│ ✅ <80ms latência p95                                  │
│ ✅ Compete com Vercel em performance                   │
│ ✅ Roda em 512MB (Lite) até 2GB (Completo)             │
│ ✅ Open source + Deploy simplificado                   │
└─────────────────────────────────────────────────────────┘
```

---

**Última atualização:** 28 de junho de 2026  
**Próxima revisão:** 05 de julho de 2026 (Fase 1 review)
