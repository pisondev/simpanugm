#!/usr/bin/env bash
# Install the Hestia reverse proxy template and provision one domain or its TLS.
set -euo pipefail

if [[ ${EUID} -ne 0 ]]; then
  echo "Run as root on the VPS." >&2
  exit 1
fi

export PATH="/usr/local/hestia/bin:${PATH}"
script_dir="$(cd "$(dirname "$0")" && pwd)"
user="${HESTIA_USER:?set HESTIA_USER to the Hestia account}"
domain="simpanugm.pisondev.com"
ip="${ORIGIN_IP:?set ORIGIN_IP to the public address of the server}"
template="simpanugm"
port="3042"

case "${1:-}" in
  domain)
    curl --fail --silent --show-error "http://127.0.0.1:${port}/healthz" >/dev/null
    install -m 644 "${script_dir}/hestia.tpl" "/usr/local/hestia/data/templates/web/nginx/php-fpm/${template}.tpl"
    install -m 644 "${script_dir}/hestia.stpl" "/usr/local/hestia/data/templates/web/nginx/php-fpm/${template}.stpl"
    if ! v-list-web-domain "${user}" "${domain}" json >/dev/null 2>&1; then
      v-add-web-domain "${user}" "${domain}" "${ip}" no none
    fi
    v-change-web-domain-tpl "${user}" "${domain}" "${template}" no
    ;;
  tls)
    if [[ ! -s "/home/${user}/conf/web/${domain}/ssl/${domain}.pem" ]]; then
      v-add-letsencrypt-domain "${user}" "${domain}"
    fi
    v-add-web-domain-ssl-force "${user}" "${domain}" no
    v-add-web-domain-ssl-hsts "${user}" "${domain}" no
    ;;
  *)
    echo "Usage: $0 domain|tls" >&2
    exit 2
    ;;
esac

nginx -t
systemctl reload nginx
echo "${domain}: ${1} configuration ready"
