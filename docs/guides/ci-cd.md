# CI/CD com GitHub Actions

Deploy automático a cada push no repositório.

## GitHub Action

Crie `.github/workflows/deploy.yml` no seu repositório:

```yaml
name: Deploy to StackRun

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Deploy
        run: |
          npx @stackrun/cli deploy \
            --token ${{ secrets.STACKRUN_TOKEN }} \
            --project my-app
```

Configure `STACKRUN_TOKEN` nos secrets do repositório em
**Settings → Secrets and variables → Actions**.

## Webhook manual

Como alternativa, use o webhook direto:

1. Vá em **Dashboard → Projeto → Settings → Webhook**
2. Copie a URL do webhook
3. Adicione ao GitHub em **Settings → Webhooks → Add webhook**

O StackRun recebe o payload e dispara o deploy automaticamente quando detecta
push na branch configurada. Sem precisar de Action ou script extra.
