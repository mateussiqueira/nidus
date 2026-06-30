#!/bin/bash
# StackRun — Daily database backup with retention
set -e

BACKUP_DIR="${BACKUP_DIR:-/root/stackrun/backups/db}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
DB_NAME="${DB_NAME:-nidus}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/stackrun_$TIMESTAMP.sql.gz"

mkdir -p "$BACKUP_DIR"

echo "[$(date +%H:%M:%S)] Starting backup..."

sudo -u postgres pg_dump "$DB_NAME" | gzip > "$BACKUP_FILE"

echo "[$(date +%H:%M:%S)] Backup created: $BACKUP_FILE ($(du -h "$BACKUP_FILE" | cut -f1))"

# Cleanup old backups
find "$BACKUP_DIR" -name "stackrun_*.sql.gz" -mtime +$RETENTION_DAYS -delete 2>/dev/null || true

# Keep only last 30 backups if retention is set
BACKUP_COUNT=$(ls -1 "$BACKUP_DIR"/stackrun_*.sql.gz 2>/dev/null | wc -l)
if [ "$BACKUP_COUNT" -gt 30 ]; then
    ls -1t "$BACKUP_DIR"/stackrun_*.sql.gz | tail -n +31 | xargs rm -f
    echo "[$(date +%H:%M:%S)] Cleaned old backups, keeping last 30"
fi

echo "[$(date +%H:%M:%S)] Done. $BACKUP_COUNT backups stored."
