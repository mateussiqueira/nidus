# Nidus

PaaS open-source inspirada em Vercel, Railway e Coolify, com suporte nativo a Dart/Vaden.

> Fase atual: **Seedbox** — deploy dos seus próprios projetos localmente e em cloud.

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
canopy/
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
- Caddy Router: https://canopy.localhost

---

## Funcionalidades da Fase 1 (Seedbox)

- [x] Criar projetos no dashboard
- [ ] Conectar repositório Git
- [ ] Detectar stack automaticamente
- [ ] Build com Docker
- [ ] Deploy com URL local
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
