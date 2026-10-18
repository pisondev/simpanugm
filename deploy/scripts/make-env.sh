#!/bin/sh
# creates deploy/env/*.env from the templates with random development secrets, existing files are kept
set -eu
cd "$(dirname "$0")/../env"

rand() {
  head -c 48 /dev/urandom | base64 | tr -d '/+=\n' | cut -c1-32
}

value() {
  grep "^$1=" "$2" 2>/dev/null | head -n 1 | cut -d= -f2- || true
}

if [ -f compose.env ]; then
  POSTGRES_PASSWORD=$(value POSTGRES_PASSWORD compose.env)
  SIMPAN_OWNER_PASSWORD=$(value SIMPAN_OWNER_PASSWORD compose.env)
  SIMPAN_APP_PASSWORD=$(value SIMPAN_APP_PASSWORD compose.env)
  echo "kept compose.env"
else
  POSTGRES_PASSWORD=$(rand)
  SIMPAN_OWNER_PASSWORD=$(rand)
  SIMPAN_APP_PASSWORD=$(rand)
  sed -e "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=$POSTGRES_PASSWORD|" \
    -e "s|^SIMPAN_OWNER_PASSWORD=.*|SIMPAN_OWNER_PASSWORD=$SIMPAN_OWNER_PASSWORD|" \
    -e "s|^SIMPAN_APP_PASSWORD=.*|SIMPAN_APP_PASSWORD=$SIMPAN_APP_PASSWORD|" \
    -e "s|^SIMPAN_READONLY_PASSWORD=.*|SIMPAN_READONLY_PASSWORD=$(rand)|" \
    compose.env.example >compose.env
  echo "wrote compose.env"
fi

ML_INTERNAL_TOKEN=$(value ML_INTERNAL_TOKEN api.env)
[ -n "$ML_INTERNAL_TOKEN" ] || ML_INTERNAL_TOKEN=$(rand)

for name in api web ml; do
  if [ -f "$name.env" ]; then
    echo "kept $name.env"
    continue
  fi
  sed -e "s|\${SIMPAN_APP_PASSWORD}|$SIMPAN_APP_PASSWORD|g" \
    -e "s|\${SIMPAN_OWNER_PASSWORD}|$SIMPAN_OWNER_PASSWORD|g" \
    -e "s|\${ML_INTERNAL_TOKEN}|$ML_INTERNAL_TOKEN|g" \
    -e "s|\${HANDOVER_PEPPER}|$(rand)|g" \
    "$name.env.example" >"$name.env"
  echo "wrote $name.env"
done

chmod 600 ./*.env 2>/dev/null || true
