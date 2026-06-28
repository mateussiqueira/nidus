#!/bin/bash
# Nidus Backup Script — PostgreSQL daily dump
# Usage: ./scripts/backup.sh [output-dir]

set -euo pipefail

BACKUP_DIR="${1:-/root/nidus/backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=7

DB_HOST="${PGHOST:-localhost}"
DB_PORT="${PGPORT:-5432}"
DB_NAME="${PGDATABASE:-nidus}"
DB_USER="${PGUSER:-nidus}"
DB_PASS="${PGPASSWORD:-nidus_prod_123}"
BACKUP_FILE="${BACKUP_DIR}/nidus_${TIMESTAMP}.sql.gz"
LATEST_LINK="${BACKUP_DIR}/nidus_latest.sql.gz"

mkdir -p "$BACKUP_DIR"

export PGPASSWORD="$DB_PASS"

echo "[$(date +%H:%M:%S)] Starting backup: $DB_NAME@$DB_HOST:$DB_PORT"
echo "[$(date +%H:%M:%S)] Output: $BACKUP_FILE"

pg_dump \
  -h "$DB_HOST" \
  -p "$DB_PORT" \
  -U "$DB_USER" \
  -d "$DB_NAME" \
  --no-owner \
  --no-acl \
  --clean \
  --if-exists \
  | gzip > "$BACKUP_FILE"

# Create/update latest symlink
ln -sf "$BACKUP_FILE" "$LATEST_LINK"

BACKUP_SIZE=$(du -h "$BACKUP_FILE" | cut -f1)
echo "[$(date +%H:%M:%S)] Backup complete: $BACKUP_SIZE"

# Rotate old backups (keep RETENTION_DAYS)
find "$BACKUP_DIR" -name "nidus_*.sql.gz" -type f -mtime +$RETENTION_DAYS -delete
echo "[$(date +%H:%M:%S)] Cleaned backups older than ${RETENTION_DAYS}d"

# Verify backup
if gzip -t "$BACKUP_FILE" 2>/dev/null; then
    echo "[$(date +%H:%M:%S)] Backup integrity verified"
else
    echo "[$(date +%H:%M:%S)] ERROR: Backup integrity check failed"
    rm -f "$BACKUP_FILE"
    exit 1
fi

echo "[$(date +%H:%M:%S)] All done"
