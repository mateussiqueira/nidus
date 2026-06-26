# Plano de Migração: Nidus PaaS → Go + Rust Híbrido

## Objetivo

Migrar o Nidus PaaS de Node.js/TypeScript para **Go + Rust** para suportar **1000+ deploys/dia** com performance extrema e economia de recursos.

## Arquitetura Alvo

```
┌──────────────────────────────────────────────────────────┐
│                    CONTROL PLANE (Go)                     │
│  API REST + Auth + WebSocket + Deploy Queue              │
│  Porta 3001                                              │
├──────────────────────────────────────────────────────────┤
│                   DEPLOY WORKER (Go)                      │
│  Worker pool: NumCPU goroutines (max 16)                 │
│  Docker SDK nativo + BuildKit streaming                  │
├──────────────────────────────────────────────────────────┤
│                    DATA PLANE (Rust)                      │
│  Reverse proxy de alta performance para apps deployados  │
│  Rate limiting + TLS + WebSocket proxy                   │
│  Substitui Caddy no tráfego de apps                      │
├──────────────────────────────────────────────────────────┤
│                   DASHBOARD (Next.js)                     │
│  Frontend SPA — sem mudança                              │
│  Porta 3000                                              │
└──────────────────────────────────────────────────────────┘
```

## Por que essa divisão

| Componente | Linguagem | Motivo |
|---|---|---|
| **Deploy Worker** | Go | Docker SDK nativo (Go), goroutines para builds concorrentes, binário único |
| **API/Control Plane** | Go | Fiber/Chi HTTP, pgx para Postgres, go-redis para fila, middleware JWT nativo |
| **Data Plane (Proxy)** | Rust | Máxima performance para proxy de apps deployadas, zero-copy, memória garantida |
| **Dashboard** | Next.js | Roda no browser, sem benefício de migração |

## Fases de Implementação

---

### Fase 1: Deploy Worker Go (Semanas 1-2)
**Prioridade: CRITICA — maior ganho de performance**

O worker Go já existe em `workers/deploy/main.go` mas precisa de melhorias:

#### O que fazer
1. **Melhorar o worker Go existente:**
   - Trocar `go-redis/v8` por `go-redis/v9` com connection pooling
   - Usar Docker SDK for Go (`github.com/docker/docker/client`) em vez de `os/exec` para `docker` CLI
   - Implementar streaming de logs do build via Docker API
   - Adicionar health check endpoint (HTTP na porta 8080)
   - Adicionar graceful shutdown com signal handling
   - Implementar BullMQ job acknowledgment correto (o atual faz BRPOP manual)
   - Adicionar métricas (Prometheus)

2. **Desativar o worker BullMQ no NestJS:**
   - Remover `onModuleInit` que inicia o worker
   - Manter apenas a enfileiração (Queue.add)
   - O Go worker consome de `deploy-queue` via BRPOP

3. **Testes de carga:**
   - Benchmark: 100 deploys simultâneos
   - Medir tempo de build Docker, memória, CPU
   - Comparar com Node.js BullMQ worker

#### Arquivos
- `workers/deploy/main.go` — worker principal (refatorar)
- `workers/deploy/go.mod` — dependências atualizadas
- `workers/deploy/Dockerfile` — containerizar o worker

#### Dependências Go
```
github.com/redis/go-redis/v9
github.com/jackc/pgx/v5
github.com/docker/docker/client
github.com/docker/docker/api/types
github.com/gorilla/mux (ou net/http padrão)
github.com/prometheus/client_golang
```

---

### Fase 2: API/Control Plane Go (Semanas 3-4)
**Prioridade: ALTA — consolidar o stack**

Reescrever o NestJS API em Go:

#### Endpoints para implementar (20 endpoints)

**Auth (3)**
- `POST /api/auth/register` — bcrypt hash + JWT
- `POST /api/auth/login` — bcrypt compare + JWT
- `GET /api/auth/me` — JWT middleware

**Projects (7)**
- `GET /api/projects` — listar do usuário
- `GET /api/projects/:id` — buscar por ID
- `POST /api/projects` — criar (gerar slug)
- `PATCH /api/projects/:id` — atualizar
- `GET /api/projects/:id/deployments` — listar deploys
- `GET /api/projects/:id/metrics` — métricas Docker
- `POST /api/projects/:id/deploy` — enfileirar deploy

**Deployments (5)**
- `GET /api/projects/:projectId/deployments` — listar
- `GET /api/projects/:projectId/previews` — previews
- `GET /api/projects/:projectId/metrics` — métricas
- `POST /api/projects/:projectId/deploy` — trigger deploy
- `GET /api/projects/:projectId/deployments/:id/logs` — logs

**Databases (4)**
- `GET /api/databases` — listar
- `GET /api/databases/:id` — buscar
- `POST /api/databases` — criar (createdb)
- `DELETE /api/databases/:id` — deletar (dropdb)

**System (3)**
- `POST /api/webhook/github` — webhook GitHub
- `GET /api/metrics` — métricas JSON
- `GET /health` — health check

#### Estrutura Go
```
cmd/
  api/
    main.go              — bootstrap, router, graceful shutdown
internal/
  auth/
    handler.go           — register, login, me
    service.go           — bcrypt, JWT
    middleware.go         — JWT guard
  projects/
    handler.go           — CRUD + deploy trigger
    service.go           — lógica de negócio
  deployments/
    handler.go           — status, logs, metrics
    service.go           — Docker metrics
  databases/
    handler.go           — CRUD databases
    service.go           — createdb/dropdb via pgx
  webhook/
    handler.go           — GitHub push webhook
  metrics/
    handler.go           — Prometheus + JSON metrics
  infra/
    database.go          — pgx pool config
    redis.go             — go-redis client
    docker.go            — Docker client
    config.go            — env vars
  middleware/
    jwt.go               — JWT validation
    cors.go              — CORS
    ratelimit.go         — rate limiting
    requestid.go         — request ID
pkg/
  models/
    user.go
    project.go
    deployment.go
    database.go
```

#### Dependências Go
```
github.com/gofiber/fiber/v2       — HTTP framework (rápido, expressivo)
github.com/golang-jwt/jwt/v5      — JWT
github.com/jackc/pgx/v5           — PostgreSQL driver
github.com/redis/go-redis/v9      — Redis client
github.com/docker/docker/client   — Docker API
golang.org/x/crypto/bcrypt        — password hashing
github.com/prometheus/client_golang — métricas
```

---

### Fase 3: Data Plane — Rust Proxy (Semanas 5-6)
**Prioridade: MÉDIA — performance extrema para apps deployadas**

Substituir Caddy por um proxy Rust de alta performance para rotear tráfego de apps deployadas:

#### O que faz
- Roteamento por hostname para containers Docker
- TLS termination (Let's Encrypt)
- Rate limiting por IP/tenant
- WebSocket proxying (para apps real-time)
- Cache de assets estáticos
- Streaming de logs em tempo real (SSE/WebSocket)
- Load balancing entre réplicas (futuro)

#### Arquivos
```
proxy/
  Cargo.toml
  src/
    main.rs              — bootstrap, signal handling
    proxy/
      mod.rs             — reverse proxy core
      router.rs          — routing por hostname
      tls.rs             — TLS/ACME
    middleware/
      ratelimit.rs       — token bucket rate limiting
      cors.rs            — CORS
      logging.rs         — structured logging
    cache/
      mod.rs             — response cache
    metrics/
      mod.rs             — Prometheus metrics
    config.rs            — env vars
```

#### Dependências Rust
```toml
[dependencies]
axum = "0.8"           — HTTP framework
hyper = { version = "1", features = ["full"] } — HTTP
hyper-util = { version = "0.1", features = ["full"] }
tokio = { version = "1", features = ["full"] } — async runtime
tower = "0.5"          — middleware
tower-http = { version = "0.6", features = ["cors", "trace"] }
redis = { version = "0.27", features = ["tokio-comp"] } — Redis
sqlx = { version = "0.8", features = ["postgres", "runtime-tokio"] } — PostgreSQL
serde = { version = "1", features = ["derive"] }
serde_json = "1"
jsonwebtoken = "9"     — JWT
rustls = "0.23"        — TLS
rcgen = "0.13"         — certificate generation
dashmap = "6"          — concurrent hashmap
metrics = "0.24"
metrics-exporter-prometheus = "0.16"
tracing = "0.1"
tracing-subscriber = "0.3"
```

---

### Fase 4: Dashboard + Integração (Semana 7)
**Prioridade: ALTA**

1. Dashboard Next.js continua como está
2. Ajustar `NEXT_PUBLIC_API_URL` para apontar para a nova API Go
3. Testar fluxo completo: login → criar projeto → deploy → acessar app
4. Deploy do stack completo via docker-compose

---

## Stack Final

```
┌─────────────────────────────────────────────────┐
│  Binários                                        │
│  ├── nidus-api        (Go, ~15MB, porta 3001)   │
│  ├── nidus-worker     (Go, ~15MB, Redis worker) │
│  ├── nidus-proxy      (Rust, ~5MB, porta 80/443)│
│  └── nidus-dashboard  (Next.js, porta 3000)     │
│                                                  │
│  Containers                                      │
│  ├── nidus-postgres   (PostgreSQL 16)            │
│  ├── nidus-redis      (Redis 7)                  │
│  └── nidus-minio      (MinIO, opcional)          │
└─────────────────────────────────────────────────┘
```

## Comparação de Performance Esperada

| Métrica | Node.js (atual) | Go + Rust (alvo) |
|---|---|---|
| RAM por worker | ~120MB | ~15MB |
| Tempo de deploy | ~45s (Docker build) | ~30s (BuildKit cache) |
| Deploys simultâneos | 2 (concurrency BullMQ) | 16 (goroutines) |
| Latência API (p99) | ~50ms | ~5ms |
| Conexões simultâneas | ~1000 | ~50000 |
| Binary size | N/A (Node.js runtime) | ~15MB (Go) + ~5MB (Rust) |

## Riscos e Mitigações

| Risco | Mitigação |
|---|---|
| Docker SDK Go é complexo | Usar `docker/cli` wrapper ou SDK oficial |
| Rust proxy tem curva de aprendizado | Começar com Axum (simpler que Actix) |
| Migração quebra API existente | Manter compatibilidade: mesmos endpoints, mesmos contratos |
| Dashboard precisa de ajustes | Testar JWT compatível (mesmo secret, mesmo formato) |
| Workers Go e Node.js competem | Desativar worker Node.js antes de ativar Go |

## Ordem de Execução

1. **Fase 1** (Go Worker) → imediato, maior impacto
2. **Fase 2** (Go API) → consolidar stack
3. **Fase 3** (Rust Proxy) → performance extrema
4. **Fase 4** (Integração) → deploy completo

---

*Plano versionado. Última atualização: 2026-06-26*
