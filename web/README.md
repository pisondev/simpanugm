# web

The Next.js frontend for all four roles: members reporting, officers working the desk,
administrators managing staff. Mobile first, Indonesian interface.

Owner: Pison Golda, with Axelle Chandra on the admin and analytics pages from Sprint 4.
Status: Sprint 1 scope implemented, see the route table below.

| Read first                                                                              | For                                     |
| --------------------------------------------------------------------------------------- | --------------------------------------- |
| [docs/ux/information-architecture.md](../docs/ux/information-architecture.md)           | every route and flow                    |
| [docs/ux/ui-guidelines.md](../docs/ux/ui-guidelines.md)                                 | tokens, components, copy, accessibility |
| [docs/product/user-stories.md](../docs/product/user-stories.md)                         | acceptance criteria                     |
| [docs/architecture/api.md](../docs/architecture/api.md)                                 | the API the generated client wraps      |
| [docs/engineering/repository-structure.md](../docs/engineering/repository-structure.md) | the layout of this folder               |

## Stack

Next.js 16.4 App Router with Turbopack, React 19, TypeScript strict with
`noUncheckedIndexedAccess`, Tailwind CSS 4, shadcn/ui on Base UI, Lucide, TanStack Query,
React Hook Form with Zod 4, next-intl, openapi-fetch over a client generated from
`api/openapi.yaml`. Tests: Vitest with Testing Library and MSW, Playwright with axe.

Node.js 22.22.2 or newer is required, jsdom 30 refuses older 22.x releases.

## Commands

| Command          | Does                                                                     |
| ---------------- | ------------------------------------------------------------------------ |
| `pnpm install`   | dependencies                                                             |
| `pnpm dev`       | dev server on port 3000, use <http://localhost:8000> through Caddy       |
| `pnpm gen`       | regenerate `src/lib/api/schema.d.ts` after `api/openapi.yaml` changes    |
| `pnpm lint`      | ESLint with `eslint-config-next`, `typescript-eslint` strict and a11y    |
| `pnpm typecheck` | route types and `tsc`, message keys are type checked too                 |
| `pnpm test`      | unit and component tests                                                 |
| `pnpm e2e`       | Playwright journeys against a running stack, `make e2e` from the root    |
| `pnpm build`     | standalone production build, the Dockerfile copies `.next/standalone`    |

## How it is put together

| Concern               | Where                                                                                                                       |
| --------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| Route guard           | `src/proxy.ts` sends visitors without a session cookie to `/masuk?next=`, layouts ask `GET /me` for the role               |
| Server data           | `src/lib/api/server.ts` forwards the cookie and client address to `API_INTERNAL_URL`, `load()` maps 401, 403 and 404       |
| Browser data          | `src/lib/api/client.ts` on `/api/v1`, TanStack Query, an expired session redirects to sign in without retrying             |
| Copy                  | every string in `src/messages/id.json`, typed through `src/i18n/messages.d.ts`                                              |
| Status vocabulary     | `src/lib/status.ts`, a test checks it against the enums in `api/openapi.yaml`                                               |
| Dates and times       | `src/lib/format.ts` and `src/lib/time.ts`, always `Asia/Jakarta`                                                            |
| Report forms          | `src/components/report/form/`, one schema per form, drafts and the idempotency key survive a reload                       |
| Photos                | `src/lib/image.ts` converts HEIC and resizes on the main thread, `photo-picker.tsx` uploads with progress                  |

Records outside a user's scope answer 404, so no page confirms that a record exists.

## Routes in Sprint 1

| Group     | Routes                                                                                                                        |
| --------- | ----------------------------------------------------------------------------------------------------------------------------- |
| Public    | `/`, `/masuk`, `/daftar`, `/verifikasi`, `/lupa-sandi`, `/atur-sandi`, `/undangan`, `/privasi`, `/ketentuan`, `/bantuan`       |
| Member    | `/beranda`, `/lapor/hilang`, `/lapor/temuan`, `/laporan`, `/laporan/hilang/[code]`, `/laporan/temuan/[code]` and their `/ubah`, `/barang-temuan`, `/profil` |
| Officer   | `/petugas`, `/petugas/terima`, `/petugas/inventaris`, `/petugas/cari`, `/petugas/lapor-hilang`, `/petugas/lapor-temuan`, `/petugas/laporan/hilang/[code]`, `/petugas/laporan/temuan/[code]` |
| Admin     | `/admin`, `/admin/pengguna`                                                                                                   |
| Operation | `/healthz` for the container health check                                                                                     |

Candidates, claims, notifications, profile editing and the remaining officer and admin
screens follow the sprints of [docs/project/project-plan.md](../docs/project/project-plan.md).
