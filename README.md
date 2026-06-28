# Nimbus

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.2.0-orange.svg)](https://github.com/mateussiqueira/nidus/releases)
[![CI](https://github.com/mateussiqueira/nidus/actions/workflows/ci.yml/badge.svg)](https://github.com/mateussiqueira/nidus/actions)

> Deploy de aplicações self-hosted sem dor de cabeça. De um Raspberry Pi a um VPS potente.

## O que é o Nimbus?

O Nimbus é uma plataforma de deploy self-hosted inspirada no Vercel. Conecta seu GitHub, faz push, e ele builda + roda seu app em containers Docker isolados. Tudo no seu servidor, sem lock-in.

## Stack

| Componente | Tecnologia | Função |
|------------|-----------|--------|
| **API** | **Go 1.25** | API REST, auth, webhooks (~30MB RAM) |
| **Deploy Worker** | **Go 1.25** | Build e deploy de containers (~20MB RAM) |
| **Data Plane** | **Rust (Axum)** | Reverse proxy de alta performance |
| **Dashboard** | Next.js 16 + React 19 | Interface do usuário |
| **CLI** | Node.js/TypeScript | Deploy via terminal |
| **Banco** | PostgreSQL 16 / SQLite | Dados (SQLite para lite) |
| **Cache** | Redis 7 / In-memory | Filas e cache |
| **Proxy** | Caddy | HTTPS automático |

## Pré-requisitos

| Versão | RAM Mínima | Disco | Uso |
|--------|-----------|-------|-----|
| **Lite** | 512MB | 2GB | API + Worker apenas |
| **Completa** | 2GB | 10GB | Todos os componentes |

**Requisitos obrigatórios:**
- Docker + Docker Compose
- Git
- Conexão com internet

## Quick Start

### Versão Completa (2GB+ RAM)

```bash
git clone https://github.com/mateussiqueira/nidus.git
cd nidus
cp .env.example .env
docker compose up -d
```

### Versão Lite (512MB RAM)

```bash
git clone https://github.com/mateussiqueira/nidus.git
cd nidus
docker compose -f docker-compose.lite.yml up -d
```

A versão lite usa SQLite em vez de PostgreSQL e não inclui Dashboard/Redis/Proxy.

### Acesse

- **Dashboard:** http://localhost:3000
- **API:** http://localhost:3001
- **Proxy:** http://localhost:3080

### Credenciais padrão

- Email: `demo@nidus.dev`
- Senha: `demo123`

## Deploy

### Via CLI

```bash
npm install -g nidus-cli
nidus login
cd meu-projeto
nidus deploy
```

### Via GitHub

Configure um webhook apontando para `http://seu-servidor:3001/api/webhook`.

## Estrutura

```
nidus/
├── apps/
│   ├── control-plane/    # NestJS API
│   ├── dashboard/        # Next.js frontend
│   ├── api/              # Go API (alternativa)
│   └── proxy/            # Rust reverse proxy
├── workers/
│   └── deploy/           # Go deploy worker
├── cli/                  # CLI Node.js
├── packages/
│   ├── shared/           # Tipos compartilhados
│   └── runtime/          # Engine de deploy
├── docker/               # Configurações Docker
├── docs/                 # Documentação
└── docs-site/            # Site da documentação
```

## Variáveis de Ambiente

Copie `.env.example` para `.env`:

```bash
DATABASE_URL=postgresql://user:pass@localhost:5432/nidus
REDIS_URL=redis://localhost:6379
JWT_SECRET=sua-chave-secreta
```

## Performance

O Deploy Worker em Go oferece:

- **10x mais rápido** que Node.js para git clone
- **~15MB** de memória em idle
- **~0.5s** para git clone
- **Cache de camadas** Docker para builds incrementais

Rode `./benchmark.sh` para ver os números.

## Documentação

- [Primeiros Passos](https://nimbus200.vercel.app/pt/docs/quickstart)
- [Arquitetura](https://nimbus200.vercel.app/pt/docs/architecture)
- [Deploy](https://nimbus200.vercel.app/pt/docs/deployment)
- [API](https://nimbus200.vercel.app/pt/docs/api)
- [CLI](https://nimbus200.vercel.app/pt/docs/cli)
- [FAQ](https://nimbus200.vercel.app/pt/docs/faq)

## Contribuindo

Veja [CONTRIBUTING.md](CONTRIBUTING.md) para detalhes.

## Licença

MIT License - veja [LICENSE](LICENSE) para detalhes.
