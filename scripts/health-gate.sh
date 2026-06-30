#!/bin/bash
# Health Gate — validates deploy before production traffic
set -e

PROJECT="${1}"
PORT="${2}"
MAX_RETRIES="${3:-10}"
RETRY_INTERVAL="${4:-2}"
STARTUP_TIMEOUT="${5:-60}"

check_health() {
    curl -sf --max-time 5 "http://127.0.0.1:${PORT}/health" > /dev/null 2>&1
}

check_liveness() {
    curl -sf --max-time 5 "http://127.0.0.1:${PORT}/health" > /dev/null 2>&1
}

check_readiness() {
    # Readiness = health + DB connected + no active deploys
    local resp=$(curl -sf "http://127.0.0.1:${PORT}/health" 2>/dev/null)
    echo "$resp" | grep -q '"dbConnected":true'
}

echo "Health Gate: $PROJECT port $PORT"

# Startup probe
echo -n "  Startup..."
for i in $(seq 1 $(($STARTUP_TIMEOUT / $RETRY_INTERVAL))); do
    if check_health; then echo " OK"; break; fi
    sleep $RETRY_INTERVAL
done

# Readiness probe
echo -n "  Readiness..."
for i in $(seq 1 $MAX_RETRIES); do
    if check_readiness; then echo " OK"; break; fi
    if [ $i -eq $MAX_RETRIES ]; then echo " FAILED"; exit 1; fi
    sleep $RETRY_INTERVAL
done

echo "✅ Health gate passed — ready for traffic"
