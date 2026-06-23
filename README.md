# Nidus

<p align="center">
  <a href="https://github.com/mateussiqueira/nidus"><img alt="GitHub" src="https://img.shields.io/github/stars/mateussiqueira/nidus?style=flat-square" /></a>
  <a href="https://github.com/mateussiqueira/canopy"><img alt="Built on Canopy" src="https://img.shields.io/badge/built%20on-Canopy-6C5CE7?style=flat-square" /></a>
</p>

PaaS open-source inspirada em Vercel, Railway e Coolify, com suporte nativo a Dart/Vaden.

> Fase atual: **Seedbox (alpha)** — deploy dos seus próprios projetos localmente e em cloud.

---

## Visão

Nidus é uma PaaS open-source para hospedar projetos full-stack:
- **Frontend**: Next.js, React, Vue, Flutter Web
- **Backend**: Node.js, Python, Go, Dart/Vaden
- **Serviços integrados**: Auth, PostgreSQL, Storage

Rode tudo localmente com Docker Compose ou suba em qualquer VPS com Docker Swarm/K3s.

---

## Estrutura

```
nidus/
├── apps/
│   ├── dashboard/          # Next.js — interface web
│   └── control-plane/      # NestJS — API de controle
├── packages/
│   ├── runtime/            # Motor de build e deploy
│   └── shared/             # Tipos e utilidades compartilhados
├── docker/                 # Dockerfiles e configurações
├── scripts/                # Scripts de setup e utilidades
├── docker-compose.yml      # Ambiente local completo
└── README.md
```

---

## Requisitos

- Docker + Docker Compose
- Node.js 22+
- Git

---

## Setup Local

```bash
# 1. Copie as variáveis de ambiente
cp .env.example .env

# 2. Suba a infraestrutura
docker compose up -d

# 3. Instale as dependências
npm install

# 4. Rode o dashboard e control-plane
npm run dev
```

Acesse:
- Dashboard: http://localhost:3000
- Control Plane API: http://localhost:3001
- Caddy Router: https://nidus.localhost (configurável via `CANOPY_ROOT_DOMAIN`)

---

## Deploy API

O Control Plane expõe endpoints REST para deploy de projetos:

| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/api/projects/:projectId/deploy` | Inicia um novo deploy |
| `GET`  | `/api/projects/:projectId/deployments` | Lista deploys do projeto |

O fluxo de deploy:
1. Clona ou atualiza o repositório Git do projeto
2. Faz build da imagem Docker com BuildKit
3. Inicia container com porta aleatória
4. Retorna URL de acesso e logs do build

Proteção via JWT — todas as requisições devem incluir `Authorization: Bearer <token>`.

---

## Infraestrutura Local

| Serviço | Porta | Finalidade |
|---------|-------|------------|
| PostgreSQL | `5433` (host) → `5432` (container) | Banco principal |
| Redis | `6379` | Cache e fila |
| MinIO API | `9000` | Storage de objetos |
| MinIO Console | `9001` | Admin de storage |
| Caddy HTTP | `80` | Proxy reverso |
| Caddy HTTPS | `443` | TLS |
| Caddy Admin | `2019` | Gerenciamento |
| Dashboard | `3000` | Interface web Next.js |
| Control Plane | `3001` | API NestJS |

---

## Funcionalidades da Fase 1 (Seedbox)

- [x] Criar projetos no dashboard
- [x] Conectar repositório Git (clone/pull automático)
- [x] Build com Docker (BuildKit)
- [x] Deploy com URL local (porta dinâmica)
- [ ] Banco PostgreSQL por projeto
- [ ] Auth básico com GoTrue

---

## Stack

| Camada | Tecnologia |
|--------|-----------|
| Dashboard | Next.js 15 + Tailwind |
| Control Plane | NestJS + Prisma |
| Runtime | Docker + BuildKit |
| Router | Caddy |
| Auth | GoTrue |
| Database | PostgreSQL 16 |
| Storage | MinIO |
| Cache/Queue | Redis |

---

## Roadmap

1. **Seedbox** — deploy dos próprios projetos
2. **Plataforma** — Appwrite + Vercel unificados
3. **Cloud** — versão comercial multi-tenant

---

## Licença

MIT
