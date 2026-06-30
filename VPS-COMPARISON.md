# StackRun — Comparativo de VPS (Junho 2026)

## Objetivo
Achar a VPS com melhor custo-benefício para hospedar o StackRun Cloud (SaaS).
Requisito mínimo: 4 vCPU, 8GB RAM, 100GB NVMe, KVM (para Firecracker).

## Comparativo

| Provedor | Plano | vCPU | RAM | Disco | R$/mês | R$/ano | R$/GB RAM |
|----------|-------|------|-----|-------|--------|--------|-----------|
| **Hostinger** | KVM 2 | 4 | 8GB | 100GB | R$55 | R$396 | R$6.87 |
| **Hostinger** | KVM 4 | 8 | 16GB | 200GB | R$105 | R$756 | R$6.56 |
| **Hostinger** | KVM 8 | 16 | 32GB | 400GB | R$205 | R$1.476 | R$6.40 |
| **Hetzner** | CX32 | 4 | 8GB | 80GB | €7.50 (R$47) | €90 (R$564) | R$5.87 |
| **Hetzner** | CX42 | 8 | 16GB | 160GB | €15 (R$94) | €180 (R$1.128) | R$5.87 |
| **NetCup** | RS 4000 | 6 | 16GB | 320GB | €10.50 (R$66) | €126 (R$792) | R$4.12 |
| **Contabo** | Cloud VPS L | 6 | 16GB | 200GB | €10.99 (R$69) | €132 (R$828) | R$4.31 |
| **Locaweb** | VPS Pro 8 | 4 | 8GB | 120GB | R$279 | R$3.348 | R$34.87 |
| **AWS Lightsail** | 8GB | 2 | 8GB | 160GB | $40 (R$220) | $480 (R$2.640) | R$27.50 |
| **DigitalOcean** | Premium AMD | 4 | 8GB | 160GB | $48 (R$264) | $576 (R$3.168) | R$33.00 |

## 🏆 TOP 3 — Melhor custo-benefício

### 1. 🥇 NetCup RS 4000 — R$66/mês (R$792/ano)
```
6 vCPU, 16GB RAM, 320GB NVMe
R$4.12/GB RAM — o mais barato por GB
Datacenter: Alemanha (~200ms latência BR)
Pontos fortes: preço imbatível, muito disco
Pontos fracos: latência alta pro Brasil, suporte em alemão
```

### 2. 🥈 Hetzner CX32 — R$47/mês (R$564/ano)
```
4 vCPU, 8GB RAM, 80GB NVMe
R$5.87/GB RAM
Datacenter: Alemanha/Finlândia (~200ms latência BR)
Pontos fortes: confiável, API excelente, provisão instantânea
Pontos fracos: latência alta pro Brasil, precisa de VPN/proxy
```

### 3. 🥉 Hostinger KVM 2 — R$55/mês (R$396/ano com 40% off)
```
4 vCPU, 8GB RAM, 100GB NVMe
R$6.87/GB RAM
Datacenter: Brasil (São Paulo) — ~5ms latência!
Pontos fortes: latência baixíssima, suporte PT-BR, PIX, nota fiscal
Pontos fracos: preço maior que Hetzner, API menos madura
```

## 🎯 Recomendação para StackRun

### Estratégia Híbrida (melhor dos 2 mundos):

| Camada | Provedor | Plano | Preço | Justificativa |
|--------|----------|------|-------|---------------|
| **API + DB** | Hostinger BR | KVM 4 | R$756/ano | Latência <5ms, suporte PT |
| **Edge/Proxy** | Hetzner | CX32 | R$564/ano | Preço baixo, escala global |
| **Build Worker** | NetCup | RS 4000 | R$792/ano | Muito RAM pra builds |
| **Staging/Dev** | Hostinger | KVM 1 | R$156/ano | Testes, CI/CD |

### Custo total: R$2.268/ano (R$189/mês)

### Ou: Tudo na Hostinger (simplicidade):

| Serviço | Plano | Preço anual |
|---------|-------|-------------|
| Produção | KVM 8 (16 vCPU, 32GB) | R$1.476/ano |
| Staging | KVM 2 (4 vCPU, 8GB) | R$396/ano |
| Backup | KVM 1 (2 vCPU, 4GB) | R$156/ano |
| **Total** | | **R$2.028/ano (R$169/mês)** |

## 📊 Projeção: 1 VPS KVM 8 na Hostinger

```
Custo: R$1.476/ano (R$123/mês)
Capacidade: 16 vCPU, 32GB RAM, 400GB NVMe
Densidade: ~400 clientes Cloud (StackRun)

Receita: 400 × R$447/ano = R$178.800/ano
Custo VPS: R$1.476/ano
Lucro: R$177.324/ano (99.2% margem!)
ROI: 120x
```

## ✅ Decisão final

**Hostinger KVM 8** é a melhor escolha para começar:
- Latência <5ms pro Brasil (95% dos clientes iniciais)
- PIX, boleto, nota fiscal brasileira
- Suporte em português (crítico em incidentes)
- 40% de desconto no plano anual
- 1 VPS comporta 400 clientes nos primeiros 12 meses

Quando escalar internacionalmente: adicionar Hetzner/NetCup na Europa.

