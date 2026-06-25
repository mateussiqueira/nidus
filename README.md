# Nidus

PaaS open-source inspirada em Vercel, Railway e Coolify, com suporte nativo a Dart/Vaden.

> Beta — deploy real com Docker, métricas, webhook, CLI e app macOS nativo.

## Stack

| Camada | Tecnologia |
|--------|-----------|
| Dashboard | Next.js 16 + Tailwind 4 |
| API | NestJS + Prisma 7 + PostgreSQL |
| Runtime | Docker + BuildKit |
| Proxy | Caddy (auto HTTPS) |
| Auth | JWT + bcrypt |
| Deploy | Docker containers isolados |
| CLI | Node.js (`npx nidus`) |
| App | SwiftUI (macOS 14+) |

## Funcionalidades

### Dashboard
- Login/Cadastro com rate limit
- Criar projeto com template (Next.js, Vaden, Express, Static)
- Deploy em 1 clique com logs completos
- Métricas do container (CPU, RAM, uptime)
- Environment variables editáveis
- Git webhook (auto-deploy via push GitHub)
- Histórico de deploys

### CLI
```bash
npx nidus login          # autenticar
npx nidus deploy         # deploy do diretório atual
npx nidus projects       # listar projetos
```

### App macOS Nativo
```bash
cd app/Nidus && swift run
```
- Chat IA multimodal (texto, imagem, áudio) via OpenRouter
- Sidebar com projetos + deploys + métricas
- Modelos: Claude, GPT, Gemini, DeepSeek, Mistral

## Deploy Rápido

```bash
npm install -g nidus
nidus login              # email + senha
cd meu-projeto
nidus deploy             # deploy automático
```

Ou acesse: **http://2.24.204.31:3000**

## Setup Local

```bash
# Terminal 1 - API
cd apps/control-plane
DATABASE_URL='postgresql://broto:broto@localhost:5432/nidus?schema=public' \
NIDUS_DEPLOYS_DIR=/tmp/nidus-deploys \
node dist/main.js

# Terminal 2 - Dashboard
cd apps/dashboard
npx next dev -p 3000
```

Acesse: http://localhost:3000 | Login: `local@nidus.dev` / `local123`

## Estrutura

```
nidus/
├── apps/dashboard/          # Next.js
├── apps/control-plane/      # NestJS API
├── app/Nidus/               # macOS app (SwiftUI)
├── cli/                     # CLI (npx nidus)
├── packages/runtime/        # Deploy engine
├── packages/shared/         # Tipos
└── docker/                  # Dockerfiles
```

## Roadmap

- [x] Deploy via Git push (webhook GitHub)
- [x] CLI (`npx nidus deploy`)
- [x] App macOS nativo (SwiftUI multimodal)
- [x] Métricas de container (CPU, RAM)
- [x] Env vars por projeto
- [ ] CLI no npm (`npm i -g nidus`)
- [ ] Preview deployments (branch → URL)
- [ ] Domínios customizados
- [ ] Template Vaden (Dart backend)
- [ ] Integração Nidus (open code fork)

## Licença MIT
