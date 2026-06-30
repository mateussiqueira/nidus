# Volumes Persistentes

Dados que sobrevivem a redeploys e reinicializações de container.

## Criar volume

### Pela API

```bash
curl -X POST https://stackrun.vercel.app/api/volumes \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "meu-volume", "size": 10}'
```

### Pelo Dashboard

**Dashboard → Projeto → Volumes → Criar Volume**

Defina nome e tamanho (em GB). O volume fica disponível imediatamente.

## Usar no docker-compose

```yaml
services:
  app:
    image: nginx
    volumes:
      - meu-volume:/usr/share/nginx/html

volumes:
  meu-volume:
    external: true
```

Declare o volume como `external: true` para usar um volume criado pelo StackRun.
O dado persiste mesmo se o container for removido ou recriado.
