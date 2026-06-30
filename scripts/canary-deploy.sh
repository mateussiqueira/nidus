#!/bin/bash
# StackRun Canary Deploy — Progressive Traffic Shifting
set -e

PROJECT="${1}"
NEW_IMAGE="${2}"
CANARY_PCT="${3:-5}"
CANARY_STEP="${4:-25}"
CANARY_INTERVAL="${5:-30}"

echo "Canary deploy: $PROJECT → $NEW_IMAGE"
echo "Starting at ${CANARY_PCT}%, stepping ${CANARY_STEP}% every ${CANARY_INTERVAL}s"

for pct in $(seq $CANARY_PCT $CANARY_STEP 100); do
    if [ $pct -gt 100 ]; then pct=100; fi
    echo "  Traffic: canary ${pct}%, stable $((100-pct))%"
    # In real impl: update load balancer config
    sleep $CANARY_INTERVAL
done

echo "✅ Canary complete — 100% on new version"
