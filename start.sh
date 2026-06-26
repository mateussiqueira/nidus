#!/bin/bash
set -e

MODE=${1:-local}

echo "=== Nidus Platform ==="
echo "Modo: $MODE"
echo ""

case "$MODE" in
  docker)
    echo "Iniciando com Docker Compose..."
    docker compose up --build -d
    echo ""
    echo "Serviços:"
    echo "  API:     http://localhost:3001"
    echo "  Proxy:   http://localhost:3080"
    echo "  Postgres: localhost:5432"
    echo "  Redis:   localhost:6379"
    echo ""
    echo "Login: local@nidus.dev / local123"
    echo ""
    echo "Status:"
    docker compose ps
    ;;

  local)
    echo "Iniciando em modo local..."

    # Verificar PostgreSQL
    if ! pg_isready -h localhost -p 5432 -q 2>/dev/null; then
      echo "❌ PostgreSQL não está rodando. Inicie com: brew services start postgresql"
      exit 1
    fi

    # Verificar Redis
    if ! redis-cli -a nidus-redis-pass ping 2>/dev/null | grep -q PONG; then
      echo "❌ Redis não está rodando. Inicie com: brew services start redis"
      exit 1
    fi

    # Criar database se não existir
    psql -U broto -tc "SELECT 1 FROM pg_database WHERE datname='nidus'" | grep -q 1 || \
      psql -U broto -c "CREATE DATABASE nidus"

    # Push schema
    cd apps/control-plane
    npx prisma db push --skip-generate 2>/dev/null || true
    cd ../..

    # Seed user
    node apps/control-plane/prisma/seed.mjs 2>/dev/null || true

    cd apps/api
    DATABASE_URL="postgresql://broto@localhost:5432/nidus" \
    REDIS_URL="redis://:nidus-redis-pass@localhost:6379" \
    JWT_SECRET="local_nidus_jwt_secret_change_me" \
    ./nidus-api &
    API_PID=$!
    cd ../..

    cd workers/deploy
    DATABASE_URL="postgresql://broto@localhost:5432/nidus" \
    REDIS_URL="redis://:nidus-redis-pass@localhost:6379" \
    ./nidus-deploy-worker &
    WORKER_PID=$!
    cd ../..

    echo ""
    echo "Serviços iniciados:"
    echo "  API:     http://localhost:3001 (PID: $API_PID)"
    echo "  Worker:  rodando (PID: $WORKER_PID)"
    echo ""
    echo "Login: local@nidus.dev / local123"
    echo ""
    echo "Pressione Ctrl+C para parar"
    wait
    ;;
esac
