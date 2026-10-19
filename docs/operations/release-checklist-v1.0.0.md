# Security Release Checklist v1.0.0 | SimpanUGM

Status of pre-release security verification per security-privacy.md section 8 and US-55.

| Item | Status | Verification Detail |
|---|---|---|
| Permission matrix tests pass for every cell | PASS | Verified in automated integration test suites (`api/internal/server/*_integration_test.go`) covering member, officer, admin, and anonymous roles |
| Response snapshot tests prove members never receive restricted fields | PASS | Verified in `reports_integration_test.go` and `profile_integration_test.go`, verifying suppression of hidden/private details and finder identities |
| EXIF fixture test passes. Masking fixture test passes | PASS | Verified in `api/internal/platform/textx/textx_test.go` (regex masking for NIK, KTM, cards) and `ml/tests/test_ocr.py` |
| Rate limits verified with an automated test per limit | PASS | Verified in `api/internal/platform/ratelimit/ratelimit_test.go` and `auth_integration_test.go` |
| CSP and security headers verified with automated configuration | PASS | Verified in `deploy/caddy/Caddyfile` (HSTS, nosniff, frame options, permissions policy) and Next.js middleware nonce generator |
| ZAP baseline scan of staging with no high finding | PENDING (HUMAN) | Requires live staging VPS deployment; execution scheduled during staging test window |
| `govulncheck`, `pnpm audit --prod`, `pip-audit` clean of high and critical | PASS | Verified on 19 October 2026: `pnpm audit --prod` 0 vulnerabilities; `pip-audit` 0 vulnerabilities; `govulncheck` clean of vulnerabilities in application packages |
| Secrets present only in server `.env`, repository scanned | PASS | Verified with `automation/check_rules.py` and secret pattern scanner; no secrets committed to git |
| Database roles verified: `simpan_app` cannot update or delete `audit_logs` | PASS | Verified via trigger protection and table permissions in `00004_audit_and_triggers.sql` and integration tests |
| Backup restore drill done and timed | PENDING (HUMAN) | Backup and restore automation scripts `deploy/scripts/backup.sh` and `deploy/scripts/restore.sh` created and verified locally; live restore drill on fresh VPS requires human operator |
| Privacy notice and terms published and linked | PASS | Privacy policy and terms links present on registration dialog and application footer |
| Two members other than the author reviewed page against implementation | PENDING (HUMAN) | Awaiting peer review by team members prior to v1.0.0 release tagging |
