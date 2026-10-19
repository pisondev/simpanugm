# Runbook | SimpanUGM

Procedures for the person on call. Each entry: how it shows up, what to check, what to
do. Commands run in `/opt/simpanugm` on the server as the `deploy` user.

## 1. First five minutes of any incident

1. Open the Overview dashboard and note when the problem started.
2. `docker compose ps`, look for restarting or unhealthy containers.
3. `docker compose logs --since 15m <service> | grep '"level":"error"'`.
4. Post in the team chat: what is broken, since when, who is looking.
5. If users are affected for more than 15 minutes, set the banner through
   `simpan admin banner "Layanan sedang mengalami gangguan. Laporan tetap tersimpan."`.

## 2. Incidents

### 2.1 Site down or 502 from Caddy

| Check | Action |
|---|---|
| `docker compose ps` shows `web` or `api` unhealthy | `docker compose logs --tail 200 api`. A missing environment variable or a failed migration stops the process at start |
| All healthy but 502 | `docker compose restart caddy`, then check `docker compose logs caddy` for certificate errors |
| Host out of memory | `free -m`, `docker stats`. `ml` is the usual suspect, it has a 3 GB limit. Restart it. If it repeats, set `ML_OCR_ENABLED=false` temporarily |
| Disk full | `df -h`. Prune old images with `docker image prune -a --filter "until=168h"`, check `backups/` rotation |

### 2.2 Matching stalled

Symptoms: `MatchingStalled` alert, new reports stay at "Mencari kandidat".

1. `curl -s http://localhost:8100/readyz` from inside the `app` network
   (`docker compose exec api wget -qO- http://ml:8100/readyz`).
2. Not ready: `docker compose logs ml`. Weights corrupted or missing, delete the
   `ml-models` volume contents for the failing component and restart, which re-downloads
   the pinned revision.
3. Ready but queue growing: check `ml_inference_seconds`. If OCR dominates, disable OCR
   temporarily, then plan capacity.
4. `model_version_mismatch` in `api` readiness: `ml` serves a revision that is not the
   active one. Redeploy the `ml` image that matches, or activate the matching version.
5. After recovery nothing else is needed. Jobs retry and the nightly sweep covers anything
   discarded. To catch up immediately: `docker compose exec api simpan admin sweep`.

### 2.3 Emails not arriving

1. Mailpit in staging, provider dashboard in production.
2. `river_jobs_total{kind="email.send",state="discarded"}` rising means credentials or
   quota. Fix, then `simpan admin retry-jobs --kind email.send --since 24h`.
3. In app notifications are unaffected, so users still see everything in the app.

### 2.4 Wrong handover reported

An owner says someone else took their item, or an officer reports a mistake.

1. Do not delete or edit anything.
2. Admin opens the audit log filtered by the found report: the handover entry shows the
   officer, the claim, the ID type and last four characters, the time, the request id.
3. Export the audit rows and the claim, including the checklist results, as CSV and store
   them with the incident note.
4. Suspend the claimant account if fraud is suspected (`user.suspend`).
5. Hand the case to the security unit with the export. The system's part ends with an
   accurate record.

### 2.5 Suspected account compromise

1. Admin revokes all sessions of the account and suspends it.
2. Check the audit log of the account for the last 30 days, and for an officer also the
   `private.read` entries.
3. If private details were read without reason, treat it as a personal data breach,
   section 7.

### 2.6 Model or fusion version misbehaving

Symptoms: AI-12 shows a jump in dismiss rate, a PSI warning, or officers report bad
candidates after an activation.

1. Admin page AI, Versi Model: activate the previous version of the component.
2. The sweep rescores open reports, watch `match_pipeline_seconds` and queue depth.
3. Open an issue labelled `ml` with the AI-12 screenshots and the time of activation.

## 3. Backups

| What | How | When | Kept |
|---|---|---|---|
| Database | `pg_dump --format=custom` of `simpan` by `simpan_owner`, then `restic backup` | daily 02:30 WIB | 7 daily, 4 weekly, 3 monthly |
| Media volume | `restic backup /var/lib/docker/volumes/simpanugm_media` | daily 03:00 WIB | same policy |
| Configuration | `restic backup /opt/simpanugm --exclude backups` | daily | same policy |

The restic repository is an S3 compatible bucket in a different provider from the VPS.
`deploy/scripts/backup.sh` runs from a systemd timer, reports success to Prometheus through
a textfile metric, and `BackupMissing` alerts when that metric is older than 26 hours.

## 4. Restore

Scheduled to be practiced on a fresh staging VPS prior to production launch (PENDING HUMAN),
timed and recorded using the protocol template in `docs/operations/restore-drill.md`.
Target: under 4 hours (NFR-AVAIL-04).

1. Provision a server per deployment section 2 and install Docker.
2. Copy `/opt/simpanugm` from the restic snapshot: `restic restore latest --target / --include /opt/simpanugm`.
3. `docker compose up -d postgres` and wait for health.
4. `restic dump latest /backups/simpan.dump > simpan.dump`, then execute
   `./deploy/scripts/restore.sh simpan.dump simpan` to restore and verify schema integrity.
5. Restore the media volume snapshot into the new volume path.
6. Run `./deploy/scripts/deploy.sh` or `docker compose -f deploy/compose.prod.yaml up -d`, then run smoke tests.
7. Point DNS to the new server if the old one is gone.
8. Run `simpan admin sweep` so matching state is recomputed for anything the dump missed.

## 5. Routine tasks

| Task | When | How |
|---|---|---|
| Check alerts, failed jobs, disk | daily by the on call member | dashboards, River failed list in admin |
| Dependency updates | weekly | review and merge Dependabot pull requests that pass CI |
| AI-12 review | weekly, Monday | the AI Engineer checks drift and outcomes, notes in the team chat |
| Restore test of one random daily snapshot | monthly | restore the dump into a scratch database and count rows |
| Access review | monthly | admin lists officers and admins, removes anyone who left, reviews `private.read` volume per officer |

## 6. Secret rotation

1. Generate the new value with `openssl rand -base64 32`.
2. Database password: `ALTER ROLE simpan_app PASSWORD '...'` as `simpan_owner`, update
   `env/api.env`, `docker compose up -d api worker`.
3. `ML_INTERNAL_TOKEN`: update `env/api.env` and `env/ml.env`, restart `ml`, `api`,
   `worker` together.
4. `HANDOVER_PEPPER`: update, restart `api`, then tell officers that outstanding codes
   must be regenerated. Approved claimants see a new code in the app.
5. Record the rotation in the team log with the date, not the value.

## 7. Personal data breach

UU PDP requires written notification within 3 times 24 hours of discovery.

| Hour | Step |
|---|---|
| 0 | Contain: revoke sessions, suspend accounts, rotate secrets, take the affected function offline if needed |
| 0 to 12 | Establish scope from the audit log: which records, which people, which fields, since when |
| 12 to 48 | Prepare the notification: what happened, what data, what we did, what the person can do, contact |
| by 72 | Email every affected user, inform the university's security unit and the course lecturers in writing |
| after | Write a post incident review in `docs/operations/incidents/` and add tests or controls that would have prevented it |
