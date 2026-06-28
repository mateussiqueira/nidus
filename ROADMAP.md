# Roadmap do Nimbus

## Visão Geral

O Nimbus está em fase beta (v0.2.0). O objetivo é estabilizar para v1.0 até o final de 2026.

## Q3 2026 — Estabilização

### Core
- [ ] 90% de cobertura de testes no Control Plane
- [ ] Testes unitários para Go Deploy Worker
- [ ] Testes unitários para Rust Proxy
- [ ] Fix do deploy via API local (git clone issue)
- [ ] Rollback de deploys

### Docs
- [ ] Documentação completa com 15+ páginas
- [ ] Guia de troubleshooting
- [ ] Exemplos para cada framework suportado
- [ ] API reference completa

### DX
- [ ] Hot reload no Dashboard
- [ ] Logs em tempo real no Dashboard
- [ ] Métricas de uso no Dashboard

## Q4 2026 — v1.0

### Features
- [ ] Deploy via CLI com progresso em tempo real
- [ ] Variables de ambiente criptografadas
- [ ] Domínios customizados com SSL automático
- [ ] Rollback com um clique
- [ ] Preview deployments (branch-based)

### Infra
- [ ] Suporte a múltiplos servidores (cluster mode)
- [ ] Load balancing entre containers
- [ ] Backup automático do banco
- [ ] Monitoring com Grafana/Prometheus

### Community
- [ ] Template marketplace
- [ ] Plugin system
- [ ] Discord community
- [ ] Blog com updates

## 2027 — Expansão

### Enterprise
- [ ] SSO/SAML
- [ ] Audit logs
- [ ] RBAC (Role-Based Access Control)
- [ ] SLA 99.9%

### Multi-cloud
- [ ] Deploy para AWS ECS
- [ ] Deploy para Google Cloud Run
- [ ] Deploy para Azure Container Apps
- [ ] Deploy para Kubernetes

### Mobile
- [ ] Dashboard mobile (PWA)
- [ ] Notificações push
- [ ] Deploy via app mobile

## Como Contribuir

Veja [CONTRIBUTING.md](CONTRIBUTING.md) para como contribuir com qualquer uma dessas features.
