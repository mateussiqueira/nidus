# Variáveis de Ambiente

## Obrigatórias

| Variável | Descrição | Exemplo |
|----------|-----------|---------|
| `DATABASE_URL` | URL do PostgreSQL | `postgresql://user:pass@localhost:5432/nidus` |
| `JWT_SECRET` | Segredo para JWT | `mude-isto-por-algo-seguro` |
| `NIDUS_HOST` | Host do servidor | `localhost` ou `2.24.204.31` |

## Opcionais

| Variável | Descrição | Default |
|----------|-----------|---------|
| `REDIS_URL` | URL do Redis | `redis://localhost:6379` |
| `CORS_ORIGINS` | Origens permitidas | `http://localhost:3000` |
| `API_PORT` | Porta da API | `3001` |
| `NIDUS_DEPLOYS_DIR` | Diretório de deploys | `/tmp/nidus-deploys` |

## Exemplo `.env`

```bash
# Database
DATABASE_URL=postgresql://nidus:nidus@localhost:5433/nidus

# Redis
REDIS_URL=redis://localhost:6379

# Server
NIDUS_HOST=localhost
API_PORT=3001
CORS_ORIGINS=http://localhost:3000

# Security
JWT_SECRET=seu-segredo-aqui

# Deploy
NIDUS_DEPLOYS_DIR=/tmp/nidus-deploys
```
