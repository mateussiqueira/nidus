# StackRun — Edge Computing & Observabilidade: Pesquisa Técnica Profunda

## Parte 1: Edge Runtime — Arquitetura WASM

### Como os gigantes fazem

#### Cloudflare Workers (V8 Isolates)
```
┌─────────────────────────────────────────┐
│              Workers Runtime             │
├─────────────────────────────────────────┤
│  V8 Isolate Pool (pre-warmed)           │
│  ├─ Isolate 1: user-a (5MB heap)        │
│  ├─ Isolate 2: user-b (5MB heap)        │
│  └─ Isolate N: user-n (5MB heap)        │
├─────────────────────────────────────────┤
│  Rust Orchestrator (workerd)            │
│  ├─ Isolate lifecycle (create/destroy)  │
│  ├─ Request routing (URL → isolate)     │
│  ├─ Memory limits (128MB hard cap)      │
│  ├─ CPU time limits (50ms wall clock)   │
│  └─ I/O proxy (network, KV, Durable)    │
├─────────────────────────────────────────┤
│  Linux Kernel                           │
│  └─ seccomp, cgroups, namespaces        │
└─────────────────────────────────────────┘
```
- Cold start: 0ms (pre-warmed pool)
- Densidade: ~1000 isolates/server
- Isolamento: V8 sandbox + seccomp
- **Não é WASM**. É JavaScript dentro de V8.

#### Vercel Edge Functions (V8 + Rust)
- Mesma arquitetura do Cloudflare (workerd)
- Diferencial: suporta WebAssembly via V8
- Limite: 1MB de código, 128MB de memória
- Cold start: <50ms (V8 instantiation)

#### Fermyon Spin (WASM puro — Referência para StackRun)
```
┌─────────────────────────────────────────┐
│              Spin Runtime                │
├─────────────────────────────────────────┤
│  wasmtime::Engine (shared, compiled)    │
│  ├─ wasmtime::Store (per-request)       │
│  ├─ wasmtime::Linker (WASI imports)     │
│  └─ wasmtime::Instance (isolated)       │
├─────────────────────────────────────────┤
│  WASI Preview 2 Interface               │
│  ├─ wasi:http/incoming-handler          │
│  ├─ wasi:keyvalue/atomics               │
│  ├─ wasi:logging/logging                │
│  └─ wasi:cli/run                        │
├─────────────────────────────────────────┤
│  Spin Application (WASM)                │
│  └─ Compiled from Rust, Go, JS, Python  │
└─────────────────────────────────────────┘
```
- Cold start: <1ms (WASM instantiation)
- Tamanho: ~2MB por instância
- Linguagens: Rust, Go, JS, Python, C, Zig
- **Vantagem**: Sem garbage collector. Determinístico.

### Arquitetura StackRun Edge

```
┌──────────────────────────────────────────────────┐
│                 stackrun-edge (Rust)                 │
├──────────────────────────────────────────────────┤
│  Request Router                                  │
│  ┌────────────────────────────────────────────┐  │
│  │ Host: fn.project.stackrun.vercel.app → EdgeFunction  │  │
│  │ Path: /api/users → EdgeFunction            │  │
│  │ Fallback → Container (Docker)              │  │
│  └────────────────────────────────────────────┘  │
├──────────────────────────────────────────────────┤
│  WASM Runtime Pool (wasmtime)                    │
│  ┌────────────────────────────────────────────┐  │
│  │ Engine (shared, compiled once)             │  │
│  │ ├─ Module cache (LRU, 100 modules)         │  │
│  │ ├─ Store pool (pre-warmed, 10 stores)      │  │
│  │ └─ Instance per request (<1ms create)      │  │
│  │                                             │  │
│  │ Resource limits:                            │  │
│  │ ├─ Memory: 128MB per instance               │  │
│  │ ├─ CPU: 50ms per request                    │  │
│  │ ├─ Fuel metering (wasmtime::Store::fuel)    │  │
│  │ └─ Epoch interruption (safety net)          │  │
│  └────────────────────────────────────────────┘  │
├──────────────────────────────────────────────────┤
│  I/O Proxy (WASI Preview 2)                      │
│  ┌────────────────────────────────────────────┐  │
│  │ wasi:http    → HTTP fetch (restricted)     │  │
│  │ wasi:keyvalue → Redis/PostgreSQL via host  │  │
│  │ wasi:logging   → Structured logging        │  │
│  │ wasi:clocks    → Monotonic time            │  │
│  │ custom:stackrun   → StackRun API calls           │  │
│  └────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────┘
```

### Comparativo Técnico

| Métrica | Docker Container | V8 Isolate (Vercel) | WASM (StackRun) |
|---------|-----------------|---------------------|--------------|
| Cold start | 500ms-30s | 50ms | **<1ms** |
| Memória base | 20-200MB | 5MB | **2MB** |
| Densidade/servidor | 10-50 | 1000 | **5000** |
| Isolamento | Kernel namespace | V8 sandbox | **WASM sandbox** |
| Linguagens | Todas | JS/TS apenas | Rust, Go, JS, Python |
| Determinismo | Não | Não | **Sim** |
| Segurança | cgroups | V8 sandbox | **Capa teórica** |

### Vantagem Estratégica do WASM

1. **Densidade 100x**: 5000 funções/servidor vs 50 containers
2. **Custo 50x menor**: R$0.006/execução vs R$0.30/container
3. **Latência previsível**: <1ms sempre (sem GC pauses)
4. **Multi-linguagem**: Rust, Go, JS, Python — não só JS
5. **Portabilidade**: Mesmo .wasm roda em qualquer cloud

## Parte 2: Observabilidade — OpenTelemetry

### Stack Enterprise

```
┌─────────────────────────────────────────────────┐
│                 Aplicação (Rust/Go)              │
│  ┌───────────────────────────────────────────┐  │
│  │ OpenTelemetry SDK (tracing-opentelemetry) │  │
│  │ ├─ Traces: cada request HTTP              │  │
│  │ ├─ Metrics: req/s, latência, erros        │  │
│  │ └─ Logs: JSON estruturado + trace_id      │  │
│  └───────────────────────────────────────────┘  │
├─────────────────────────────────────────────────┤
│                 Coleta                           │
│  ┌───────────────────────────────────────────┐  │
│  │ OpenTelemetry Collector (Rust)            │  │
│  │ ├─ Receivers: OTLP gRPC (:4317)          │  │
│  │ ├─ Processors: batch, tail sampling       │  │
│  │ └─ Exporters: Jaeger + Prometheus         │  │
│  └───────────────────────────────────────────┘  │
├─────────────────────────────────────────────────┤
│                 Storage                          │
│  ┌───────────────────────────────────────────┐  │
│  │ Jaeger (traces) / Tempo (traces)          │  │
│  │ Prometheus (metrics) / VictoriaMetrics     │  │
│  │ Loki (logs) / Quickwit (logs)             │  │
│  └───────────────────────────────────────────┘  │
├─────────────────────────────────────────────────┤
│                 Visualização                     │
│  ┌───────────────────────────────────────────┐  │
│  │ Grafana Dashboards                        │  │
│  │ ├─ StackRun Overview (health, deploys, API)  │  │
│  │ ├─ Project Detail (per-project metrics)   │  │
│  │ ├─ Edge Functions (cold starts, latency)  │  │
│  │ └─ Business (MRR, users, deploys/dia)     │  │
│  └───────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
```

### Tracing — Anatomia de um Request

```
Trace: deploy-abc123 (1.2s total)
├─ Span: API POST /deploy (200ms)
│  ├─ Span: auth.validate_token (5ms)
│  ├─ Span: db.insert_deployment (15ms)
│  └─ Span: redis.enqueue_job (3ms)
├─ Span: Worker process (950ms)
│  ├─ Span: git.clone (200ms)
│  │  └─ Span: http.fetch_github (180ms)
│  ├─ Span: builder.build (600ms)
│  │  ├─ Span: docker.pull_base_image (100ms)
│  │  ├─ Span: docker.build_layer_1 (200ms)
│  │  └─ Span: docker.build_layer_2 (300ms)
│  └─ Span: docker.run_container (150ms)
└─ Span: health_check (50ms)
```

### Métricas Chave (RED Method)

```
Rate:     req/s por endpoint
Errors:   % de 5xx
Duration: p50, p95, p99 latência

Plus:
- Build duration (p50/p95/p99)
- Container density (apps/server)
- Cold start time (edge functions)
- Cache hit ratio (builder, proxy)
- MRR (Monthly Recurring Revenue)
```

### Custo de Observabilidade

| Stack | Custo/mês | Notas |
|-------|-----------|-------|
| Grafana Cloud Free | $0 | 10K metrics, 50GB logs, 50GB traces |
| Self-hosted (VPS) | $0 | Roda no mesmo servidor |
| Datadog | $500+ | Enterprise |
| New Relic | $300+ | Enterprise |

**Vantagem StackRun**: Observabilidade enterprise-level por $0.
