# 🚀 StackRun — Deploy like Vercel. Run on your own server.

**Self-hosted PaaS. Open source. Zero vendor lock-in.**

[![Production](https://img.shields.io/badge/production-stackrun.vercel.app-22c55e)](https://stackrun.vercel.app)
[![Tests](https://img.shields.io/badge/tests-99%20passing-22c55e)](https://github.com/mateussiqueira/stackrun/actions)

## ⚡ Quick Start

```bash
curl -sSL https://stackrun.vercel.app/install.sh | bash
```

60 seconds. Your own PaaS. On your server.

## 🎯 What is StackRun?

StackRun is a self-hosted platform that gives you the Vercel/Railway experience on your own server. Git push to deploy, automatic SSL, managed databases, Docker support — all without vendor lock-in.

```
Your code → git push → StackRun builds → Docker container → Live URL
```

## ✨ Features

- 🚀 **Git Push to Deploy** — Push to main, your app is live in seconds
- 🦀 **Rust Core** — Proxy (8MB), CLI (6MB), Builder (3.6MB), Edge WASM (15MB)
- 🐹 **Go API** — REST API, worker queue, health checker, cron scheduler
- 🐳 **Docker + Compose** — Multi-service stacks with persistent volumes
- 🔒 **SSL Automático** — TLS certificates for all domains via Caddy
- 🗄️ **Managed Databases** — PostgreSQL provisioned with 1 click
- 🌐 **Custom Domains** — Your domain, your brand, no lock-in
- 📊 **Real-time Metrics** — CPU, RAM, uptime, logs via WebSocket
- 🤖 **CI/CD Native** — GitHub Actions, webhooks, preview deployments
- 🔐 **RBAC** — Admin, developer, viewer roles per project
- 💳 **Billing** — AbacatePay (Brazil) + Stripe (Global)

## 📦 SDKs

```js
// JavaScript
import { StackRunClient } from "@stackrun/sdk"
```

```python
# Python
from stackrun import StackRunClient
```

```go
// Go
import "github.com/mateussiqueira/stackrun/packages/sdk-go/stackrun"
```

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  stackrun.vercel.app                     │
├─────────────────────────────────────────────────────────┤
│  🦀 Rust Proxy (hyper)     ← TLS, HTTP/3, SNI routing   │
│  🐹 Go API (net/http)      ← REST, OAuth, billing       │
│  🦀 Rust Builder (BuildKit) ← Docker build cache         │
│  🦀 Rust Edge (wasmtime)   ← WASM functions <1ms start   │
│  🐹 Go Worker              ← Deploy queue, health check  │
│  ⚡ Next.js Dashboard      ← UI, admin, templates        │
│  🐘 PostgreSQL + Redis     ← Data + job queue            │
└─────────────────────────────────────────────────────────┘
```

## 🚀 Deploy StackRun

### Option 1: Self-Hosted (Free)
```bash
# Buy a VPS (Hostinger KVM 2 recommended, R$55/mês)
curl -sSL https://stackrun.vercel.app/install.sh | bash
```

### Option 2: StackRun Cloud (R$49/mês)
Managed by us. Zero configuration. [stackrun.vercel.app/dashboard/billing](https://stackrun.vercel.app/dashboard/billing)

## 📊 Benchmarks

| Endpoint | Req/sec | Latency |
|----------|---------|---------|
| Health check | 5,660 | 14.5ms |
| API projects | 3,366 | 17.2ms |
| Dashboard SSR | 910 | 11.0ms |
| Static file | 2,780 | 35.9ms |

*Benchmarked on VPS 4 vCPU, 16GB RAM. Zero failures in 17,000 requests.*

## 🧪 Tests

- **99 unit tests** — Go API, 55.4% coverage, 0 race conditions
- **27 API endpoints** — 100% pass rate
- **17,000 stress test** — 0 failures

## 🤝 Contributing

StackRun is MIT licensed. Contributions welcome!

```bash
git clone https://github.com/mateussiqueira/stackrun.git
cd stackrun
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

MIT © 2026 StackRun

---

**Deploy your stack. Run the world.** 🚀
