# Contributing to SimpanUGM

Three people, three languages, one repository, nine weeks. The rules below keep two people
off the same lines, keep `main` deployable, and make every merged change traceable to a
requirement.

## 1. Ownership

| Path | Owner | Reviewer by default |
|---|---|---|
| `api/` | Pison | Axelle for migrations, Irfan for anything under `internal/matching` |
| `web/` | Pison | Axelle or Irfan, whoever is free |
| `ml/` | Irfan | Pison for the HTTP contract, Axelle for evaluation data handling |
| `data/` | Axelle | Irfan for evaluation data, Pison for seed formats |
| `deploy/`, `.github/`, root files | Pison | any member |
| `docs/` | the owner of the area the page describes | any member |

`.github/CODEOWNERS` requests the owner's review automatically.

Editing outside your area is allowed through a pull request that the owner approves. The
owner is the one who answers for the area at review time, not the only one who may type
in it.

## 2. Contracts between areas

Four seams connect the parts. A change to any of them goes in a pull request labelled
`contract` and needs approval from every consumer.

| Contract | Producer | Consumers | Lives in |
|---|---|---|---|
| Public REST API | `api/` | `web/` | `api/openapi.yaml`, catalogued in [docs/architecture/api.md](docs/architecture/api.md) |
| Inference API | `ml/` | `api/` | `ml/openapi.json`, described in [docs/ai/matching-engine.md](docs/ai/matching-engine.md) |
| Database schema | `api/migrations/` | `api/`, `data/` analytics queries | [docs/architecture/data-model.md](docs/architecture/data-model.md) |
| Master data seeds | `data/seed/` | `api/` seed command | [data/README.md](data/README.md) |

Adding an optional field is safe. Renaming, removing, or changing the meaning of a field
is a breaking change and must say so in the pull request title.

## 3. Branches

`main` is always deployable and protected. Nobody pushes to it directly.

```
feat/<area>-<topic>     new behaviour           feat/api-lost-reports
fix/<area>-<topic>      bug fix                 fix/web-upload-heic
docs/<topic>            documentation only      docs/matching-weights
chore/<topic>           tooling, deps, ci       chore/ci-cache-pnpm
exp/<topic>             ml experiments          exp/fusion-logreg
```

`<area>` is one of `api`, `web`, `ml`, `data`, `deploy`. Branch from the latest `main`,
keep a branch alive for at most four days, then split the work. Rebase on `main` before
asking for review.

## 4. Commits

English, imperative mood, Conventional Commits, one logical change per commit.

```
feat(api): add lost report creation with idempotency key
fix(web): keep the draft when an upload fails
test(ml): pin the clip embedding of the fixture pair
docs(ai): record the calibrated tier thresholds
chore(deploy): pin postgres to 18.3
```

Scopes are the area names above. A commit compiles and its tests pass on its own.
History is read by people outside the team, so it should read as a clean story.

## 5. Pull requests

1. Open the pull request early as a draft if you want feedback.
2. Fill in the template: what changed, which requirement ids (`FR-...`, `US-...`) it
   serves, how it was tested, screenshots for UI.
3. CI must be green. A red check is never merged "to fix later".
4. One approval from the reviewer in section 1. Two for anything labelled `contract` or
   `security`.
5. Squash merge with a Conventional Commit title. Delete the branch.

Keep pull requests under roughly 400 changed lines excluding generated code. Larger work
lands behind a feature flag in several pull requests.

## 6. Definition of done

A story is done when it meets [docs/project/definition-of-done.md](docs/project/definition-of-done.md).
In short: reachable from the real UI or API, unit and integration tests pass, docs and
`CHANGELOG.md` updated in the same pull request, deployed to staging without errors.

## 7. Before you push

```
make check          # runs the same lint, typecheck and test steps as CI, for every area
```

The targets are listed in [docs/engineering/local-development.md](docs/engineering/local-development.md)
section 4. `make e2e` runs the browser journeys against the stack started by `make dev`.

## 8. Issues and the board

Work is tracked as GitHub issues on the project board with the columns Backlog, Ready,
In progress, In review, Done. Every issue carries one area label, one priority label
(`P0` Must, `P1` Should, `P2` Could) and the story id in the title, for example
`US-08 File a lost report`.
