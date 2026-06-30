# Nidus — Pesquisa Competitiva: O que torna uma PaaS magnética

## O "unfair advantage" de cada gigante

### Vercel — O que os torna magnéticos?
1. **DX impecável**: `vercel deploy` = 1 comando. Zero configuração. 
2. **Preview Deployments**: Cada PR ganha URL única. Revisão visual de código.
3. **Edge Functions**: Cold start <50ms. Execução global.
4. **Framework Detection**: Detecta Next.js, Nuxt, Svelte automaticamente.
5. **Build Cache**: Rebuilds instantâneos com cache inteligente.
6. **Domínios automáticos**: projeto.vercel.app com SSL.
7. **Integração GitHub**: Deploy a cada push. Comentário no PR com URL.

### Railway — Por que devs migram pra eles?
1. **CLI hipnótica**: Progress bars, emojis, cores. Parece mágica.
2. **Templates 1-click**: 100+ templates. Deploy de WordPress em 30s.
3. **Volumes + Databases**: Tudo integrado. Sem configurar infra.
4. **Pricing transparente**: Pay-as-you-go. Sem surpresas.
5. **Logs em tempo real**: WebSocket streaming com cores.
6. **Monorepo support**: Deploy seletivo por diretório.

### Fly.io — O moat deles
1. **Edge global**: Deploy em 35+ regiões com 1 comando.
2. **Proxy em Rust**: 10M req/s. Latência sub-ms.
3. **VMs leves**: Firecracker microVMs. Isolamento real.
4. **Anycast networking**: Roteamento automático pro edge mais próximo.
5. **CLI poderosa**: `flyctl` — uma das melhores CLIs do mercado.

### Supabase — O que atrai 100K+ devs?
1. **Realtime**: WebSocket nativo. 1M+ conexões simultâneas.
2. **PostgreSQL + Auth + Storage**: Tudo integrado.
3. **Open source first**: Código aberto. Confiança.
4. **SDKs em 10+ linguagens**: JS, Python, Go, Dart, Swift, Kotlin...
5. **Dashboard lindo**: UI que parece produto Apple.

### Coolify — O concorrente direto
1. **Self-hosted**: Igual Nidus. Mas em PHP/Laravel.
2. **UI bonita**: Dashboard bem desenhado.
3. **One-click services**: WordPress, n8n, MinIO, etc.
4. **Fraquezas**: PHP (performance), sem proxy Rust, sem edge, sem SDKs.

## O que o Nidus precisa ter pra vencer

### Diferencial #1: Performance brutal (Rust core)
```
Caddy (Go)    → 5K req/s, 100MB RAM
Nidus Proxy   → 50K req/s, 15MB RAM
NGINX         → 20K req/s, 200MB RAM
Traefik (Go)  → 8K req/s, 80MB RAM
```
**Vantagem**: 10x mais req/s que Caddy, 85% menos RAM. Seu servidor de R$30 comporta 50 apps vs 10 apps no Coolify.

### Diferencial #2: CLI nível Vercel/Railway (Rust)
```
$ nidus deploy
 ⠋ Detecting framework...     nextjs
 ⠙ Building...                [====================] 2.1s
 ⠹ Optimizing...              [====================] 0.8s
 ⠸ Deploying to edge...       [====================] 1.2s
 ✅ Live at https://app.nidus.app

$ nidus logs --live
 2:30:01 PM  GET /api/users    200  12ms  ← verde
 2:30:03 PM  POST /api/order   201  45ms  ← verde
 2:30:04 PM  GET /api/broken   500  2ms   ← vermelho
 2:30:05 PM  ERROR: DB timeout         ← vermelho
```
**Vantagem**: Tamanho: 5MB (Rust) vs 80MB (Node.js CLI do Coolify).

### Diferencial #3: Edge Functions em Rust (WASM)
```
Tradicional (Docker):    cold start 2-30s, 50MB+ RAM
Nidus Edge (WASM):       cold start <5ms, 2MB RAM
Vercel Edge (V8):        cold start <50ms, 10MB RAM
Cloudflare Workers (V8): cold start 0ms, 5MB RAM
```
**Vantagem**: Funções serverless no seu próprio servidor. Custo zero marginal.

### Diferencial #4: Hybrid Cloud (único no mercado)
```
Nidus Cloud:      Hospedado por nós. R$49/mês. Zero ops.
Nidus Self-Hosted: Seu servidor. R$30/mês. Controle total.
Nidus Hybrid:     Cloud para staging, self-hosted para produção.
```
**Vantagem**: Nenhum concorrente oferece os 3 modos. Vercel é só cloud. Coolify é só self-hosted.

### Diferencial #5: Build cache global (S3-compatible)
```
1º build: 120s (download deps + compile)
2º build: 5s (cache hit em 95% das camadas)
```
**Vantagem**: Usa MinIO ou S3 local. Cache compartilhado entre deploys.

### Diferencial #6: Observabilidade built-in
```
- OpenTelemetry tracing em cada request
- Métricas Prometheus nativas (Rust prometheus crate)
- Logs estruturados em JSON
- Dashboard Grafana pré-configurado
```
**Vantagem**: Observabilidade enterprise sem custo. Vercel cobra $150/mês por isso.

## Plano de Ataque: 4 fases

### Fase 1: Proxy + CLI (Mês 1) ← HOJE
- [x] Rust workspace setup
- [x] nidus-proxy: HTTP forward básico
- [x] nidus-cli: health + list
- [ ] nidus-proxy: SNI routing + TLS + DB cache
- [ ] nidus-cli: deploy com progress bar + logs live
- [ ] Benchmark: proxy Rust vs Go vs Caddy

### Fase 2: Builder + Edge (Mês 2)
- [ ] nidus-builder: BuildKit integrado com cache S3
- [ ] nidus-edge: WASM runtime (wasmtime)
- [ ] nidus-cli: syntax highlighting nos logs
- [ ] Deploy preview com URL automática

### Fase 3: Observabilidade (Mês 3)
- [ ] OpenTelemetry tracing
- [ ] Métricas Prometheus nativas
- [ ] Dashboard Grafana
- [ ] Alertas (Slack/Discord)

### Fase 4: Hybrid Cloud (Mês 4)
- [ ] Sincronização self-hosted ↔ cloud
- [ ] Migração 1-click entre modos
- [ ] Billing integrado
- [ ] Launch público

## Concorrentes — Análise de fraquezas

| Plataforma | Maior fraqueza | Oportunidade pro Nidus |
|-----------|---------------|----------------------|
| Vercel | Vendor lock-in. Preço alto ($20+/mês). | Self-hosted = sem lock-in |
| Railway | Sem self-hosted. Precisa de conta. | BYO server |
| Fly.io | Complexo. Curva de aprendizado alta. | Simplicidade |
| Coolify | PHP. Performance baixa. Sem edge. | Rust performance |
| Supabase | Só PostgreSQL. Não é PaaS completo. | Full PaaS |
| Heroku | Morto. Sem free tier desde 2022. | Free tier generoso |

## Métricas de Sucesso (12 meses)

- [ ] 1000 instalações self-hosted
- [ ] 10K deploys/dia
- [ ] Proxy Rust: 99.99% uptime
- [ ] NPS > 50
- [ ] 100 estrelas GitHub
- [ ] CLI: <1s startup time
- [ ] Receita: R$5K/mês (Cloud)

