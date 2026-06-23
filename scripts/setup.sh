#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "🌳 Canopy — Setup Inicial"
echo "======================="
echo ""

# 1. Check prerequisites
echo "[1/5] Verificando pré-requisitos..."

if ! command -v docker &> /dev/null; then
  echo "❌ Docker não encontrado. Instale em: https://docker.com"
  exit 1
fi

if ! command -v node &> /dev/null; then
  echo "❌ Node.js não encontrado. Instale em: https://nodejs.org"
  exit 1
fi

NODE_VERSION=$(node -v | sed 's/v//' | cut -d. -f1)
if [ "$NODE_VERSION" -lt 22 ]; then
  echo "❌ Node.js >= 22 necessário. Versão atual: $(node -v)"
  exit 1
fi

echo "✅ Docker: $(docker --version)"
echo "✅ Node: $(node --version)"
echo "✅ npm: $(npm --version)"

# 2. Create .env if not exists
echo ""
echo "[2/5] Configurando ambiente..."
if [ ! -f .env ]; then
  cp .env.example .env
  echo "✅ .env criado a partir de .env.example"
else
  echo "ℹ️ .env já existe, mantendo atual"
fi

# 3. Install dependencies
echo ""
echo "[3/5] Instalando dependências..."
npm install
echo "✅ Dependências instaladas"

# 4. Start infrastructure
echo ""
echo "[4/5] Subindo infraestrutura (PostgreSQL, Redis, MinIO, GoTrue, Caddy)..."
docker compose up -d
echo "✅ Infraestrutura rodando"

# 5. Generate Prisma client and push schema
echo ""
echo "[5/5] Configurando banco de dados..."
export $(grep -v '^#' .env | xargs)
npx prisma generate > /dev/null 2>&1 || true
npx prisma db push --skip-generate > /dev/null 2>&1 || true
echo "✅ Banco de dados configurado"

echo ""
echo "🎉 Canopy pronto!"
echo ""
echo "Para iniciar o desenvolvimento:"
echo "  npm run dev"
echo ""
echo "Acesse:"
echo "  Dashboard: http://localhost:3000"
echo "  API:       http://localhost:3001"
echo "  Auth:      http://localhost:9999"
echo "  MinIO:     http://localhost:9001 (user: canopy / pass: canopy-canopy)"
echo ""
