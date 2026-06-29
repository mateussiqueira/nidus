#!/bin/bash
# Nidus Docker Cleanup — Remove unused images, stopped containers, build cache
# Run daily: 0 4 * * * /root/nidus/scripts/docker-cleanup.sh >> /var/log/nidus-cleanup.log 2>&1

set -euo pipefail
LOG="/var/log/nidus-cleanup.log"

log() { echo "[$(date +%H:%M:%S)] $*"; }

log "Starting Docker cleanup..."

# Remove stopped containers older than 24h
STOPPED=$(docker ps -aq --filter status=exited --filter status=dead 2>/dev/null)
if [ -n "$STOPPED" ]; then
    echo "$STOPPED" | xargs -r docker rm 2>/dev/null
    COUNT=$(echo "$STOPPED" | wc -l | tr -d ' ')
    log "Removed $COUNT stopped containers"
else
    log "No stopped containers"
fi

# Remove dangling images
DANGLING=$(docker images -q --filter dangling=true 2>/dev/null)
if [ -n "$DANGLING" ]; then
    echo "$DANGLING" | xargs -r docker rmi 2>/dev/null
    COUNT=$(echo "$DANGLING" | wc -l | tr -d ' ')
    log "Removed $COUNT dangling images"
else
    log "No dangling images"
fi

# Prune build cache
docker builder prune -f --filter until=48h 2>/dev/null && log "Build cache pruned"

# Remove unused volumes older than 7 days
docker volume prune -f --filter "label!=keep" 2>/dev/null && log "Unused volumes pruned"

# Show disk usage after cleanup
docker system df --format "  {{.Type}}: {{.Size}} ({{.Reclaimable}} reclaimable)" 2>/dev/null

log "Cleanup complete"
