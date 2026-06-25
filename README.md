# Nidus

Plataforma de deploy self-hosted. Pense num Vercel que roda na sua máquina.

## O que é

Nidus faz deploy de apps web via Docker. Você conecta o GitHub, faz push, e ele builda + roda o container automaticamente.

**Ainda em beta** — funciona mas falta polish.

## Stack

- **Frontend:** Next.js 16 + Tailwind
- **API:** NestJS + Prisma + PostgreSQL
- **Proxy:** Caddy (HTTPS automático)
- **Deploy:** Docker containers isolados
- **CLI:** `npx nidus`

## Como rodar

```bash
# Com Docker (mais fácil)
docker compose up -d

# Sem Docker
cd apps/control-plane
npm install
npm run build
npm start

# Em outro terminal
cd apps/dashboard
npm run dev
```

Abre http://localhost:3000

## Deploy

```bash
# Via CLI
npx nidus login
cd meu-projeto
npx nidus deploy

# Via GitHub
Configura o webhook no GitHub apontando pra http://seu-ip:3001/api/webhook
```

## Variáveis de ambiente

Copia o `.env.example` pra `.env` e ajusta:

```bash
DATABASE_URL=postgresql://user:pass@localhost:5432/nidus
REDIS_URL=redis://localhost:6379
NIDUS_HOST=localhost
JWT_SECRET=mude-isto
```

## Estrutura

```
nidus/
├── apps/
│   ├── dashboard/        # Next.js
│   └── control-plane/    # NestJS API
├── workers/
│   └── deploy/           # Worker Go (deploy rápido)
├── cli/                  # CLI
├── packages/
│   ├── runtime/          # Engine de deploy
│   └── shared/           # Tipos compartilhados
└── docker/               # Caddyfile
```

## Performance

O deploy worker foi escrito em Go por performance:

- Git clone: ~0.5s (vs ~5s em Node)
- Docker build: ~25s com layer caching
- Memória: ~15MB (vs ~100MB+ em Node)

Roda `./benchmark.sh` pra ver os números.

## License

MIT
