# SimpanUGM live deployment on Hestia

The planned public URL is `https://simpanugm.pisondev.com`. The app uses one origin for
the website, API and media. Cloudflare proxies HTTPS to Hestia nginx; Hestia terminates
Let's Encrypt TLS and proxies to Caddy on `127.0.0.1:3042`. PostgreSQL, Go, Next.js and
ML have no host-published ports. Cloudflare zone SSL stays Full (strict).

This setup has been prepared but is not live. The app hostname still has no DNS record,
and the application containers have not been started. The Hestia mail domain, mailbox,
TLS certificate and DNS records are configured. SMTP STARTTLS and account authentication
passed, but no email was sent and external inbox delivery has not been verified. Local
commits from this work remain unpublished, and no matching GHCR image tags were produced.

## 1. Deployment files

The Hestia overlay is `deploy/compose.hestia.yaml`; its Caddy port binds only to loopback.
The templates in `deploy/hestia/` preserve Hestia's ACME and per-domain includes, forward
the original HTTPS scheme, disable proxy buffering for server-sent events and allow the
app's 10 MiB media uploads. `deploy/scripts/configure_cloudflare_simpanugm.py` has three
scoped modes: inspect, create the DNS-only A record, and enable proxy after an HTTPS
health check. It refuses conflicting DNS and never changes zone-wide SSL.

Release image names match `.github/workflows/release.yml`: `ghcr.io/pisondev/simpanugm/api`,
`/web` and `/ml`. The production Compose override uses those paths. Set `VERSION` to a
published tag; do not deploy `latest` for a first live release. The Hestia overlay must
be supplied last so it replaces the production Caddy 80 and 443 bindings.

## 2. Requirements before provisioning

The current VPS has about 3.9 GiB RAM plus 4 GiB swap, with about 2.2 GiB available while
other production sites were running. The `tierratie` Compose project used about 716 MiB,
including roughly 485 MiB for its monitoring containers. Stopping monitoring alone would
not leave enough safe headroom for the local ML service, which used 2.45 to 2.62 GiB by
itself. Do not stop other sites to make room. The production Compose override now limits
PostgreSQL to 512 MiB, API and worker to 384 MiB each, web to 384 MiB and Caddy to 128 MiB.
The local ML service is opt-in with the `local-ml` profile and stays off by default. The
app can run without AI image assistance and visual matching, or its API can point to an
external compatible ML service. Swap does not make the full local stack safe.

There is no drop-in free host for the current CLIP service. Cloudflare Workers AI has a
free daily neuron allocation, but the current image embedding model and service need a
compatibility port and ranking validation. Cloud Run can scale to zero, but has cold
starts, billing setup and a free quota that is too small for an always-running 2.5 GiB
model. Treat either as a separate engineering task, not a ready deploy target.

The deploy account (set as `HESTIA_USER`) logs in with an SSH key and can run Docker. Root SSH
is denied, so Hestia administration uses authenticated sudo. Keep every password outside the
repository and chat messages.

The VPS already runs Exim and Dovecot. A dedicated Hestia mail domain and
`no-reply@simpanugm.pisondev.com` account now use the hostname
`mail.simpanugm.pisondev.com`. Its DNS-only A record, MX, SPF, DKIM and DMARC records are
published for the SimpanUGM subdomain. The API's ignored `deploy/env/live-api.env` is configured
for authenticated STARTTLS on port 587. A TLS and SMTP-authentication probe passed without
sending a message. External inbox delivery remains unverified. Exim's primary hostname
does not match the VPS reverse DNS name, which may affect delivery reputation. The project
can use this VPS mail service without adding an external SMTP provider.

SimpanUGM currently supports filesystem media storage only (`STORAGE_DRIVER=fs`). The VPS
has about 32 GiB free, and the demo image pack is about 16 MiB, so a new R2 bucket is not
needed for this demo. User photos stay in the persistent local media volume. The existing
Cloudflare R2 buckets belong to other projects, so do not reuse them. For longer-term
off-site media storage, implement the S3-compatible driver and use a separate private
bucket with bucket-scoped credentials. Do not expose uploaded photos through a public
custom domain or `r2.dev` URL.

## 3. Runtime values

Create `deploy/env/live-api.env`, `live-web.env`, `live-ml.env` and `live-compose.env` from
the examples. Keep `api.env`, `web.env`, `ml.env` and `compose.env` for local development.
Use long random secrets and preserve them across releases. Set:

| File | Required live values |
|---|---|
| `live-api.env` | `APP_ENV=production`, `PUBLIC_BASE_URL=https://simpanugm.pisondev.com`, `SESSION_COOKIE_SECURE=true`, `SEED_OVERLAY_DIR=` and SMTP values |
| `live-web.env` | `NEXT_PUBLIC_BASE_URL=https://simpanugm.pisondev.com`, `API_INTERNAL_URL=http://api:8090` |
| `live-ml.env` | Generate a separate `ML_INTERNAL_TOKEN`; keep `HF_HOME=/models` |
| `live-compose.env` | published `VERSION`, distinct PostgreSQL role passwords, `ML_INTERNAL_TOKEN` and an absolute protected backup directory |

Leave both login attempt settings at their production defaults of 20 per IP and 5 per
email/IP per 15 minutes. Do not enable demo mode, fake clock, Mailpit, or the development
rate limits. Set file permissions to 600 and do not copy real credentials into examples.
The production Compose override uses the `live-*.env` files, leaves ML disabled by
default, and applies the memory limits listed above. Do not enable `local-ml` on this VPS.

## 4. Provision the domain and origin TLS

First inspect the shared Cloudflare zone and DNS record. Run the Python command from a
trusted local shell after providing a zone-scoped token with Zone Read, Zone Settings
Read and DNS Edit for `pisondev.com`:

```sh
python deploy/scripts/configure_cloudflare_simpanugm.py inspect
python deploy/scripts/configure_cloudflare_simpanugm.py dns
```

The DNS command creates only `simpanugm.pisondev.com` as a DNS-only A record to
the address in `ORIGIN_IP`. It refuses to replace a record that points elsewhere. Wait for DNS to
resolve to the VPS. Do not turn on proxy yet.

On the VPS, place the repository's `deploy/hestia/` files in one directory readable by
root. Start the app with the production and Hestia Compose files, then install the
Hestia templates and create the virtual host:

```sh
docker compose -f deploy/compose.yaml -f deploy/compose.prod.yaml -f deploy/compose.hestia.yaml --env-file deploy/env/live-compose.env up -d --wait
sudo bash deploy/hestia/simpanugm.sh domain
sudo bash deploy/hestia/simpanugm.sh tls
```

The production profile does not start the local ML container. Without an external ML
endpoint, text-based flows remain available while image assistance and visual matching
are unavailable. The `local-ml` profile is intended only for a larger host.

The root script checks the local Caddy health endpoint before changing Hestia. After
Let's Encrypt succeeds, verify `https://simpanugm.pisondev.com/healthz` and the browser
certificate. Then enable Cloudflare proxy:

```sh
python deploy/scripts/configure_cloudflare_simpanugm.py proxy
```

The proxy command verifies the origin's public HTTPS certificate and `/healthz` first.
It only changes this DNS record. Cloudflare must remain Full (strict).

## 5. Release and recovery

Create a reviewed version tag only after the release checks pass and images are available
in GHCR. On the VPS, take and verify a database backup before migration. Apply migrations
with the `migrate` service, then update all services and wait for health:

```sh
docker compose -f deploy/compose.yaml -f deploy/compose.prod.yaml -f deploy/compose.hestia.yaml --env-file deploy/env/live-compose.env pull
docker compose -f deploy/compose.yaml -f deploy/compose.prod.yaml -f deploy/compose.hestia.yaml --env-file deploy/env/live-compose.env run --rm migrate
docker compose -f deploy/compose.yaml -f deploy/compose.prod.yaml -f deploy/compose.hestia.yaml --env-file deploy/env/live-compose.env up -d --wait
```

Smoke check the public home page, `/healthz`, API readiness through the same origin,
registration email delivery and one real handover in a private test account. Keep the
database and media volumes during rollback. Application rollback does not undo database
migrations; restore a verified backup only after checking migration compatibility.

Do not report the service as live until DNS, origin TLS, Cloudflare Full (strict), login,
SMTP delivery, health checks, backups and the live browser flow have all been verified.
