#!/bin/bash
# StackRun Container Health Check — Auto-restart dead containers
# Run via cron every 5 minutes: */5 * * * * /root/stackrun/scripts/health-check.sh

set -euo pipefail
LOG="/var/log/stackrun-health.log"

log() { echo "[$(date +%H:%M:%S)] $*" >> "$LOG"; }

# Check all stackrun containers
docker ps -a --format '{{.Names}} {{.Status}}' --filter name=stackrun- | while read -r name status; do
    if echo "$status" | grep -qi "unhealthy\|dead\|restarting"; then
        log "WARN: Container $name unhealthy ($status), restarting..."
        docker restart "$name" 2>/dev/null && log "OK: $name restarted" || log "ERR: Failed to restart $name"
    elif echo "$status" | grep -qi "exited\|Exited"; then
        exit_code=$(docker inspect -f '{{.State.ExitCode}}' "$name" 2>/dev/null || echo "?")
        log "INFO: Container $name exited with code $exit_code, starting..."
        docker start "$name" 2>/dev/null && log "OK: $name started" || docker restart "$name" 2>/dev/null
    fi
done

# Check service health
for svc in 3001 3000 8081; do
    if ! curl -sf http://localhost:$svc/health >/dev/null 2>&1; then
        log "WARN: Port $svc not responding"
    fi
done
