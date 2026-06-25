# Nidus

Self-hosted deploy platform. Think Vercel that runs on your own machine.

## What is it

Nidus deploys web apps via Docker. Connect GitHub, push, and it builds + runs the container automatically.

**Still in beta** — works but needs polish.

## Stack

- **Frontend:** Next.js 16 + Tailwind
- **API:** NestJS + Prisma + PostgreSQL
- **Proxy:** Caddy (auto HTTPS)
- **Deploy:** Isolated Docker containers
- **CLI:** `npx nidus`

## Quick start

```bash
# With Docker (easiest)
docker compose up -d

# Without Docker
cd apps/control-plane
npm install
npm run build
npm start

# In another terminal
cd apps/dashboard
npm run dev
```

Open http://localhost:3000

## Deploy

```bash
# Via CLI
npx nidus login
cd my-project
npx nidus deploy

# Via GitHub
Set up webhook in GitHub pointing to http://your-ip:3001/api/webhook
```

## Environment variables

Copy `.env.example` to `.env` and adjust:

```bash
DATABASE_URL=postgresql://user:pass@localhost:5432/nidus
REDIS_URL=redis://localhost:6379
NIDUS_HOST=localhost
JWT_SECRET=change-this
```

## Structure

```
nidus/
├── apps/
│   ├── dashboard/        # Next.js
│   └── control-plane/    # NestJS API
├── workers/
│   └── deploy/           # Go worker (fast deploys)
├── cli/                  # CLI
├── packages/
│   ├── runtime/          # Deploy engine
│   └── shared/           # Shared types
└── docker/               # Caddyfile
```

## Performance

Deploy worker written in Go for speed:

- Git clone: ~0.5s (vs ~5s in Node)
- Docker build: ~25s with layer caching
- Memory: ~15MB (vs ~100MB+ in Node)

Run `./benchmark.sh` to see the numbers.

## License

MIT
