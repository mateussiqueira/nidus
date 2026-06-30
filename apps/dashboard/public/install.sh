#!/usr/bin/env bash
set -euo pipefail

NIDUS_HOST=${NIDUS_HOST:-2.24.204.31}
NIDUS_API_PORT=${NIDUS_API_PORT:-3001}
NIDUS_DASH_PORT=${NIDUS_DASH_PORT:-3000}

RED="[1;31m"; GREEN="[1;32m"; YELLOW="[1;33m"; CYAN="[1;36m"; NC="[0m"
info()  { echo -e "${CYAN}"→"${NC} $1"; }
ok()    { echo -e "${GREEN}"✓"${NC} $1"; }
warn()  { echo -e "${YELLOW}"⚠"${NC} $1"; }

echo "  Nimbus 200 - Nidus PaaS"
echo "  Deploy, Host, Scale. Simple."

if ! command -v docker &>/dev/null; then
  echo "Instalando Docker..."
  curl -fsSL https://get.docker.com | bash
fi

if ! command -v psql &>/dev/null; then
  echo "Instalando PostgreSQL..."
  apt-get update -qq && apt-get install -y -qq postgresql postgresql-client
fi

if ! command -v redis-cli &>/dev/null; then
  echo "Instalando Redis..."
  apt-get install -y -qq redis-server
fi

echo "Instalacao concluida!"
echo "  Docker:       "$(docker --version 2>/dev/null || echo pendente)
echo "  PostgreSQL:   "$(psql --version 2>/dev/null || echo pendente)
echo "  Redis:        "$(redis-cli --version 2>/dev/null || echo pendente)
echo "  Dashboard:    http://${NIDUS_HOST}:${NIDUS_DASH_PORT}"
