---
layout: home
hero:
  name: StackRun
  text: PaaS self-hosted
  tagline: Deploy de apps web via Docker. Pense num Vercel que roda na sua máquina.
  actions:
    - theme: brand
      text: Começar
      link: /guides/getting-started
    - theme: alt
      text: GitHub
      link: https://github.com/mateussiqueira/stackrun
features:
  - title: Deploy via Git
    details: Conecta o GitHub, faz push, e ele builda + roda o container automaticamente.
  - title: CLI
    details: npx stackrun deploy. Simples assim.
  - title: Go Worker
    details: Deploy worker em Go. 10-50x mais rápido que Node.js.
  - title: Docker
    details: Cada app roda em container isolado. Sem conflitos.
  - title: SDKs (JS, Python, Go)
    details: Integração programática com SDKs oficiais para JavaScript, Python e Go.
  - title: Docker Compose
    details: Deploy stacks multi-container com um só YAML. WordPress + MySQL, MongoDB + Node, ou qualquer combinação.
  - title: Volumes Persistentes
    details: Dados que sobrevivem a redeploys. Banco de dados, uploads, arquivos de configuração.
  - title: Domínios Customizados
    details: Adicione seu próprio domínio com HTTPS automático via Caddy.
---

# StackRun

StackRun é um PaaS open-source inspirado em Vercel, Railway e Coolify.

A diferença? Ele roda na sua máquina. Ou no seu servidor. Sem depender de serviço externo.

## Como funciona

```
Seu código → Git push → StackRun detecta → Build Docker → Roda container → URL pronta
```

Em 30 segundos, seu app está no ar.

## Stack

- **Frontend:** Next.js 16 + Tailwind
- **API:** NestJS + Prisma + PostgreSQL
- **Proxy:** Caddy (HTTPS automático)
- **Deploy:** Docker containers isolados
- **Worker:** Go (performance)
- **CLI:** Node.js

## Quick start

```bash
# Com Docker
docker compose up -d

# Sem Docker
cd apps/control-plane
npm install && npm run build && npm start

# Em outro terminal
cd apps/dashboard
npm run dev
```

Acesse http://localhost:3000
