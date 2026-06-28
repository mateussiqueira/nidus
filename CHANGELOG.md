# Changelog

Todas as mudanças notáveis neste projeto serão documentadas neste arquivo.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/),
e este projeto adere ao [Semantic Versioning](https://semver.org/).

## [Não Lançado]

### Adicionado
- Testes unitários para Control Plane (auth, projects, deployments, cache, metrics)
- Documentação completa com sidebar e TOC
- Landing page com branding Nimbus
- Deploy Worker com suporte a repositórios locais
- Health check endpoint
- Metrics endpoint com formato Prometheus

### Modificado
- README atualizado com stack real (Go + Rust + NestJS + Next.js)
- CONTRIBUTING.md expandido com guia completo de desenvolvimento
- Deploy Worker: correção de git clone para repositórios locais
- Deploy Worker: sanitização de URLs com `file://`

### Corrigido
- Docker build com ImageTag vazio
- Git clone falhando para repositórios bare locais
- Checkout de arquivos após clone com `--depth`

## [0.2.0] - 2026-06-27

### Adicionado
- Data Plane em Rust (Axum) com reverse proxy
- Rate limiting com token bucket via Redis
- TLS automático via Caddy
- WebSocket proxy transparente
- Deploy Worker em Go com pool de goroutines
- CLI para deploy via terminal
- Dashboard com Next.js 16
- Health check endpoint
- Prometheus metrics

### Modificado
- Control Plane migrado de Express para NestJS
- ORM migrado de raw queries para Prisma
- Deploy queue usando BullMQ

## [0.1.0-beta] - 2026-06-25

### Adicionado
- Lançamento inicial beta
- Control Plane com NestJS
- Dashboard com Next.js
- Deploy via GitHub webhook
- PostgreSQL + Redis
- Docker Compose para desenvolvimento
- Suporte a múltiplos frameworks (Next.js, Nuxt, Vite, Angular)
