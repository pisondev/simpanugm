# Definition of Ready and Definition of Done | SimpanUGM

## Definition of ready

A story moves to Ready only when:

1. It has an issue with the story text, acceptance criteria, requirement ids, area,
   priority and estimate.
2. Its acceptance criteria are testable as written.
3. Its contracts are known: the endpoints in `openapi.yaml` or a pull request adding them,
   the tables in data-model, the screens in information-architecture.
4. Its dependencies are merged or scheduled earlier in the same sprint.
5. It fits in one sprint. Anything above 8 points is split.

## Definition of done

A story is done only when every line holds.

### Behaviour

- [ ] Every acceptance criterion is met and reachable through the real entry point: the
      page for UI stories, the endpoint for API stories, the job for background stories.
- [ ] Error, empty and loading states of information-architecture section 4 exist for
      every new screen.
- [ ] UI copy is Indonesian, comes from the message catalog and follows ui-guidelines.

### Tests

- [ ] Unit tests cover the logic, including each business rule the story touches.
- [ ] Integration tests cover each boundary the story touches: HTTP, database, storage,
      jobs, inference API.
- [ ] Permission and projection tests are extended for every new endpoint or field.
- [ ] `make check` passes locally and CI is green.
- [ ] The affected end to end journey still passes.

### Quality

- [ ] Lint and type checks are clean, coverage did not drop by more than 2 points without
      a stated reason.
- [ ] No secret, personal data, dataset or weight is committed.
- [ ] Logs added by the story follow observability section 1 and contain no redacted key.
- [ ] Accessibility: axe has no serious or critical finding on new pages, keyboard
      operation checked by hand.

### Documentation

- [ ] Docs that describe the changed behaviour are updated in the same pull request.
- [ ] `CHANGELOG.md` has a line under `Unreleased`.
- [ ] A new decision that is expensive to reverse has an ADR.

### Delivery

- [ ] Reviewed and approved per CONTRIBUTING section 5, squash merged.
- [ ] Deployed to staging by the pipeline and checked there by someone other than the
      author.

## Done for AI work

An experiment, a model version or a fusion version is done when, in addition:

- [ ] The run is reproducible from the repository per evaluation section 10.
- [ ] The run id, config and gate table are in the pull request.
- [ ] Results that fail a gate are recorded in the evaluation report anyway.
- [ ] A model card section exists for anything activated.

## 3. Sprint 4 status and verification audit

Status as of Sprint 4 completion (19 October 2026). Only proven criteria are checked; human and live staging VPS actions remain unchecked per project policy.

### Behaviour
- [x] Every acceptance criterion is met and reachable through the real entry point for US-41 (semantic search), US-42 (moderation), US-43 (runtime settings), US-44 (audit log viewer), US-45 (account deletion), US-46 (campus hotspot map), US-47 (operational trends), US-48 (case clustering), US-49 (AI monitoring), US-52 (observability), US-53 (deployment artifacts), US-54 (backup/restore scripts), and US-55 (security hardening).
- [x] Error, empty and loading states exist for every new admin and officer screen.
- [x] UI copy is Indonesian, comes from the message catalog and follows ui-guidelines.

### Tests
- [x] Unit tests cover all business logic (`go test -tags nodynamic ./...`, `pnpm test`, `uv run pytest`).
- [x] Integration tests cover boundaries across API modules (`go test -tags integration,nodynamic ./internal/server`).
- [x] Permission and projection tests extended for all admin, moderation, and settings endpoints.
- [x] `make check` passes locally (gofmt, golangci-lint, typecheck, eslint, ruff, mypy).
- [x] Second load test completed with `make load` (p95: public listing 28.5 ms <= 300 ms budget, login 126.4 ms <= 500 ms budget, claims listing 22.9 ms <= 300 ms budget).
- [ ] Staging end to end journey execution on CI (pending push and workflow dispatch by human).

### Quality
- [x] Lint and type checks are clean across Go, TypeScript, and Python.
- [x] No secrets, personal data, or binary model weights committed.
- [x] Logs follow observability specifications without leaking redacted keys.
- [x] Production runtime dependency audits clean: `pnpm audit --prod` (0 vulnerabilities), `pip-audit` (0 vulnerabilities), `govulncheck` clean on application packages.
- [ ] ZAP baseline scan of staging with no high finding (pending live staging VPS deployment).

### Documentation
- [x] Docs updated in the same commits: `evaluation.md`, `model-cards.md`, `traceability.md`, `release-checklist-v1.0.0.md`.
- [x] `CHANGELOG.md` updated under `Unreleased` for all Sprint 4 features and fixes.

### AI Work
- [x] Learned fusion experiment implemented with 5-fold grouped cross-validation on synthetic data fixture (SINTETIS).
- [x] Multi-modal fusion model card added to `model-cards.md`.
- [x] Calibrated linear fusion retained as active model until frozen real dataset evaluation.
- [ ] Official gate run and activation on frozen real campus dataset (TERTUNDA until 15 November dataset freeze).

### Delivery (Requires Human Operator)
- [ ] Merged and approved PR #4.
- [ ] Deployment to staging VPS and production VPS.
- [ ] Live backup restore drill on fresh VPS.
- [ ] Usability test round 2 with campus participants.
- [ ] v1.0.0 release tagging.

