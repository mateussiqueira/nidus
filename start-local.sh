#!/bin/bash

# Nidus Local Server Startup Script
# This script starts both the API (control-plane) and Dashboard

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "🚀 Starting Nidus Local Server..."
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
  echo "⚠️  Docker is not running. Deploys will not work without Docker."
  echo "   Please start Docker Desktop and try again."
  echo ""
fi

# Check if PostgreSQL is running
if ! pg_isready -h localhost -p 5432 > /dev/null 2>&1; then
  echo "❌ PostgreSQL is not running on port 5432."
  echo "   Please start PostgreSQL and try again."
  exit 1
fi

echo "✅ PostgreSQL is running"
echo ""

# Load environment variables
export $(grep -v '^#' .env | xargs)

# Start API in background
echo "📡 Starting API (control-plane) on port ${API_PORT:-3001}..."
cd apps/control-plane
npm run dev &
API_PID=$!
cd ../..

# Wait for API to start
sleep 3

# Start Dashboard
echo "🖥️  Starting Dashboard on port ${PORT:-3000}..."
cd apps/dashboard
npm run dev &
DASHBOARD_PID=$!
cd ../..

echo ""
echo "═══════════════════════════════════════════════════"
echo "  Nidus Local Server is running!"
echo "═══════════════════════════════════════════════════"
echo ""
echo "  Dashboard:  http://localhost:${PORT:-3000}"
echo "  API:        http://localhost:${API_PORT:-3001}"
echo ""
echo "  Login:      local@nidus.dev / local123"
echo ""
echo "  Press Ctrl+C to stop all services"
echo "═══════════════════════════════════════════════════"
echo ""

# Handle cleanup on exit
cleanup() {
  echo ""
  echo "🛑 Stopping services..."
  kill $API_PID 2>/dev/null || true
  kill $DASHBOARD_PID 2>/dev/null || true
  echo "✅ All services stopped"
  exit 0
}

trap cleanup SIGINT SIGTERM

# Wait for both processes
wait
