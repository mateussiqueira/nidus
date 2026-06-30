# StackRun Deploy Worker v2 - Configuration Guide

## Environment Variables

### Performance Tuning
```bash
# Maximum concurrent builds (0-100, default: 50)
# Higher = more parallelism but more memory usage
MAX_CONCURRENT_BUILDS=50

# Docker build timeout in seconds (default: 120)
DOCKER_TIMEOUT_SECONDS=120

# Enable BuildKit for better caching and performance
BUILDKIT_ENABLED=true

# Memory limit per container in bytes (default: 512m)
DOCKER_MEMORY_LIMIT=536870912  # 512MB
```

### Database Configuration
```bash
# PostgreSQL connection (production)
DATABASE_URL=postgresql://user:pass@postgres:5432/nidus

# SQLite connection (Lite edition)
DATABASE_URL=sqlite:///data/nidus.db
```

### Cache & Queue
```bash
# Redis connection for job queue
REDIS_URL=redis://:password@redis:6379/0

# If not set, worker uses in-process channel queue
```

### Storage
```bash
# Directory for deploys and artifacts
STACKRUN_DEPLOYS_DIR=/tmp/stackrun-deploys

# Host for container URL mapping
STACKRUN_HOST=localhost
```

### Server
```bash
# Health check and metrics port
WORKER_PORT=8081
```

## Docker Compose Examples

### Production (2GB+ RAM)
```yaml
deploy-worker:
  image: stackrun:deploy-worker-v2
  environment:
    MAX_CONCURRENT_BUILDS: 50
    DOCKER_TIMEOUT_SECONDS: 120
    BUILDKIT_ENABLED: "true"
    DATABASE_URL: postgresql://user:pass@postgres:5432/nidus
    REDIS_URL: redis://redis:6379/0
    STACKRUN_DEPLOYS_DIR: /data/deploys
    STACKRUN_HOST: deploy-worker
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock
    - deploy-data:/data
  resources:
    limits:
      cpus: "4.0"
      memory: 1G
    reservations:
      cpus: "2.0"
      memory: 512M
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:8081/health"]
    interval: 30s
    timeout: 3s
    retries: 3
    start_period: 10s
```

### Lite Edition (512MB RAM)
```yaml
deploy-worker:
  image: stackrun:deploy-worker-v2
  environment:
    MAX_CONCURRENT_BUILDS: 2
    DOCKER_TIMEOUT_SECONDS: 60
    BUILDKIT_ENABLED: "false"
    DATABASE_URL: sqlite:///data/nidus.db
    # No Redis - uses in-process queue
    STACKRUN_DEPLOYS_DIR: /data/deploys
    STACKRUN_HOST: localhost
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock
    - deploy-data:/data
  resources:
    limits:
      cpus: "0.5"
      memory: 256M
    reservations:
      cpus: "0.25"
      memory: 128M
```

## Monitoring

### Health Check
```bash
curl http://localhost:8081/health
```

Response:
```json
{
  "status": "ok",
  "db": true,
  "redis": true,
  "workers": 4,
  "active": 2,
  "cpu": 4,
  "maxConcurrent": 50
}
```

### Metrics (Prometheus)
```bash
curl http://localhost:8081/metrics
```

Key metrics:
- `stackrun_deploys_total` - Total deploys by status
- `stackrun_deploy_duration_seconds` - Deploy duration histogram
- `stackrun_deploy_active` - Currently processing deploys
- `stackrun_build_duration_seconds` - Docker build duration
- `stackrun_git_duration_seconds` - Git operations duration
- `stackrun_build_cache_hits_total` - Docker cache hits
- `stackrun_build_cache_misses_total` - Docker cache misses

## Performance Tuning Tips

### For Maximum Throughput (Production)
```bash
MAX_CONCURRENT_BUILDS=100
DOCKER_TIMEOUT_SECONDS=300
BUILDKIT_ENABLED=true
# Enable Redis for distributed queue
REDIS_URL=redis://redis:6379/0
```

### For Minimum Memory (Raspberry Pi)
```bash
MAX_CONCURRENT_BUILDS=2
DOCKER_TIMEOUT_SECONDS=60
BUILDKIT_ENABLED=false
# No Redis - uses in-process queue
# Use SQLite instead of PostgreSQL
DATABASE_URL=sqlite:///data/nidus.db
```

### Balanced (1-2GB servers)
```bash
MAX_CONCURRENT_BUILDS=20
DOCKER_TIMEOUT_SECONDS=120
BUILDKIT_ENABLED=true
REDIS_URL=redis://redis:6379/0
DATABASE_URL=postgresql://user:pass@postgres:5432/nidus
```

## Troubleshooting

### High Memory Usage
1. Reduce `MAX_CONCURRENT_BUILDS`
2. Lower Docker build timeout
3. Enable BuildKit for better layer caching
4. Monitor with `docker stats`

### Slow Builds
1. Enable BuildKit: `BUILDKIT_ENABLED=true`
2. Increase Docker timeout
3. Check Redis connectivity (logs)
4. Verify Docker daemon isn't resource-constrained

### Failed Deployments
1. Check `/metrics` for cache hit ratio
2. Review health check: `curl http://localhost:8081/health`
3. Inspect logs for specific framework issues
4. Verify environment variables are correct
