═══════════════════════════════════════════════════════════════════════
              NIDUS — ROADMAP 12 MESES (Jul 2026 – Jun 2027)
                 Da seedbox ao PaaS competitivo
═══════════════════════════════════════════════════════════════════════

OBJETIVO: Transformar o Nidus de um "funciona mas é bruto" em um PaaS
que compete em UX com Railway, em leveza com Fly.io, e em custo com
"VPS de R$30 + software grátis".

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
MÊS 1-2: FUNDAÇÃO SÓLIDA (Jul-Ago 2026)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Meta: Experiência de primeiro deploy em < 60 segundos.
       Nidus instalável e funcional sem conhecimento prévio.

SEMANA 1-2: ONBOARDING MÁGICO
  ☐ Script install.sh 1-liner:
       curl -sSL https://nidus.app/install.sh | bash
       → Detecta OS, instala Docker/Go/Redis se ausente
       → Sobe Nidus com docker-compose em 1 comando
       → Gera senha admin e imprime URL de acesso

  ☐ CLI tool (`nidus`):
       nidus deploy          → deploy do diretório atual
       nidus logs            → streaming de logs
       nidus projects        → lista projetos
       nidus env set KEY VAL → configura env vars
       nidus db create       → provisiona banco

  ☐ Template "Hello World" automático:
       Ao criar projeto sem repo, oferecer:
       - Next.js starter (com Dockerfile.nidus)
       - Express API starter
       - Vaden/Dart starter
       - Static HTML starter
       → Gerar repo local, commit inicial, deploy imediato

SEMANA 3-4: DASHBOARD COM DADOS REAIS
  ☐ SSR com dados via getServerSideProps/generateMetadata
       → /dashboard/projects mostra cards com métricas reais
       → /dashboard/projects/[id] carrega dados no servidor
       → Sem "tela branca até JS carregar"

  ☐ Logs de deploy em tempo real:
       → WebSocket funcional e estável
       → Logs com scroll infinito e syntax highlighting
       → Botão "copiar logs"

  ☐ Progresso de deploy visual:
       → Barra de progresso: clone → build → push → run
       → Tempo estimado por etapa
       → Notificação toast quando conclui

  ☐ Página de projeto refatorada:
       → Métricas reais (CPU/RAM do contêiner)
       → Gráfico de histórico de deploys
       → Status do último deploy visível

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
MÊS 3-4: ECOSSISTEMA DE DEPLOY (Set-Out 2026)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Meta: Paridade de features com Railway/Render.

SEMANA 5-6: DNS & SSL AUTOMÁTICO
  ☐ Domínios automáticos no deploy:
       → projeto.nidus.app (wildcard DNS)
       → SSL via Caddy auto Let's Encrypt
       → Custom domains com TLS automático

  ☐ Painel de domínios:
       → Verificar DNS propagation
       → Status SSL em tempo real
       → Redirect HTTP → HTTPS

  ☐ Health checks reais:
       → HTTP GET /health a cada 30s
       → Métrica de uptime por projeto
       → Notificação se cair
       → Auto-restart com backoff

SEMANA 7-8: BANCOS DE DADOS GERENCIADOS
  ☐ Provisionamento 1-click:
       → PostgreSQL, Redis, MySQL/MariaDB
       → Credenciais automáticas injetadas como env vars
       → Conexão segura (rede Docker interna)

  ☐ Backups automáticos:
       → pg_dump diário para S3 ou disco local
       → Retenção configurável (7/30/90 dias)
       → Restore com 1 clique

  ☐ Métricas de banco:
       → Conexões ativas, queries/segundo
       → Tamanho do banco, crescimento
       → Slow query log visível

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
MÊS 5-6: ECOSSISTEMA DE COLABORAÇÃO (Nov-Dez 2026)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Meta: Times podem usar Nidus em produção.

SEMANA 9-10: GIT INTEGRATION PROFUNDA
  ☐ GitHub OAuth:
       → Login com GitHub (além do email/senha)
       → Importar repositórios com 1 clique
       → Lista de repos do usuário

  ☐ Deploy automático por push:
       → Webhook endpoint seguro com HMAC
       → Deploy na branch principal → produção
       → Deploy em outras branches → preview
       → Comentário no PR com URL do preview

  ☐ Preview deployments:
       → URL única por PR: pr-42.projeto.nidus.app
       → Isolamento: cada preview em container separado
       → Auto-cleanup ao fechar PR
       → Badge de status no README

SEMANA 11-12: RBAC & SEGURANÇA
  ☐ Sistema de roles:
       → Owner, Admin, Developer, Viewer
       → Permissões granulares por projeto
       → Audit log de ações

  ☐ Segurança de deploy:
       → Scan de imagem (Trivy)
       → Secrets nunca em logs
       → Rate limiting por IP
       → 2FA para contas admin

  ☐ Variáveis de ambiente por ambiente:
       → Production, Staging, Preview
       → Secrets criptografados (AES-256)
       → Bulk edit/import .env files

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
MÊS 7-8: PLATAFORMA AVANÇADA (Jan-Fev 2027)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Meta: Nidus como plataforma, não só ferramenta.

SEMANA 13-14: API PÚBLICA & WEBHOOKS
  ☐ API REST documentada (OpenAPI/Swagger):
       → Tokens de API por projeto/usuário
       → Rate limiting por token
       → SDKs: JavaScript, Python, Go, Dart

  ☐ Webhooks de eventos:
       → deploy.started, deploy.completed, deploy.failed
       → Integração com Slack, Discord, Telegram
       → Payload configurável

  ☐ GitHub Actions oficial:
       → nidus-deploy-action
       → Configuração zero (detecta projeto)

SEMANA 15-16: DOCKER COMPOSE & MULTI-SERVICE
  ☐ Suporte a docker-compose.yml:
       → Deploy de stacks multi-container
       → Rede interna entre serviços
       → Volumes persistentes

  ☐ Escalonamento horizontal:
       → Réplicas por serviço
       → Load balancing entre réplicas
       → Auto-scale baseado em CPU/memória

  ☐ Cron jobs:
       → Agendamento via expressão cron
       → Logs de execução
       → Notificação de falha

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
MÊS 9-10: ECOSSISTEMA & MERCADO (Mar-Abr 2027)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Meta: Nidus é conhecido e adotado por devs LATAM/BR.

SEMANA 17-18: MARKETPLACE DE TEMPLATES
  ☐ Galeria de templates 1-click:
       → Next.js, Nuxt, SvelteKit, Astro
       → Express, Fastify, Hono, Elysia
       → Vaden/Dart, Laravel, Django
       → WordPress, Ghost, Strapi
       → n8n, Supabase, PocketBase
       → Comunidade pode submeter

  ☐ Template YAML schema:
       → name, description, icon, tags
       → docker-compose.yml ou Dockerfile.nidus
       → env vars padrão, ports, volumes
       → Validação automática no submit

SEMANA 19-20: LANDING PAGE & DOCS
  ☐ Landing page profissional:
       → Animação de produto (não só estática)
       → Demo interativa (terminal simulado)
       → Comparativo de preços interativo
       → Depoimentos/cases

  ☐ Documentação completa:
       → docs.nidus.app (Docusaurus/Starlight)
       → Getting started em 5 minutos
       → Guias por framework
       → API reference
       → Troubleshooting/FAQ
       → Vídeos tutoriais (YouTube)

  ☐ Blog técnico:
       → "Por que migrei do Vercel pro Nidus"
       → "Deploy de apps Dart/Vaden em produção"
       → "Hospedando 20 projetos por R$30/mês"

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
MÊS 11-12: ESCALA & MONETIZAÇÃO (Mai-Jun 2027)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Meta: Nidus é sustentável financeiramente.

SEMANA 21-22: NIDUS CLOUD (HOSTED)
  ☐ Versão gerenciada (SaaS):
       → nidus.app — plataforma hospedada
       → Planos: Free (1 projeto), Pro (R$49/mês), Team (R$99/mês)
       → Infra na Hostinger/Hetzner BR
       → Boleto/PIX para pagamento

  ☐ Migração self-hosted → cloud:
       → Export/import de projetos
       → Ferramenta de migração automática
       → Zero downtime na transição

SEMANA 23-24: ENTERPRISE & COMUNIDADE
  ☐ Enterprise features:
       → SSO (SAML/OIDC)
       → Audit logs avançados
       → SLA 99.9%
       → Suporte dedicado
       → White label (logo customizada)

  ☐ Comunidade:
       → Discord ativo (1000+ membros)
       → Programa de embaixadores
       → Hackathons trimestrais
       → Contribuidores no GitHub (50+)

  ☐ Programa de parceiros:
       → Agências certificadas Nidus
       → Comissão por indicação
       → Templates oficiais de parceiros

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
MÉTRICAS DE SUCESSO (12 MESES)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

MÊS 3:
  ☐ 50 instalações self-hosted ativas
  ☐ NPS ≥ 30 (early adopters)
  ☐ Tempo de 1º deploy < 2 minutos

MÊS 6:
  ☐ 200 instalações ativas
  ☐ 10 contribuidores no GitHub
  ☐ 500 estrelas no GitHub
  ☐ Primeiro caso de uso em produção (não-nosso)

MÊS 9:
  ☐ 500 instalações ativas
  ☐ Nidus Cloud beta com 20 clientes
  ☐ 5 templates da comunidade
  ☐ 1 talk em conferência (TDC, RubyConf, etc)

MÊS 12:
  ☐ 1000+ instalações
  ☐ Nidus Cloud com 100 clientes pagantes
  ☐ Receita: R$5.000-10.000/mês
  ☐ Time de 2-3 pessoas
  ☐ Presença em 3+ conferências LATAM

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
PRIORIDADES ABSOLUTAS (PRÓXIMOS 30 DIAS)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. ⬛ Script install.sh 1-liner
2. ⬛ Template Hello World automático
3. ⬛ Dashboard SSR com dados reais
4. ⬛ Log streaming funcional
5. ⬛ Health check HTTP (não só docker inspect)

═══════════════════════════════════════════════════════════════════════
