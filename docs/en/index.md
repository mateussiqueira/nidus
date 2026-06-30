---
layout: home
hero:
  name: StackRun
  text: Self-hosted PaaS
  tagline: Deploy web apps via Docker. Think Vercel that runs on your own machine.
  actions:
    - theme: brand
      text: Get Started
      link: /en/guides/getting-started
    - theme: alt
      text: GitHub
      link: https://github.com/mateussiqueira/stackrun
features:
  - title: Git Deploy
    details: Connect GitHub, push, and it builds + runs the container automatically.
  - title: CLI
    details: npx stackrun deploy. Simple as that.
  - title: Go Worker
    details: Deploy worker in Go. 10-50x faster than Node.js.
  - title: Docker
    details: Each app runs in an isolated container. No conflicts.
---

# StackRun

StackRun is an open-source PaaS inspired by Vercel, Railway, and Coolify.

The difference? It runs on your machine. Or your server. No external service dependency.

## How it works

```
Your code → Git push → StackRun detects → Docker build → Run container → URL ready
```

In 30 seconds, your app is live.

## Stack

- **Frontend:** Next.js 16 + Tailwind
- **API:** NestJS + Prisma + PostgreSQL
- **Proxy:** Caddy (auto HTTPS)
- **Deploy:** Isolated Docker containers
- **Worker:** Go (performance)
- **CLI:** Node.js

## Quick start

```bash
# With Docker
docker compose up -d

# Without Docker
cd apps/control-plane
npm install && npm run build && npm start

# In another terminal
cd apps/dashboard
npm run dev
```

Open http://localhost:3000
