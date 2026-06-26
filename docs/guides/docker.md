# Docker

## Docker Compose

```yaml
services:
  postgres:
    image: postgres:16-alpine
    ports:
      - "5433:5432"
    environment:
      POSTGRES_USER: nidus
      POSTGRES_PASSWORD: nidus
      POSTGRES_DB: nidus

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  caddy:
    image: caddy:2-alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./docker/Caddyfile:/etc/caddy/Caddyfile
```

## Rodar

```bash
docker compose up -d
```

## Status

```bash
docker ps
```

## Logs

```bash
docker compose logs -f
```

## Parar

```bash
docker compose down
```
