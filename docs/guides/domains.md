# Domínios Customizados

Adicione seu próprio domínio aos projetos.

## Adicionar domínio

1. Vá em **Dashboard → Domínios → Adicionar**
2. Aponte o DNS do seu domínio para **2.24.204.31**
3. Clique em **Verificar**

Após a verificação, seu projeto passa a responder no domínio customizado.

## Subdomínio automático

Todo projeto ganha automaticamente um subdomínio no formato:

```
slug-do-projeto.nidus.app
```

Esse subdomínio fica disponível imediatamente após o deploy, sem nenhuma configuração.

## SSL

Certificados TLS são provisionados automaticamente via Caddy.

Zero configuração — o Caddy detecta o domínio, gera o certificado via Let's Encrypt
e mantém a renovação automática. HTTPS sempre ativo.
