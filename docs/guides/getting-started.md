# Getting Started

## Pré-requisitos

- Node.js 20+
- Docker (opcional)
- PostgreSQL (ou Docker)

## Instalação

```bash
git clone https://github.com/mateussiqueira/nidus.git
cd nidus
npm install
```

## Configuração

```bash
cp .env.example .env
```

Edite o `.env`:

```bash
DATABASE_URL=postgresql://user:pass@localhost:5432/nidus
REDIS_URL=redis://localhost:6379
NIDUS_HOST=localhost
JWT_SECRET=mude-isto
```

## Rodar

### Com Docker (recomendado)

```bash
docker compose up -d
```

### Sem Docker

```bash
# Terminal 1 - API
cd apps/control-plane
npm run build
npm start

# Terminal 2 - Dashboard
cd apps/dashboard
npm run dev
```

## Acessar

- **Dashboard:** http://localhost:3000
- **API:** http://localhost:3001
- **Docs:** http://localhost:3001/api/docs

## Deploy do primeiro app

```bash
# Via CLI
npx nidus login
cd meu-projeto
npx nidus deploy

# Via GitHub
Configure o webhook apontando pra http://seu-ip:3001/api/webhook
```
