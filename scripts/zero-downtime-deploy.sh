#!/bin/bash
# StackRun Zero-Downtime Deploy — Blue/Green Strategy
set -e

PROJECT="${1:?Usage: $0 <project-slug> <new-image>}"
NEW_IMAGE="${2:?}"
HEALTH_URL="${3:-/health}"
HEALTH_RETRIES="${4:-10}"
HEALTH_INTERVAL="${5:-2}"
DRAIN_SECONDS="${6:-30}"
BLUE_PORT=$(docker port "stackrun-${PROJECT}" 2>/dev/null | head -1 | cut -d: -f2)
GREEN_PORT=$((BLUE_PORT + 1))
NETWORK="${7:-nidus}"

echo "═══════════════════════════════════"
echo " Zero-Downtime Deploy: $PROJECT"
echo " Blue (active):  port $BLUE_PORT"
echo " Green (new):    port $GREEN_PORT"
echo " Image:          $NEW_IMAGE"
echo "═══════════════════════════════════"

# 1. Start green container
echo ""
echo "[1/5] Starting green container..."
docker run -d --rm \
    --name "stackrun-${PROJECT}-green" \
    --network "$NETWORK" \
    --label "stackrun.project=$PROJECT" \
    --label "stackrun.color=green" \
    -p "$GREEN_PORT:3000" \
    "$NEW_IMAGE"

# 2. Health check green
echo "[2/5] Health checking green..."
for i in $(seq 1 $HEALTH_RETRIES); do
    if curl -sf "http://127.0.0.1:$GREEN_PORT$HEALTH_URL" > /dev/null 2>&1; then
        echo "  ✓ Health check $i/$HEALTH_RETRIES passed"
        break
    fi
    if [ $i -eq $HEALTH_RETRIES ]; then
        echo "  ✗ Health check failed after $HEALTH_RETRIES attempts"
        docker stop "stackrun-${PROJECT}-green"
        exit 1
    fi
    sleep $HEALTH_INTERVAL
done

# 3. Switch traffic via proxy port update
echo "[3/5] Switching traffic..."
if [ -S /tmp/stackrun-proxy.sock ]; then
    echo "db.stackrun_proxy.update_port('$PROJECT', $GREEN_PORT)" | nc -U /tmp/stackrun-proxy.sock
fi
echo "  ✓ Traffic → port $GREEN_PORT"

# 4. Drain connections from blue
echo "[4/5] Draining connections ($DRAIN_SECONDS s)..."
sleep $DRAIN_SECONDS

# 5. Stop blue
echo "[5/5] Stopping blue container..."
docker stop "stackrun-${PROJECT}" 2>/dev/null || true
docker rename "stackrun-${PROJECT}-green" "stackrun-${PROJECT}"

echo ""
echo "═══════════════════════════════════"
echo " ✅ Deploy complete!"
echo "    $PROJECT → port $GREEN_PORT"
echo "═══════════════════════════════════"
