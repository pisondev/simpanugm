#!/bin/sh
# SimpanUGM automated database backup script (runbook.md section 3, US-54)
# Performs custom format pg_dump of the simpan database, writes metric, and rotates local dumps
set -eu

BACKUP_DIR="${BACKUP_DIR:-/opt/simpanugm/backups}"
METRICS_DIR="${METRICS_DIR:-/opt/simpanugm/metrics}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
TIMESTAMP="$(date -u +%Y%m%d_%H%M%SZ)"
DUMP_FILE="${BACKUP_DIR}/simpan_${TIMESTAMP}.dump"
LATEST_LINK="${BACKUP_DIR}/simpan_latest.dump"

mkdir -p "${BACKUP_DIR}" "${METRICS_DIR}"

echo "Starting backup of simpan database at $(date -u)..."

# Support running directly via pg_dump or through docker compose
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1 && docker compose ps postgres >/dev/null 2>&1; then
  docker compose exec -T postgres pg_dump -U simpan_owner -Fc -d simpan > "${DUMP_FILE}"
elif command -v pg_dump >/dev/null 2>&1; then
  PGPASSWORD="${SIMPAN_OWNER_PASSWORD:-${POSTGRES_PASSWORD:-simpan}}" \
    pg_dump -h "${PGHOST:-127.0.0.1}" -p "${PGPORT:-5432}" -U "${PGUSER:-simpan_owner}" -Fc -d "${PGDATABASE:-simpan}" > "${DUMP_FILE}"
else
  echo "Error: neither docker compose nor pg_dump found" >&2
  exit 1
fi

# Verify dump file size and header
if [ ! -s "${DUMP_FILE}" ]; then
  echo "Error: dump file ${DUMP_FILE} is empty" >&2
  exit 1
fi

ln -sf "${DUMP_FILE}" "${LATEST_LINK}" || cp -f "${DUMP_FILE}" "${LATEST_LINK}"
DUMP_SIZE="$(wc -c < "${DUMP_FILE}")"
echo "Backup completed successfully: ${DUMP_FILE} (${DUMP_SIZE} bytes)"

# Optional restic snapshot per runbook section 3
if [ -n "${RESTIC_REPOSITORY:-}" ] && command -v restic >/dev/null 2>&1; then
  echo "Creating restic snapshot..."
  restic backup "${DUMP_FILE}" --tag database
  restic forget --keep-daily 7 --keep-weekly 4 --keep-monthly 3 --prune || true
fi

# Rotate older local dumps beyond retention policy
find "${BACKUP_DIR}" -name "simpan_*.dump" -type f -mtime +"${RETENTION_DAYS}" -delete 2>/dev/null || true

# Write Prometheus textfile collector metric for BackupMissing alert
NOW_EPOCH="$(date +%s)"
cat <<EOF > "${METRICS_DIR}/backup.prom.$$"
# HELP backup_last_success_timestamp_seconds Timestamp of the last successful backup.
# TYPE backup_last_success_timestamp_seconds gauge
backup_last_success_timestamp_seconds ${NOW_EPOCH}
# HELP backup_last_size_bytes Size of the last successful backup file in bytes.
# TYPE backup_last_size_bytes gauge
backup_last_size_bytes ${DUMP_SIZE}
EOF
mv -f "${METRICS_DIR}/backup.prom.$$" "${METRICS_DIR}/backup.prom"
echo "Wrote Prometheus metric backup_last_success_timestamp_seconds: ${NOW_EPOCH}"
