# Política de Segurança

## Versões Suportadas

| Versão | Suportada |
|--------|-----------|
| 0.2.x  | ✅        |
| 0.1.x  | ✅        |
| < 0.1  | ❌        |

## Reportando Vulnerabilidades

**NÃO** abra issues públicas para vulnerabilidades de segurança.

Envie um email para: **security@nidus.dev**

Inclua na mensagem:
- Descrição da vulnerabilidade
- Passos para reproduzir
- Potencial impacto
- Sugestão de correção (se tiver)

## Processo

1. **Recebemos** seu relatório
2. **Confirmamos** recebimento em 48 horas
3. **Investigamos** e validamos a vulnerabilidade
4. **Desenvolvemos** a correção
5. **Lançamos** patch de segurança
6. **Divulgamos** publicamente (após período de graça de 90 dias)

## Escopo

### Coberto
- Control Plane (API)
- Deploy Worker
- Data Plane (Proxy)
- Dashboard
- CLI

### Não Coberto
- Dependências de terceiros (reporte diretamente aos mantenedores)
- Infraestrutura de deploy (Vercel, Docker Hub, etc.)
- Issues que requerem acesso físico ao servidor

## Recomendações para Produção

1. **Altere** as credenciais padrão imediatamente
2. **Use** HTTPS em todos os endpoints
3. **Configure** firewall para limitar acesso
4. **Mantenha** o Nimbus atualizado
5. **Monitore** logs para atividade suspeita

## Agradecimentos

Agradecemos a todos que reportam vulnerabilidades de forma responsável. Seu contribuição ajuda a manter o Nimbus seguro para todos.
