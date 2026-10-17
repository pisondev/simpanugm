# Coding Standards | SimpanUGM

Rules every reviewer checks. Formatting and most of the rest are enforced by tools, so
review time goes to design and behaviour.

## 1. Everywhere

| Rule | Detail |
|---|---|
| Language | Code, identifiers, comments, logs, commit messages and docs in English. UI strings in Indonesian, in the message catalog only |
| Domain words | Identifiers follow the glossary: `LostReport`, `FoundReport`, `Match`, `Claim`, `Custody`, `Post`. Never mix `item`, `barang`, `object` for the same thing |
| Comments | Only where the code cannot say it: a constraint, a non obvious decision, a trap. One line, lowercase first letter, no full stop, for example `// pgvector applies the filter inside the index only with iterative scan on` |
| No banners | No separator comments, no boxed headings, no emoji |
| Secrets | Never in code, tests, fixtures or logs |
| Time | Inject a clock. Never call `time.Now()`, `Date.now()` or `datetime.now()` inside domain code |
| Errors | Users see Indonesian messages from the catalog. Logs carry the technical cause |
| Typography | Plain hyphens and commas only in text. No typographic dashes |

## 2. Go

| Rule | Detail |
|---|---|
| Format and lint | `gofmt`, `goimports`, `golangci-lint` with `govet`, `staticcheck`, `errcheck`, `gosec`, `revive`, `gocritic`, `bodyclose`, `sqlclosecheck`, `depguard` |
| Module boundaries | `depguard` forbids importing another module's `store` or `queries`. Cross module calls go through the other module's `Service` |
| Context | Every function doing I/O takes `ctx context.Context` first |
| Errors | Wrap with `fmt.Errorf("claims: approve: %w", err)`. Domain errors are typed values (`ErrInvalidTransition`, `ErrStaleVersion`) mapped to problem codes in one place |
| Panics | Only for programmer errors at start up. The HTTP recover middleware turns a panic into a 500 with a request id |
| Transactions | `db.InTx(ctx, func(tx pgx.Tx) error)` at the service layer. Handlers never open transactions. Stores accept a `db.Querier` so they run inside or outside one |
| SQL | Only in `queries.sql` files through sqlc. No string built SQL. Every list query takes the caller scope as a parameter |
| Logging | `slog` with the request scoped logger from context. Keys in `snake_case`. No personal data beyond ids |
| Configuration | One `config.Config` struct built at start, passed explicitly. No global variables except the logger default |
| Concurrency | Goroutines only inside the platform layer (SSE hub, River). Domain code is sequential |
| Naming | Packages are short nouns. No `util`, `common`, `helpers` |
| Tests | Table driven, `t.Parallel()` where safe, `go-cmp` for structs. Integration tests carry `//go:build integration` |

## 3. TypeScript and Next.js

| Rule | Detail |
|---|---|
| Strictness | `strict`, `noUncheckedIndexedAccess`, no `any`, no non null assertions except in tests |
| Format and lint | Prettier, ESLint with `eslint-config-next`, `@typescript-eslint/strict`, `jsx-a11y` |
| Components | Server components by default. `"use client"` only for interactivity, kept as small leaves |
| Data | Server components call `api` through `lib/api` with the forwarded cookie. Client components use TanStack Query hooks wrapping the same client. Never `fetch` an API URL by hand |
| Types | API types come only from the generated `schema.d.ts`. Hand written duplicates of API shapes are rejected in review |
| Forms | React Hook Form with a Zod schema per step. Server errors map back to fields by `errors[].field` |
| Strings | Every user facing string in `messages/id.json`, read through `next-intl` |
| Styling | Tailwind utilities and tokens. No inline style objects except for computed values such as bar widths |
| Security | No `dangerouslySetInnerHTML`. No secrets in client bundles: only `NEXT_PUBLIC_` variables reach the browser and none of them is secret |
| Files | `kebab-case.tsx` for files, `PascalCase` for components, one component per file for anything exported |
| Tests | Behaviour through the DOM with Testing Library queries by role and label, API mocked with MSW |

## 4. Python

| Rule | Detail |
|---|---|
| Format and lint | `ruff format`, `ruff check` with rule sets `E`, `F`, `I`, `B`, `UP`, `N`, `S`, `SIM`, `RUF` |
| Types | Annotations everywhere in `ml/app`, `mypy --strict` on it. `ml/eval` annotated, checked without strict |
| Models | Loaded once at start in `registry.py`. Never inside a request handler |
| Request handlers | Thin: validate, call one analysis function, return. CPU heavy work runs in the thread pool with `run_in_threadpool` |
| Determinism | No randomness in serving. Evaluation seeds per evaluation section 10 |
| Notebooks | Exploration only, outputs stripped by `nbstripout`, importing from `ml/app` or `ml/eval` rather than redefining logic |
| Tests | `pytest`, fixtures in `conftest.py`, model heavy tests marked `slow` and run in CI with cached weights |

## 5. SQL and migrations

| Rule | Detail |
|---|---|
| Conventions | data-model section 1 |
| One change per migration | A migration does one thing and has a down step that undoes it |
| Safe on live data | Add columns nullable or with a default. Build indexes `CONCURRENTLY` in their own migration. Rename by add, backfill, switch, drop over two releases |
| Review | Every migration is reviewed by Axelle for data correctness and by Pison for application impact |
| Queries | Use explicit column lists, never `SELECT *`. Every query that can grow has a `LIMIT` and an index that serves its `WHERE` and `ORDER BY` |

## 6. API design

| Rule | Detail |
|---|---|
| Contract first | Change `openapi.yaml`, regenerate, then implement (ADR-0009) |
| Resources and actions | Nouns for resources. State machine events are sub resources with `POST`, for example `POST /claims/{id}/approve`, never a `PATCH` of `status` |
| Projections | A new field is added to each role's response type deliberately. Member types are reviewed against BR-VIS |
| Errors | A new failure mode gets a stable `code` in api section 1.1 |
