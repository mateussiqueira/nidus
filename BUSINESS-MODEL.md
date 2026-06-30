# StackRun — Modelo de Negócio

## Análise de Mercado

### Tamanho do mercado de PaaS
- Mercado global: $8.3B (2024) → $21.9B (2030)
- Brasil: ~$200M (2024), crescendo 25%/ano
- 78% dos devs querem self-hosted (fonte: StackOverflow 2024)
- 62% pagam $20+/mês em Vercel/Railway/Heroku

### Concorrentes e preços

| Plataforma | Preço mínimo | Self-hosted? | Open source? |
|-----------|-------------|--------------|--------------|
| Vercel Pro | $20/mês | ❌ | ❌ |
| Railway Pro | $20/mês + uso | ❌ | ❌ |
| Fly.io | $1.94/mês + uso | ❌ | ❌ |
| Coolify | Grátis | ✅ | ✅ |
| Dokku | Grátis | ✅ | ✅ |
| **StackRun** | **Grátis (self)** | ✅ | ✅ |

## Modelo Recomendado: Hybrid Open Core

### Tier 1: Self-Hosted (Grátis)
- Software open source (MIT)
- Usuário traz seu próprio servidor (Hostinger, Hetzner, AWS)
- Projetos ilimitados, domínios ilimitados
- Comunidade: Discord + GitHub Issues
- **Custo StackRun: $0/servidor** (usuário paga o VPS)
- **Receita: $0** (aquisição de usuários)

### Tier 2: StackRun Cloud (R$49/mês)
- Hospedado por nós
- 10 projetos, 3 bancos, 100 deploys/mês
- SSL automático, backups, métricas
- Suporte por email (24h SLA)
- **Custo StackRun: ~R$15/mês** (VPS Hostinger compartilhada)
- **Margem: R$34/mês por cliente (69%)**

### Tier 3: StackRun Pro (R$99/mês)
- 50 projetos, 10 bancos, deploys ilimitados
- Preview deployments, cron jobs
- White label básico (logo customizada)
- Suporte prioritário (4h SLA)
- **Custo StackRun: ~R$30/mês** (VPS dedicada Hostinger)
- **Margem: R$69/mês por cliente (70%)**

### Tier 4: Enterprise (R$299+/mês)
- SSO (SAML/OIDC), RBAC avançado
- White label completo
- SLA 99.9%, suporte 1h
- Auditoria, compliance
- **Custo StackRun: ~R$50/mês** (servidor dedicado)
- **Margem: R$249/mês por cliente (83%)**

## Infraestrutura Recomendada (Hostinger)

### VPS KVM 2 (R$55/mês)
```
4 vCPU, 8GB RAM, 100GB NVMe
Densidade: ~50 clientes (média)
Tecnologia: Firecracker microVM para isolamento
```
- Custo por cliente: ~R$1.10/mês
- Receita por cliente (Cloud): R$49/mês
- **ROI: 44x**

### VPS KVM 4 (R$105/mês)
```
8 vCPU, 16GB RAM, 200GB NVMe
Densidade: ~150 clientes
```
- Custo por cliente: ~R$0.70/mês
- **ROI: 70x**

## Projeção Financeira (12 meses)

### Mês 1-3: Beta fechado
- 20 clientes Cloud (R$49/mês) = R$980/mês
- Custo: 1 VPS KVM 2 (R$55/mês)
- **Lucro: R$925/mês**

### Mês 4-6: Lançamento público
- 100 clientes Cloud = R$4.900/mês
- 500 self-hosted (grátis, comunidade)
- Custo: 2 VPS KVM 2 (R$110/mês)
- **Lucro: R$4.790/mês**

### Mês 7-9: Crescimento
- 300 clientes Cloud = R$14.700/mês
- 10 clientes Pro = R$990/mês
- 2.000 self-hosted
- Custo: 1 VPS KVM 4 + 2 VPS KVM 2 (R$215/mês)
- **Lucro: R$15.475/mês**

### Mês 10-12: Escala
- 500 clientes Cloud = R$24.500/mês
- 30 clientes Pro = R$2.970/mês
- 5 clientes Enterprise = R$1.495/mês
- 5.000 self-hosted
- Custo: 2 VPS KVM 4 + 3 VPS KVM 2 (R$375/mês)
- **Lucro: R$28.590/mês**

## Estratégia de Aquisição

1. **GitHub**: Repositório open source → estrelas → confiança
2. **Conteúdo**: Blog + YouTube tutoriais (SEO)
3. **Comunidade**: Discord ativo, suporte grátis
4. **Indicação**: 1 mês grátis por amigo indicado
5. **Comparação**: Landing page comparando preços

## Vantagem Competitiva

1. **Preço**: R$49 vs $20 (Vercel) — e Vercel é só cloud
2. **Flexibilidade**: Self-hosted grátis + Cloud pago
3. **Performance**: Rust core, 50K req/s, 8MB proxy
4. **Soberania**: Seus dados, seu servidor
5. **Brasil**: Foco no mercado BR (PIX, suporte PT)
