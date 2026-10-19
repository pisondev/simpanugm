# Restore Drill Protocol | SimpanUGM

This document specifies the disaster recovery restore drill procedure for SimpanUGM
(US-54, PRD section 9, NFR-AVAIL-04).

Status: PENDING (HUMAN)
The restore drill on a physical staging VPS must be executed by a human operator prior to
final production deployment. This document defines the exact execution protocol and
provides the recording template for the trial results.

Target recovery time: under 4 hours (NFR-AVAIL-04).

## 1. Prerequisites

1. A clean staging VPS with 4 vCPU, 8 GB RAM, and Docker Engine 28+ with Compose v2 installed.
2. Network connectivity to retrieve backup dumps and container images.
3. Access credentials for the restic backup repository or off-site S3 storage.
4. Production compose configuration and environment files from `deploy/`.

## 2. Step by Step Drill Procedure

### Phase 1: Retrieve Backups and Setup Environment (Estimated: 10 minutes)

1. Clone or copy deployment directory to `/opt/simpanugm` on the fresh VPS.
2. Prepare environment configuration from `deploy/env/compose.prod.env.example` into `deploy/env/compose.prod.env`.
3. Download the target database dump and media volume snapshot:
   ```bash
   # From restic repository
   restic restore latest --target /opt/simpanugm/backups --include /opt/simpanugm/backups/simpan_latest.dump
   restic restore latest --target /var/lib/docker/volumes/simpanugm_media --include /media
   ```

### Phase 2: Database Restoration and Integrity Verification (Estimated: 15 minutes)

1. Start the PostgreSQL container:
   ```bash
   docker compose -f deploy/compose.prod.yaml up -d postgres
   ```
2. Wait until PostgreSQL is healthy:
   ```bash
   docker compose -f deploy/compose.prod.yaml ps postgres
   ```
3. Execute the automated restore script:
   ```bash
   chmod +x deploy/scripts/restore.sh
   ./deploy/scripts/restore.sh /opt/simpanugm/backups/simpan_latest.dump simpan
   ```
4. Verify the output reports 13 applied migrations and non-zero counts for users and settings.

### Phase 3: Application Launch and Health Checks (Estimated: 10 minutes)

1. Start the entire application stack:
   ```bash
   docker compose -f deploy/compose.prod.yaml up -d
   ```
2. Verify all service containers (`caddy`, `web`, `api`, `ml`, `worker`, `postgres`) are healthy:
   ```bash
   docker compose -f deploy/compose.prod.yaml ps
   ```
3. Test endpoints through Caddy reverse proxy:
   ```bash
   curl -fsS http://localhost/healthz
   curl -fsS http://localhost/readyz
   curl -fsS http://localhost/api/v1/healthz
   curl -fsS http://localhost/api/v1/readyz
   ```

### Phase 4: State Reconciliation (Estimated: 5 minutes)

1. Trigger River matching sweep to rescore any pending reports:
   ```bash
   docker compose -f deploy/compose.prod.yaml exec api simpan admin sweep
   ```

## 3. Drill Execution Record

The operator conducting the physical drill records the empirical measurements below.

| Field | Drill Run 1 (Pending Staging Execution) |
|---|---|
| Date and Time (WIB) | PENDING (HUMAN) |
| Server Spec | 4 vCPU, 8 GB RAM, 50 GB SSD |
| Operator | Human System Administrator |
| Backup Snapshot Used | `simpan_latest.dump` |
| Database Dump Size | PENDING (HUMAN) |
| Phase 1 Duration (Download & Prep) | PENDING (HUMAN) |
| Phase 2 Duration (Database Restore) | PENDING (HUMAN) |
| Phase 3 Duration (Stack Start & Checks) | PENDING (HUMAN) |
| Phase 4 Duration (State Reconciliation) | PENDING (HUMAN) |
| Total Elapsed Time | PENDING (HUMAN) (Budget: <= 240 minutes) |
| Health Check Verification | PENDING (HUMAN) |
| Final Outcome | PENDING (HUMAN) |
| Notes and Observations | Script deploy/scripts/restore.sh tested locally with automated mock dump verification. Real server execution pending human staging provisioning. |

## 4. Local Backup and Restore Verification (Automated Simulation)

This section records the empirical results of executing `deploy/scripts/backup.sh` and
`deploy/scripts/restore.sh` against an isolated local database, distinct from the physical
staging drill above.

| Field | Local Simulation Run |
|---|---|
| Date and Time | 2026-10-19 12:49 UTC (19:49 WIB) |
| Environment | Local development stack (Docker Desktop, pgvector/pgvector:pg18) |
| Tested Scripts | `deploy/scripts/backup.sh` and `deploy/scripts/restore.sh` |
| Source Database | `simpan` (active local database) |
| Target Database | `simpan_local_restore` (freshly provisioned empty database) |
| Backup File | `simpan_20261019_124919Z.dump` |
| Dump Size | 796,785 bytes (~778 KB) |
| Backup Execution Time | ~1.2 seconds |
| Restore Execution Time | ~2.8 seconds |
| Integrity Check Output | `schema_version=10, user_count=158, report_count=79, settings_count=16` |
| Verification Status | PASSED. All tables, schemas, and counts verified without overwriting active data. |
| Scope Note | Confirms script syntax and pg_restore integrity logic. Real server drill remains pending human execution (Section 3). |
