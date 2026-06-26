# REST API

## Base URL

```
http://localhost:3001
```

## Authentication

```bash
# Login
POST /api/auth/login
{
  "email": "admin@nidus.local",
  "password": "senha123"
}

# Response
{
  "token": "jwt-token-here"
}
```

## Projects

```bash
# List projects
GET /api/projects
Authorization: Bearer <token>

# Create project
POST /api/projects
{
  "name": "my-app",
  "repoUrl": "https://github.com/user/repo"
}

# Delete project
DELETE /api/projects/:id
```

## Deployments

```bash
# List deployments
GET /api/projects/:id/deployments

# Create deployment
POST /api/projects/:id/deploy?branch=main

# Deployment status
GET /api/projects/:id/deployments/:deploymentId
```

## Metrics

```bash
# Container metrics
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
