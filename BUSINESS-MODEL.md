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

## 💰 Planos Anuais (Mercado Brasileiro)

### Por que anual?
- 73% dos SaaS BR vendem plano anual (fonte: ABStartups)
- PIX à vista = sem taxa de cartão (economia de 3-5%)
- Cliente anual tem LTV 3x maior que mensal
- Churn de plano anual é 4x menor

### Planos Anuais — Preços Finais

| Plano | Mensal | **Anual** | Desconto | PIX (à vista) |
|-------|--------|-----------|----------|---------------|
| Cloud | R$49/mês | **R$497/ano** | 15% off | **R$447** (25% off) |
| Pro | R$99/mês | **R$997/ano** | 16% off | **R$897** (25% off) |
| Enterprise | R$299/mês | **R$2.997/ano** | 17% off | **R$2.697** (25% off) |

### Métodos de Pagamento (Brasil)

| Método | Taxa | Prazo | Preferência |
|--------|------|-------|-------------|
| **PIX** | 0% | Instantâneo | 65% dos BR |
| Cartão crédito | 3-5% | 30 dias | 25% |
| Boleto | R$3-5 | 3 dias | 10% |

### Projeção Anual (com PIX à vista)

**Cenário Conservador — Ano 1:**
```
Mês 1-3:  20 clientes Cloud  × R$447/ano = R$8.940  (entrada)
Mês 4-6:  80 clientes Cloud  × R$447/ano = R$35.760 (entrada)
Mês 7-9:  200 clientes Cloud × R$447/ano = R$89.400 (entrada)
Mês 10-12: 350 clientes Cloud × R$447/ano = R$156.450 (entrada)

Pro:  30 × R$897  = R$26.910
Ent:  5  × R$2.697 = R$13.485

RECEITA ANO 1: R$331.945
CUSTO ANO 1:  R$4.500 (3 VPS Hostinger × 12 meses)
LUCRO ANO 1:  R$327.445  (98.6% margem!)
```

**Cenário Realista — Ano 1:**
```
Total clientes: 650 Cloud + 50 Pro + 10 Enterprise
RECEITA: R$438.270
CUSTO:   R$6.600 (VPS Hostinger anual)
LUCRO:   R$431.670
```

### Estratégia de Venda (Brasil)

1. **Lançamento**: R$297/ano (50% off — early adopters)
2. **PIX**: Sempre oferecer desconto extra no PIX
3. **Garantia**: 7 dias de reembolso total
4. **Boleto**: Parcelamento em até 12x no cartão
5. **Indicação**: 1 mês grátis para quem indicar

### Gateways de Pagamento (Brasil)

| Gateway | PIX | Boleto | Cartão | Taxa |
|---------|-----|--------|--------|------|
| **Stripe** | ✅* | ❌ | ✅ | 3.99% |
| **Mercado Pago** | ✅ | ✅ | ✅ | 3.79% |
| **Pagar.me** | ✅ | ✅ | ✅ | 3.49% |
| **Efí (Gerencianet)** | ✅ | ✅ | ✅ | 0.99% (PIX) |

*Stripe PIX em beta no Brasil. Recomendação: **Efí** (menor taxa no PIX).

### VPS Hostinger — Custo Anual

| Plano | Mensal | **Anual (40% off)** | Clientes | Custo/cliente |
|-------|--------|---------------------|----------|---------------|
| KVM 2 (4 vCPU, 8GB) | R$55 | **R$396/ano** | 50 | R$7.92/cliente |
| KVM 4 (8 vCPU, 16GB) | R$105 | **R$756/ano** | 150 | R$5.04/cliente |
| KVM 8 (16 vCPU, 32GB) | R$205 | **R$1.476/ano** | 400 | R$3.69/cliente |

### ROI por VPS (plano anual KVM 4)
```
Investimento: R$756/ano (1 VPS)
Faturamento: 150 clientes × R$447 = R$67.050/ano
ROI: 88x
Payback: 2 clientes (R$894 cobre o VPS)
```


## 💳 Stack de Pagamento

### Brasil → AbacatePay
```
┌─────────────────────────────────────────┐
│              ABACATEPAY                  │
├─────────────────────────────────────────┤
│ PIX:        0.99%  (D+0)               │
│ Cartão:     2.99%  (D+30)              │
│ Boleto:     R$3.50 (D+3)               │
│ Sem mensalidade                        │
│ API REST simples                       │
│ Webhook de confirmação                 │
│ Split payment nativo                   │
└─────────────────────────────────────────┘
```

### Internacional → Stripe
```
┌─────────────────────────────────────────┐
│               STRIPE                     │
├─────────────────────────────────────────┤
│ Cartão:     2.9% + $0.30               │
│ PIX (beta): 0.99%                      │
│ 135+ moedas                            │
│ 47 países                              │
│ Billing automático                     │
│ Invoice + Subscription                 │
│ Dashboard analytics                    │
└─────────────────────────────────────────┘
```

### Fluxo de Pagamento

```
Cliente BR → AbacatePay → PIX/Cartão/Boleto → Confirmado
Cliente Global → Stripe → Cartão → Confirmado

StackRun API:
  POST /api/billing/checkout
  {
    "plan": "cloud",
    "country": "BR",        // auto-detect
    "paymentMethod": "pix"  // pix | card | boleto
  }
  → Redirect para AbacatePay (BR) ou Stripe (global)
  → Webhook confirma pagamento
  → Ativa assinatura no banco
```

### Por que AbacatePay + Stripe?

| Razão | Detalhe |
|-------|---------|
| **Taxa zero no PIX BR** | 0.99% vs 3.99% do Stripe |
| **Sem mensalidade** | AbacatePay não cobra fixo |
| **Stripe maduro** | 10+ anos, usado por toda startup |
| **Boleto nativo** | AbacatePay gera boleto registrado |
| **Split payment** | Futuro: afiliados ganham % |

### Integração Técnica

```
stackrun-api (Go)
  ├─ POST /api/billing/checkout  → cria cobrança AbacatePay/Stripe
  ├─ POST /api/billing/webhook   → recebe confirmação (HMAC validation)
  └─ GET  /api/billing/status    → status da assinatura

Tabelas novas:
  payments (id, user_id, plan_id, gateway, gateway_id, 
            amount_cents, status, payment_method, created_at)
```

