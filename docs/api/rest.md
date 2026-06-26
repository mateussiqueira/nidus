# API REST

## Base URL

```
http://localhost:3001
```

## Autenticação

```bash
# Login
POST /api/auth/login
{
  "email": "admin@nidus.local",
  "password": "senha123"
}

# Response
{
  "token": "jwt-token-aqui"
}
```

## Projetos

```bash
# Listar projetos
GET /api/projects
Authorization: Bearer <token>

# Criar projeto
POST /api/projects
{
  "name": "meu-app",
  "repoUrl": "https://github.com/user/repo"
}

# Deletar projeto
DELETE /api/projects/:id
```

## Deploys

```bash
# Listar deploys
GET /api/projects/:id/deployments

# Criar deploy
POST /api/projects/:id/deploy?branch=main

# Status do deploy
GET /api/projects/:id/deployments/:deploymentId
```

## Métricas

```bash
# Métricas do container
GET /api/projects/:id/metrics

# Response
{
  "status": "running",
  "cpu": 2.5,
  "memory": {
    "usage": "128MB",
    "limit": "512MB",
    "percent": 25
  }
}
```

## Webhook

```bash
# GitHub webhook
POST /api/webhook
{
  "ref": "refs/heads/main",
  "repository": {
    "full_name": "user/repo"
  }
}
```
