# Getting Started

## Prerequisites

- Node.js 20+
- Docker (optional)
- PostgreSQL (or Docker)

## Installation

```bash
git clone https://github.com/mateussiqueira/nidus.git
cd nidus
npm install
```

## Configuration

```bash
cp .env.example .env
```

Edit `.env`:

```bash
DATABASE_URL=postgresql://user:pass@localhost:5432/nidus
REDIS_URL=redis://localhost:6379
NIDUS_HOST=localhost
JWT_SECRET=change-this
```

## Run

### With Docker (recommended)

```bash
docker compose up -d
```

### Without Docker

```bash
# Terminal 1 - API
cd apps/control-plane
npm run build
npm start

# Terminal 2 - Dashboard
cd apps/dashboard
npm run dev
```

## Access

- **Dashboard:** http://localhost:3000
- **API:** http://localhost:3001
- **Docs:** http://localhost:3001/api/docs

## Deploy your first app

```bash
# Via CLI
npx nidus login
cd my-project
npx nidus deploy

# Via GitHub
Configure webhook pointing to http://your-ip:3001/api/webhook
```
