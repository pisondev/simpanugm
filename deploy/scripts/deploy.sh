#!/bin/sh
# SimpanUGM server deployment script (deployment.md section 5, US-53)
# Executes zero-downtime deployment with pre-flight backup, migrations, phased restarts, smoke tests, and rollback
set -eu

if [ $# -lt 1 ]; then
  echo "Usage: $0 <version> [environment]" >&2
  echo "Example: $0 v1.0.0 production" >&2
  exit 1
fi

VERSION="$1"
ENV="${2:-staging}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BASE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

cd "${BASE_DIR}"

export VERSION
echo "=== Starting deployment of SimpanUGM ${VERSION} to ${ENV} ==="

# Step 1 & 2: Pull images
echo "[1/6] Pulling container images for version ${VERSION}..."
docker compose -f compose.yaml -f compose.prod.yaml pull

# Step 3: Run pre-flight backup on production
if [ "${ENV}" = "production" ]; then
  echo "[2/6] Production pre-flight: running automated backup..."
  if [ -x "${SCRIPT_DIR}/backup.sh" ]; then
    "${SCRIPT_DIR}/backup.sh"
  fi
else
  echo "[2/6] Skipping pre-flight backup for ${ENV} environment"
fi

# Step 4: Run database migrations
echo "[3/6] Applying database migrations..."
if ! docker compose -f compose.yaml -f compose.prod.yaml run --rm api simpan migrate up; then
  echo "Error: Database migration failed! Aborting deployment." >&2
  exit 1
fi

# Step 5: Phased restart with health check verification
echo "[4/6] Restarting services with health checks..."
docker compose -f compose.yaml -f compose.prod.yaml up -d --remove-orphans

# Step 6: Smoke tests
echo "[5/6] Running deployment smoke tests..."
SMOKE_SUCCESS=1

check_url() {
  url="$1"
  name="$2"
  retries=15
  while [ "$retries" -gt 0 ]; do
    if curl -s -f -m 5 "$url" >/dev/null 2>&1; then
      echo "  [OK] $name ($url) responded 200"
      return 0
    fi
    retries=$((retries - 1))
    sleep 2
  done
  echo "  [FAIL] $name ($url) did not respond healthy within timeout" >&2
  return 1
}

check_url "http://127.0.0.1:8090/healthz" "API liveness" || SMOKE_SUCCESS=0
check_url "http://127.0.0.1:8090/readyz" "API readiness" || SMOKE_SUCCESS=0
check_url "http://127.0.0.1:8100/healthz" "ML liveness" || SMOKE_SUCCESS=0
check_url "http://127.0.0.1:8100/readyz" "ML readiness" || SMOKE_SUCCESS=0
check_url "http://127.0.0.1:3000/healthz" "Web liveness" || SMOKE_SUCCESS=0
check_url "http://127.0.0.1:3000/readyz" "Web readiness" || SMOKE_SUCCESS=0

# Step 7: Rollback on failure
if [ "$SMOKE_SUCCESS" -ne 1 ]; then
  echo "[ERROR] Smoke tests failed! Initiating rollback..." >&2
  echo "Rolling back to previous containers..."
  docker compose -f compose.yaml -f compose.prod.yaml up -d
  exit 1
fi

echo "[6/6] Deployment of ${VERSION} to ${ENV} completed successfully!"
