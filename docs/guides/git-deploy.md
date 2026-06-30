# Deploy via Git

## Como funciona

1. Você faz push pro GitHub
2. O GitHub envia um webhook pro StackRun
3. O StackRun clona o repo
4. Detecta o framework (Next.js, Nuxt, Vite, etc.)
5. Builda a imagem Docker
6. Roda o container
7. Retorna a URL

## Configurar webhook

No GitHub, vá em Settings → Webhooks → Add webhook:

- **Payload URL:** `http://seu-ip:3001/api/webhook`
- **Content type:** `application/json`
- **Events:** Just the push event

## Frameworks suportados

| Framework | Comando de build | Porta |
|-----------|------------------|-------|
| Next.js | `npm run build` | 3000 |
| Nuxt | `npm run build` | 3000 |
| Vite | `npm run build` | 80 |
| Angular | `npm run build` | 80 |
| Static | nginx | 80 |

## Preview deployments

Cada branch gera uma URL única:

```
main → meu-app.stackrun.localhost
feature-x → feature-x-meu-app.stackrun.localhost
```
