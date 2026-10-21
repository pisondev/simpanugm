# Follow-up validation

Validation for the demo photo and story corrections, configurable local login budgets,
and complete browser suites. Run on 2026-10-21 with the real local ML service.

| Check | Result |
|---|---|
| `make demo` | Pass. 8 users, 19 lost reports, 110 found reports, 5 claims and 1 duplicate. Match and cluster counts depend on real model scoring. |
| `make demo-check` | 20 passed, 0 failed, 0 warnings. Includes approved handover, owner information reply and finder-held item intake. |
| Normal browser suite | `make e2e`, 24 passed, 0 failed, 0 skipped. This target enables the fake clock required by J6. |
| Demo browser suite | `DEMO_E2E=1 pnpm exec playwright test --reporter=list`, 35 passed, 0 failed, 0 skipped. |
| `golangci-lint run --build-tags nodynamic,integration ./...` in `api` | `0 issues.` |
| Go unit tests | `go test -tags nodynamic ./...` passed across all packages. |
| Go server integration tests | `go test -tags integration,nodynamic ./internal/server` passed in 59.419 seconds. |
| Web format, types and lint | Prettier, typecheck and ESLint passed. |
| Web unit tests | 16 files, 75 passed. |
| Web production build | Passed with Next.js 16.4.2. |
| Demo data tests | 6 passed. |
| Data tests | 171 passed. The `ERROR: source drift` line is an expected negative fixture; the full suite result is `OK`. |
| Repository rules | `python automation/check_rules.py ai-model/simpanugm` scanned 657 text files and reported `clean`. |
| Production Compose | `docker compose ... config --quiet` passed for production plus Hestia overlays. |
| Caddy | `caddy validate` reported `Valid configuration`. |
| Axe | No violations in the passing combined browser runs. |

The first final normal-suite attempt used the development Compose stack. J6 received a
refused fake-clock request at `/api/test/clock/advance`; `deploy/compose.e2e.yaml` showed
the stack lacked `APP_ENV=e2e` and `FEATURE_FAKE_CLOCK=true`. The complete `make e2e`
run with the correct override then passed 24/24.

The demo suite first exposed three real test issues. The owner reference screen opened
the newest report, which is the already approved umbrella and has no open candidates.
It now chooses a report with current candidates from the owner's API results. The claim
detail page had a 350 px intrinsic photo width inside a 328 px phone content column; its
grid and photo strip now allow shrinking. Both issues passed a full demo suite rerun.

Representative Playwright screenshots are generated under the ignored
`web/test-results/screenshots/` directory. The set includes public landing and owner
candidate views at desktop and phone sizes, photo-assisted found-report forms, and the
officer claim queue and detail at both widths. They are local test artifacts and are not
committed.
