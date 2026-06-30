# Nidus v2.0 — Arquitetura Rust + Go (Roadmap 12 meses)

## Research: O que os melhores do mundo fazem

### Fly.io — Rust + Go
- **Proxy (Rust)**: Fly Proxy é escrito em Rust. Lida com ~10M req/s por edge node.
  - Latência de proxy: <1ms (Rust + io_uring)
  - Memória: ~20MB por instância vs ~200MB em Go
- **Orchestrator (Go)**: Fly Machines API em Go. Gerencia ciclo de vida de VMs.
- **Build system (Go + Rust)**: Cloud Builders usam Go para orquestração, Rust para sandbox.
- **Lições**: Proxy/edge em Rust, API/orchestration em Go, build em Rust sandbox.

### Railway — Go + Rust
- **API (Go)**: Railway API é Go puro. Monólito bem estruturado.
- **Build system**: BuildKit + Docker. Cache inteligente.
- **Edge proxy**: Baseado em Envoy (C++) com sidecar Rust.
- **Lições**: Monólito Go bem feito escala até 1M usuários. Rust complementa no path crítico.

### Vercel — Rust edge functions
- **Edge Runtime (Rust)**: Vercel Edge Functions usam V8 isolado + Rust orchestrator.
  - Cold start: <50ms (vs 200ms+ em Node.js)
  - Densidade: 1000 funções/container (vs 10 em Node)
- **Build system**: Turborepo (Rust). Build cache compartilhado.
- **Analytics**: ClickHouse (C++) + Rust pipelines.
- **Lições**: Rust no path quente (edge, build) reduz custos em 10x.

### Cloudflare — Rust everywhere
- **Workers (Rust)**: V8 isolates managed by Rust orchestrator.
  - 0ms cold start (pre-warmed isolates)
  - 1M+ requests por segundo por servidor
- **Proxy (Rust)**: Pingora — substituiu Nginx. 160TB/s throughput.
- **Lições**: Rust no proxy economiza 1000+ servidores vs Nginx.

### Discord — Go → Rust migration
- **Read States**: Migrou de Go para Rust. Latência caiu de 10ms para 1ms.
- **Memória**: 2.5GB (Go) → 500MB (Rust) para mesma carga.
- **GC pauses**: Eliminados completamente.
- **Lições**: Serviços stateful se beneficiam MUITO de Rust.

### Supabase — PostgreSQL + Rust + Go
- **Auth (Go)**: Supabase Auth em Go. Batalha-testado.
- **Edge Functions (Rust)**: Deno runtime + Rust orchestrator.
- **Realtime (Rust)**: WebSocket server em Rust. 1M conexões simultâneas.
- **Lições**: Cada linguagem no seu melhor. Go pra APIs, Rust pra path crítico.

## Arquitetura Nidus v2.0

### Rust (core crítico de performance)
1. **nidus-proxy** — Reverse proxy (substitui Caddy/NGINX)
   - Tech: hyper + tokio + rustls
   - Memória: ~15MB vs ~100MB (Caddy)
   - Throughput: 50K req/s vs 5K
   - Features: SNI routing, TLS 1.3, HTTP/3, rate limiting nativo

2. **nidus-builder** — Build system (substitui exec.Command docker)
   - Tech: buildkit-rs + tokio
   - Build cache inteligente com S3
   - Builds paralelos com resource limits
   - Métricas de build em tempo real

3. **nidus-edge** — Edge functions runtime
   - Tech: wasmtime (WASM) ou v8 isolates
   - Cold start <5ms
   - Zero-downtime deploys
   - Multi-tenant isolation

4. **nidus-metrics** — Coletor de métricas
   - Tech: tikv + prometheus
   - Stream processing em tempo real
   - Armazenamento colunar eficiente

### Go (orquestração e API)
1. **nidus-api** — API REST (refatorar, já existe)
2. **nidus-orchestrator** — Gerenciador de containers
3. **nidus-auth** — Autenticação/OAuth
4. **nidus-scheduler** — Cron jobs + health checks

### Frontend
1. **nidus-dashboard** — Next.js (já existe)
2. **nidus-cli** — Rust CLI (tui-rs + indicatif)
   - Progressive bar estilo Railway
   - Logs com syntax highlighting
   - Tamanho: 5MB vs 80MB (Node)

## Comparativo de Performance

| Componente | Atual | v2.0 Rust | Melhoria |
|-----------|-------|-----------|----------|
| Proxy | Caddy (Go) 100MB | hyper (Rust) 15MB | 85% menos RAM |
| Builder | exec.Command | buildkit-rs | 5x mais rápido |
| CLI | Node.js 80MB | Rust 5MB | 16x menor |
| Health checker | Go goroutine | tokio task | 3x mais eficiente |
| WebSocket | gorilla/ws | tokio-tungstenite | 10x mais conexões |

## Developer Experience (DX)

### CLI v2.0 (Rust)
```
$ nidus deploy
 ⠋ Building...          [2.3s]
 ⠙ Pushing image...      [1.1s]  
 ⠹ Starting container...  [0.8s]
 ✅ Deployed in 4.2s → https://app.nidus.app

$ nidus logs --follow
 2026-06-30 15:30:01 GET / 200 2ms
 2026-06-30 15:30:05 POST /api 201 15ms
```

### Dashboard v2.0
- Live metrics com WebSocket streaming (via Rust proxy)
- Build progress com eventos SSE (via Rust builder)
- Deploy preview com iframe embutido
- Dark mode + themes customizáveis

## Plano de Migração (12 meses)

### Fase 1: Fundação Rust (Mês 1-3)
- [ ] Setup workspace Rust (Cargo workspace)
- [ ] nidus-proxy: reverse proxy com TLS + SNI routing
- [ ] nidus-cli: CLI em Rust com TUI
- [ ] Testes de carga: 10K req/s no proxy

### Fase 2: Core em Rust (Mês 4-6)
- [ ] nidus-builder: Build system com cache S3
- [ ] nidus-metrics: Coletor de métricas em tempo real
- [ ] Migração do health checker para Rust
- [ ] Testes: 100 projetos simultâneos

### Fase 3: Edge (Mês 7-9)
- [ ] nidus-edge: WASM runtime para funções
- [ ] nidus-scheduler: Cron jobs + filas
- [ ] Integração completa Rust↔Go via gRPC
- [ ] Benchmark: 1M req/s no proxy

### Fase 4: Produção (Mês 10-12)
- [ ] Migração completa do proxy (Caddy → Rust)
- [ ] Deploy canário (10% tráfego Rust, 90% Go)
- [ ] Monitoramento comparativo
- [ ] Cutover final com zero downtime

## Tech Stack Final

```
┌─────────────────────────────────────────────┐
│                 nidus.app                    │
├─────────────────────────────────────────────┤
│  Rust Proxy (hyper)    ← TLS, HTTP/3, WSS   │
├─────────────────────────────────────────────┤
│  Go API (net/http)     ← REST, OAuth        │
├─────────────────────────────────────────────┤
│  Rust Builder          ← BuildKit, cache    │
├─────────────────────────────────────────────┤
│  Rust Edge (wasmtime)  ← Functions runtime  │
├─────────────────────────────────────────────┤
│  Rust Metrics          ← Prometheus + colunar│
├─────────────────────────────────────────────┤
│  Go Orchestrator       ← Docker, Compose    │
├─────────────────────────────────────────────┤
│  PostgreSQL + Redis    ← Dados e filas       │
└─────────────────────────────────────────────┘
```

## Vantagens Competitivas

1. **Custo**: R$30/mês vs R$200+/mês (Vercel/Railway)
   - Rust proxy: 15MB RAM vs 100MB (Caddy) vs 500MB (Nginx)
   - Densidade: 50 apps/servidor vs 10 apps/servidor

2. **Performance**: 
   - Proxy: 50K req/s (Rust) vs 5K (Go)
   - Build: Cache inteligente reduz builds em 80%
   - Edge: 5ms cold start vs 200ms (Node)

3. **DX**:
   - CLI em Rust: 5MB, instantâneo
   - Logs com syntax highlighting
   - Deploy preview automático
   - Dashboard com WebSocket live metrics

4. **Soberania**:
   - Self-hosted: seus dados, seu servidor
   - Sem vendor lock-in
   - Open source (MIT)
   - Migração fácil: export/import JSON

