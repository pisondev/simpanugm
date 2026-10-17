#!/bin/sh
set -eu
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres \
  -v owner_password="$SIMPAN_OWNER_PASSWORD" \
  -v app_password="$SIMPAN_APP_PASSWORD" \
  -v readonly_password="$SIMPAN_READONLY_PASSWORD" \
  -f /docker-entrypoint-initdb.d/init.sql.tpl
