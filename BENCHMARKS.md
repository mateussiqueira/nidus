# Nidus Proxy Benchmarks

## Methodology

All benchmarks run on broto-vps against the same upstream (dashboard on `:3000`).

### Tools
- **wrk** — HTTP benchmarking for end-to-end proxy throughput
- **criterion** — Rust micro-benchmarks for internal proxy components
- **memory-compare.sh** — Binary size and memory footprint comparison

### Proxies Tested
| Proxy | Language | Port | Binary |
|-------|----------|------|--------|
| nidus-proxy (Rust) | Rust + hyper | 8089 | `/root/nidus/rust/target/release/nidus-proxy` |
| nidus-proxy (Go) | Go | 8080 | `/root/nidus/apps/proxy/nidus-proxy` |
| Caddy | Go | 443 | `/usr/bin/caddy` |

### Benchmark Parameters
- Concurrency: 100 connections (default)
- Duration: 10 seconds per proxy
- Threads: 4 (wrk)

## Results

> **Last run:** TBD

### wrk Throughput

| Proxy | Requests/sec | Avg Latency | P50 | P99 |
|-------|-------------|-------------|-----|-----|
| Rust (nidus-proxy) | — | — | — | — |
| Go (nidus-proxy) | — | — | — | — |
| Caddy | — | — | — | — |

### Binary Size

| Binary | Size |
|--------|------|
| Rust proxy | — |
| Go proxy | — |
| Caddy | — |
| Go API | — |
| Rust CLI | — |

### Memory (RSS)

| Proxy | Idle (KB) | Under Load (KB) |
|-------|-----------|-----------------|
| Rust proxy | — | — |
| Go proxy | — | — |
| Caddy | — | — |

### Rust Internal Benchmarks (criterion)

| Benchmark | Time | Throughput |
|-----------|------|------------|
| proxy_forward_local | — | — |
| reqwest_keepalive | — | — |
| hyper_direct | — | — |

## How to Run

```bash
# Full end-to-end benchmark
bash /root/nidus/scripts/benchmark-proxy.sh

# Memory comparison
bash /root/nidus/scripts/memory-compare.sh

# Rust criterion benchmarks
cd /root/nidus/rust && cargo bench -p nidus-proxy
```
