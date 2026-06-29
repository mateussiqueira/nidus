# Nimbus

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.2.0-orange.svg)](https://github.com/mateussiqueira/nidus/releases)

> Self-hosted PaaS. Deploy apps, databases, and domains from a single control plane. Vercel-like experience on your own infra.

## What is Nimbus?

Nimbus is a self-hosted platform for deploying applications. Connect your Git repo, push, and it builds + runs your app in isolated Docker containers — with managed databases, custom domains, real-time metrics, and transactional email. All on your server, no vendor lock-in.

**Try it:** [nimbus200.vercel.app](https://nimbus200.vercel.app)

## Features

| Category | Feature | Status |
|---|---|---|
| **Deploy** | Git push → auto build + container | ✅ |
| **Domains** | Custom domains + SSL via Caddy | ✅ |
| **Databases** | Managed PostgreSQL provisioning | ✅ |
| **Mail** | Transactional email API + MCP server | ✅ |
| **Monitoring** | Prometheus + Grafana (5 dashboards) | ✅ |
| **Metrics** | Per-project CPU/Memory history | ✅ |
| **Rollback** | Instant rollback to any deployment | ✅ |
| **Previews** | Branch-based preview deployments | ✅ |
| **CLI** | `nidus deploy` from terminal | ✅ |

## Stack

| Component | Language | Role | Memory |
|---|---|---|---|
| **API** | Go 1.25 | REST API, auth, webhooks, mail | ~16MB |
| **Deploy Worker** | Go 1.25 | Docker build + deploy pipeline | ~14MB |
| **Dashboard** | Next.js 16 + React 19 | Admin interface + charts | ~76MB |
| **CLI** | Node.js/TypeScript | Terminal deploy tool | — |
| **Database** | PostgreSQL 16 / SQLite | Application data | — |
| **Cache/Queue** | Redis 7 | Job queue + rate limiting | — |
| **Proxy** | Caddy | HTTPS, routing, security headers | — |
| **Monitoring** | Prometheus + Grafana + cAdvisor | Metrics, dashboards, alerts | — |
| **Mail** | Nidus Mail (sendmail/SMTP) | Transactional email | — |

## Quick Start

### Prerequisites

| Mode | RAM | Disk | Includes |
|---|---|---|---|
| **Lite** | 512MB | 2GB | API + Worker (SQLite) |
| **Full** | 2GB | 10GB | All components |

- Docker + Docker Compose
- Git

### Full Stack

```bash
git clone https://github.com/mateussiqueira/nidus.git
cd nidus
cp .env.example .env
docker compose up -d
```

### Lite Mode (512MB)

```bash
git clone https://github.com/mateussiqueira/nidus.git
cd nidus
docker compose -f docker-compose.lite.yml up -d
```

Lite mode uses SQLite instead of PostgreSQL and excludes Dashboard/Redis/Proxy.

### Access

- **Dashboard:** [http://localhost:3000](http://localhost:3000)
- **API:** [http://localhost:3001](http://localhost:3001)
- **Grafana:** [http://localhost:3004](http://localhost:3004) (admin / nidus_grafana_2026)

### Default Credentials

- Email: `demo@nidus.dev`
- Password: `demo123456`

## Deploy Your App

### Via CLI

```bash
npm install -g nidus-cli
nidus login
cd my-project
nidus deploy
```

### Via API

```bash
# Create project
curl -X POST http://localhost:3001/api/projects \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"my-app","slug":"my-app","repoUrl":"https://github.com/user/repo.git"}'

# Deploy
curl -X POST http://localhost:3001/api/projects/$PROJECT_ID/deploy \
  -H "Authorization: Bearer $TOKEN"
```

### Via Git Webhook

Point your GitHub/GitLab webhook to `http://your-server:3001/api/webhook/github`.

## Nidus Mail

Send transactional emails from your apps without external dependencies.

```bash
# Send via template
curl -X POST http://localhost:3001/api/mail/send \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "user@example.com",
    "template_id": "welcome",
    "vars": {"name": "Maria"}
  }'

# List templates
curl http://localhost:3001/api/mail/templates \
  -H "Authorization: Bearer $TOKEN"
```

### MCP Server

Connect AI agents to Nidus Mail:

```bash
NIDUS_API_URL=http://localhost:3001 NIDUS_API_TOKEN=$TOKEN node mcp/mail/server.js
```

## Managed Databases

```bash
# Create a database (provisioned instantly)
curl -X POST http://localhost:3001/api/databases \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"projectId":"$PROJECT_ID","name":"my-production-db"}'

# Returns PostgreSQL connection string + credentials
```

## Monitoring

- **API metrics:** `GET /api/metrics` — request stats, memory, uptime
- **Project metrics history:** `GET /api/projects/{id}/metrics/history` — CPU/Memory time-series
- **Grafana dashboards:** `http://localhost:3004` — 5 panels with per-container filtering
- **Prometheus targets:** cadvisor, nidus-api, node-exporter

## Structure

```
nidus/
├── apps/
│   ├── api/              # Go API (mail, metrics, auth, projects)
│   │   └── mail/         # Nidus Mail package
│   ├── dashboard/        # Next.js admin + charts
│   │   └── components/   # MetricsChart, etc.
│   └── landing-page/     # Static landing page
├── workers/
│   └── deploy/           # Go deploy worker (Docker build pipeline)
├── cli/                  # Node.js CLI
├── mcp/
│   └── mail/             # MCP server for AI-driven email
├── packages/
│   ├── shared/           # Shared TypeScript types
│   └── runtime/          # Deploy runtime engine
├── docker/
│   ├── nidus-monitoring.yml  # Prometheus + Grafana + cAdvisor
│   ├── prometheus.yml        # Metrics scraping config
│   └── grafana/              # Dashboard provisioning
├── docs-site/            # Documentation (Next.js, en/pt)
└── ecosystem.config.cjs  # PM2 process manager config
```

## Environment Variables

```bash
DATABASE_URL=postgresql://user:pass@localhost:5432/nidus
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-secret-key
# Optional: SMTP for Nidus Mail
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USER=apikey
SMTP_PASS=your-api-key
```

## Performance

- **Go API:** ~16MB RAM idle, ~2ms avg response, 4 goroutines
- **Go Worker:** ~14MB RAM idle, sub-second git clone (cache)
- **Docker builds:** layer caching, incremental deploys
- **Redis queue:** <1ms job dispatch

## Documentation

- [Quickstart](https://nimbus200.vercel.app/en/docs/quickstart)
- [Architecture](https://nimbus200.vercel.app/en/docs/architecture)
- [Deployment](https://nimbus200.vercel.app/en/docs/deployment)
- [API Reference](https://nimbus200.vercel.app/en/docs/api)
- [CLI Guide](https://nimbus200.vercel.app/en/docs/cli)
- [FAQ](https://nimbus200.vercel.app/en/docs/faq)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT — see [LICENSE](LICENSE).
