# Demo polish phase 2 playbook

Execute on top of the phase 1 reference implementation. Read
[demo-first-plan.md](demo-first-plan.md), [demo-polish.md](demo-polish.md),
[ui-guidelines.md](../ux/ui-guidelines.md), [design-research.md](../ux/design-research.md)
and [phase1-validation.md](../ux/phase1-validation.md) first. This is presentation and
interaction work on a real application. Hosting remains on hold.

## 1. Starting state and boundaries

- [ ] Confirm `git rev-parse --show-toplevel` is the SimpanUGM repository and inspect
  `git status --short`. Preserve any owner changes. Do not reset or clean unrelated work.
- [ ] Use local git identity `Pison Golda <pison.gm.dev@gmail.com>`. Verify installed hooks.
- [ ] Keep the three canonical flows working: public `/`, owner candidate list and
  detail, and `/lapor/temuan`. Read their implementations before modifying shared UI.
- [ ] Keep real model outputs, report permissions, private evidence, identity blurring,
  all existing actions, error recovery, draft persistence and demo labels.
- [ ] Change `web/`, `docs/` and `data/demo/`, plus the required root `CHANGELOG.md`.
  The data seeding integration in section 6 is a necessary, isolated exception in
  `api/internal/demo/`; document that exception in the final report.
- [ ] Inspect the current route files with `rg --files web/src/app`. The older IA has
  planned route names. The queue is currently inside `/petugas`; the board is
  `/petugas/papan`; handover is inside the officer claim detail. Fusion activation is
  in `/admin/ai/monitoring`. Do not invent `/petugas/antrean`, `/petugas/serah-terima`
  or `/admin/ai/model` pages to satisfy an outdated route list.

## 2. Design system recipe

Import motion from `@/components/motion/primitives`. Use existing UI, matching, report
and layout components. All new UI copy goes in `web/src/messages/id.json`, under the
existing feature namespace. Move hardcoded strings into that file when touching admin
screens. Do not rewrite unrelated screens in the same batch.

| Role | Token or class | Rule |
|---|---|---|
| Page, recessed, raised | `bg-background`, `bg-surface-inset`, `bg-surface-raised` | White content on a quiet background; do not tint every panel |
| Evidence | `surface`, `ai-panel` | Resting bordered card; subtle gradient only for model assistance or evidence |
| Elevation | `shadow-surface`, `shadow-raised`, `shadow-floating` | Choose one role per panel; never animate the shadow |
| Radius | 8 px controls, 12 px cards, `--radius-hero` at 24 px | Keep established shapes; hero radius is not for every button |
| Brand and status | `text-primary`, `text-muted-foreground`, `text-success-text`, `text-warning-text` | Use the status map; raw Tailwind green and amber text are not substitutes |
| Motion | `--motion-fast: 120ms`, `--motion-enter: 160ms`, `--motion-score: 200ms`, `--motion-ease` | No spring library, layout animation or delayed input |
| Image | Existing `PhotoStrip`, `mediaUrl`, `loading="lazy"` for secondary images | Protected same-origin media; retain blurred variants and reserved ratios |
| Typeface | Existing local Inter | No Google font imports, remote CSS or runtime CDN |

### Surfaces, background and empty state

```tsx
<section className="surface p-5">{content}</section>
<section className="ai-panel p-5">{modelEvidence}</section>
<div className="auth-texture rounded-2xl p-4 sm:p-8">{authCard}</div>
<EmptyState icon={Search} action={retryButton}>{emptyMessage}</EmptyState>
```

`EmptyState` is from `components/layout/empty-state`. Use one explanatory sentence and
one relevant action. Keep textures behind cards, never behind small form text. The
public landing's Lucide composition is decorative, not a fake report or an AI result.

### Entrance and stagger

```tsx
<Reveal variant="fade">{summary}</Reveal>
<Reveal>{newResult}</Reveal>
<Stagger className="grid gap-3">
  {items.map((item, index) => (
    <article key={item.id} style={{ "--motion-index": index } as CSSProperties}>
      {renderItem(item)}
    </article>
  ))}
</Stagger>
```

Import `CSSProperties` from React. For semantic lists, keep `ol` or `ul` and use
`className="motion-stagger"` instead of changing it to a div. Delay caps at 120 ms;
individual entrances finish in 160 ms. Content is present immediately. Do not remount
form inputs with a key to restart an animation.

### Press, lift and loading

```tsx
<Link href={href} className="surface motion-press motion-lift block p-4">
  {cardContent}
</Link>
<div role="status">{loadingMessage}<Skeleton className="mt-3 h-24 w-full" /></div>
```

`Button` and `ButtonLink` already have press feedback. Add no second scale animation.
Lift is only for interactive cards and hover-capable devices. `Skeleton` already has
finite transform shimmer. Match its dimensions to the loaded layout. Keep actionable
errors inline and existing neutral toasts for success.

### Scores and numbers

```tsx
<CountUp value={match.overall * 100} suffix="%" />
<ScoreBars explanation={match.explanation} scores={officerScores} />
```

`CountUp` receives the real percentage, not the raw 0 to 1 fraction. `ScoreBars` is from
`components/matching/score-bars`. For member data, omit `scores` and preserve the level
proxies and the caveat on the canonical candidate screen. Never pass invented numeric
component scores. The low-level `AnimatedBar` accepts a 0 to 100 value; when using it
outside `ScoreBars`, put final `aria-valuenow`, label and reason on its meter wrapper.
It only animates the decorative fill. Do not animate the wrapper width.

### Suggestions and slow models

Use `AssistNote`, `SuggestedTag`, `CategoryAlternatives` and `RedraftButton` from
`components/report/form/assist-ui`. Keep `useAssist` as the only suggestion state owner.

```tsx
<AssistNote assist={assist} />
<Field id="brand" label={brandLabel} suggested={assist.suggested("brand", brand)}
  motionIndex={2} hint={<SuggestedTag show={assist.suggested("brand", brand)} />}>
  {(aria) => <Input {...aria} {...register("brand")} />}
</Field>
```

Apply values immediately, only into empty fields. Stagger decorative field highlights
in category, colors, brand, description order using indices 0 to 3. Suggestions stay
editable. Do not introduce fake analysis stages, percentages, typing delays or branded
model personalities. Keep the eight-second request bound, inline unavailable message,
explicit retry and manual entry. A real incorrect classification must remain visible
and correctable; the reference earbuds image was classified as Charger during testing.

### Comparison and finder hint

```tsx
<ComparisonPanel leftTitle={t("boardTypeLost")} rightTitle={t("boardTypeFound")}
  left={lostEvidence} right={foundEvidence} />
<FinderHint items={permittedHintItems} />
```

Both components are in `components/matching`. `lostEvidence` and `foundEvidence` are the
existing permission-checked board content, not new API data. Keep existing pin, rematch,
invite, claim and dismissal actions. The comparison stacks below `lg`. The finder hint
must contain only the permitted item and category, never owner identity or hidden details.

### Page transitions and reduced motion

```tsx
<PageTransition className="flex flex-col gap-6">{pageContent}</PageTransition>
const reduced = useReducedMotion()
```

Use one `PageTransition` per page, never nest named wrappers. Native cross-document
transitions are progressive enhancement for same-origin full navigation. Next.js soft
navigation stays immediate. Do not intercept links or enable experimental React/Next
flags to force effects. CSS utilities respect reduced motion; custom JS effects must
also use the subscribed preference and cancel active frames. All existing counts and
meters have final accessible values even while visual animation runs.

## 3. Universal gate G, required for every batch

For each screen in a batch:

- [ ] Inspect loaded, loading, empty, error and permission behavior with the real API.
  Preserve existing retry and draft recovery. Do not remove a state because the seed
  does not currently show it. Use network failure interception only to test failure.
- [ ] Check 1366 by 900 and 360 by 800, keyboard operation, visible focus and readable
  labels. Scroll every field above fixed mobile controls; no horizontal page overflow.
- [ ] Keep one obvious primary action. All prior actions must still be reachable.
- [ ] Add a case named with the batch id to
  `web/tests/e2e/demo-polish-phase2.spec.ts`, following `demo-polish.spec.ts`. Capture
  both widths into `test-results/screenshots/phase2-<batch>-<route>-<width>.png` and call
  `expectAccessible`. Assert required content; no conditional skip of the main journey.
- [ ] Assert no external HTTP requests or page errors in the changed journeys. Keep
  failure simulation isolated from real-model success captures.
- [ ] Inspect the PNGs. Fix clipping, misplaced sticky controls, unreadable text and
  competing primary actions before committing.

Run these commands from `web/`, in order, checking each exit status:

```powershell
npx prettier --check src tests
npm run typecheck
npm run lint
npm test
npm run build
$env:DEMO_E2E = '1'
npx playwright test demo-polish-phase2 --project=desktop --grep B01 --output=test-results/phase2-playwright
```

Replace only `B01` with the exact batch id in the table. Rebuild the local web container
before browser checks so screenshots are of the edited source:

```powershell
docker compose -f deploy/compose.yaml -f deploy/compose.dev.yaml --env-file deploy/env/compose.env build web
docker compose -f deploy/compose.yaml -f deploy/compose.dev.yaml --env-file deploy/env/compose.env up -d --no-deps web
```

Run Docker commands from the repository root. Finish builds before browser tests to
avoid competing with local inference. From `C:\programming\5th-semester` run:

```powershell
python automation/check_rules.py ai-model/simpanugm
```

Then inspect `git diff --check`, stage explicit paths, update `demo-polish.md` and
`CHANGELOG.md` for that batch, and make the local Conventional Commit listed below.
No push or PR. Do not proceed past a failed check without fixing it or recording a
verified pre-existing blocker and leaving the batch uncommitted.

## 4. Screen batches, execute in order

Paths below are relative to `web/src/`. Every row requires Gate G and its named browser
case. Shared components may affect more pages; rerun the canonical reference suite when
changing one. Do not combine two rows into one implementation commit.

| Batch and local commit | Screens and exact files | Apply | Observable completion criteria |
|---|---|---|---|
| B01 `feat: polish public help and legal views` | `/bantuan`, `/privasi`, `/ketentuan`: `app/(public)/{bantuan,privasi,ketentuan}/page.tsx`; `app/(public)/layout.tsx`; `components/layout/legal-page.tsx` | Match landing spacing and subdued header/footer; grouped post information, existing legal content | All three pages readable at 360 px, post links and opening hours retained, unofficial service notice visible; existing sign-in and register entry preserved |
| B02 `feat: polish sign-in and verification views` | `/masuk`, `/daftar`, `/verifikasi`, `/konfirmasi-email`: corresponding `app/(public)/<route>/page.tsx`; `components/auth/{auth-card,login-form,register-form,verify-email,confirm-email}.tsx` | Texture behind white auth card, clear validation and loading; preserve demo role buttons | Login, registration, verification and email confirmation still work; expired token has recovery action; credentials and email never appear in logs or screenshots outside demo fixtures |
| B03 `feat: polish account recovery views` | `/lupa-sandi`, `/atur-sandi`, `/undangan`: corresponding `app/(public)/<route>/page.tsx`; `components/auth/{forgot-form,set-password-form,password-input}.tsx` | Same auth surface, visible password state and inline error | Valid and expired links covered, password visibility toggle named, no success before the real response |
| B04 `feat: polish member discovery and lost reporting` | `app/(member)/{beranda,laporan,barang-temuan,lapor/hilang}/page.tsx`; `components/report/report-card.tsx`; `components/report/form/lost-report-form.tsx` | Prioritized home action, candidate cards, scanable lists, existing assistant and form steps | Lost report can be created without a photo; search/filter and pagination work; public list still exposes no private photos/descriptions; draft survives reload |
| B05 `feat: polish member report details and editing` | `app/(member)/laporan/{hilang,temuan}/[code]/{page.tsx,ubah/page.tsx}`; `components/report/{lost-report-view,found-report-view,timeline,detail-list}.tsx` | Surface hierarchy, restrained timeline, existing finder hint and candidate list | Detail, edit, close and photo zoom work; hidden details remain owner scoped; conflicts preserve edits; canonical candidate detail remains unchanged |
| B06 `feat: polish member claims and notifications` | `/klaim`, `/klaim/baru`, `/klaim/[code]`, `/notifikasi`: corresponding member pages; `components/claims/{claim-card,claim-form,claim-body,claim-thread,member-actions}.tsx`; `components/notifications/{notification-list,notification-item}.tsx` | Group claim evidence and status, keep secure code legible, empty states | Submit, reply, withdraw and receipt confirmation work; code only in permitted claim state; notification link and mark-read behavior retained |
| B07 `feat: polish member profile settings` | `app/(member)/profil/{page.tsx,keamanan/page.tsx,notifikasi/page.tsx}`; `components/profile/*.tsx`; `components/notifications/preferences-form.tsx` | Settings groups and confirmation hierarchy | Profile edits, sessions, password, email and preferences retain validation; destructive account deletion still requires confirmation; do not execute deletion on a non-demo account |
| B08 `feat: polish officer review and search workspaces` | `app/(officer)/petugas/{page.tsx,papan/page.tsx,cari/page.tsx,duplikat/page.tsx}`; `components/officer/{review-board,duplicate-list}.tsx` | Quiet counters, priority queue, `ComparisonPanel`, four evidence bars, keyboard filters | Seeded queue is actionable; open a real pair on the board, pin/unpin, rematch and invite remain reachable; semantic search and duplicate rejection/merge work; board stacks at 360 px |
| B09 `feat: polish officer custody workspaces` | `/petugas/terima`, `/petugas/inventaris`: corresponding officer pages; `components/officer/{receive-panel,custody-actions,due-label}.tsx` | Task-focused receive surface and status/retention groups | Scan/manual code fallback, receive, storage slot, transfer, retention and disposition retain confirmation and validation; inventory filters work |
| B10 `feat: polish officer claim verification and handover` | `app/(officer)/petugas/klaim/{page.tsx,[code]/page.tsx}`; `components/claims/{officer-panel,action-dialog,claim-thread}.tsx` | Separate verification evidence, actions and `HandoverForm` on existing claim detail | Request info, approve/reject, checklist, six-digit code, lock state and handover work; never expose full ID data; no new handover route |
| B11 `feat: polish officer proxy reports and records` | `app/(officer)/petugas/{lapor-hilang/page.tsx,lapor-temuan/page.tsx,laporan/hilang/[code]/page.tsx,laporan/temuan/[code]/page.tsx}`; shared report form/view components | Reuse canonical form, preserve proxy identity and post/slot fields | Both proxy reports submit, staff-only evidence stays scoped, low-tier candidate toggle and moderation actions retained |
| B12 `feat: polish admin monitoring and fusion controls` | `app/(admin)/admin/{page.tsx,ai/monitoring/page.tsx}`; `components/admin/model-version-activator.tsx` | Group actual metrics, warnings, table alternatives and fusion controls | Real filters and model/version indicators retained; activate and rollback are deliberate confirmed actions; no synthetic accuracy or fake drift; strings moved to messages |
| B13 `feat: polish admin clusters maps and trends` | `app/(admin)/admin/{ai/klaster,peta-rawan,tren}/page.tsx`; `components/admin/{campus-hotspot-map,cluster-run-trigger,export-csv-form}.tsx` | Bounded charts/map, scanable clusters, empty states and visible table alternatives | Cluster rerun, member links, suppressed small counts, map controls, trend filters and CSV export work without external runtime calls; no invented precision |
| B14 `feat: polish admin users audit and settings` | `app/(admin)/admin/{pengguna,audit,pengaturan}/page.tsx`; `components/admin/{edit-user-dialog,invite-dialog,user-moderation-dialog,settings-form}.tsx` | Dense tables and restrained dialogs with explicit verb labels | Invite/edit/suspend scopes, audit filters/export and settings validation retained; scroll areas keyboard reachable; destructive changes confirmed |
| B15 `feat: polish category and location administration` | `app/(admin)/admin/master/{page.tsx,kategori/page.tsx,lokasi/page.tsx}`; `components/admin/master/{master-nav,category-dialog,compat-dialog,location-dialog,master-dialog}.tsx` | Consistent master-data shell and form dialogs | Search, add/edit, hierarchy and compatibility rules retained; inactive values stay identifiable; errors remain linked to controls |
| B16 `feat: polish zone and post administration` | `app/(admin)/admin/master/{zona,pos}/page.tsx`; `components/admin/master/{zone-dialog,post-dialog,toggle-active}.tsx`; `components/admin/post-checklist.tsx` | Same master-data controls and opening-hour group | Zones, post mapping, opening hours and activation validation work; no hardcoded timezone or silent destructive action |
| B17 `feat: polish global feedback and navigation states` | `app/{error,not-found}.tsx`; `components/layout/{member-shell,staff-shell,offline-banner,verify-banner,page-header,pager}.tsx` | Consistent focus, empty/error hierarchy, offline explanation; do not redesign canonical content | Error retry and 404 work, all role navigation remains reachable at both widths, unread label includes visible capped badge, offline drafts recover; no nested `PageTransition` |

Brace notation in file paths means edit only those named existing files. B02 and B03
also run `npx playwright test j1-register` on the running stack. B04 to B11 rerun
`npx playwright test demo-polish demo-ai demo-record --project=desktop` with demo mode on.
Run clock-advancing claim journeys only in their configured e2e stack, with one worker
and no parallel sessions. If full `make e2e` is needed, clear `DEMO_E2E`, run it alone,
then restore normal demo configuration with `make dev`, `make demo`, `make demo-check`.
Never use an e2e fake-clock stack for the presentation without restoring it.

## 5. Batch execution and commit order

1. Read the row's route and component files, not the entire repository. Confirm actual
   translated headings and API shapes before writing selectors.
2. Implement one row. Reuse the recipes; keep API and model behavior intact.
3. Run Gate G, its batch screenshot case and the extra tests above when applicable.
4. Review images and fix defects. Update the batch checklist in `demo-polish.md`, the
   affected demo script instructions and `CHANGELOG.md` under `[Unreleased]`.
5. Run the semester rule checker, inspect the diff, commit that row locally. Check
   `git status --short`. Do not push. Repeat B01 through B17.
6. Execute D01 below separately. Do not intermingle photo licensing and UI changes.
7. Finish with `make demo-check`, the three reference tests, both demo suites, a bundle
   comparison using the method in `phase1-validation.md`, and the final report.

If repeated login diagnostics reach HTTP 429, record it and wait for the displayed
window or restart only the local test API before a fresh suite. Do not raise production
login limits or change authentication rules. Do not run build processes alongside the
browser/model verification on a memory-constrained laptop.

## 6. D01, expand the demo photo pool safely

Target about 150 total images, including existing hero/derived assets. At the start of
phase 2 the manifest had 52 images across 19 categories; D01 later expanded it to 159.
No image download was performed in phase 1.

### Repair collection before using it

- [ ] Read `data/demo/fetch_public.py`, `validate_manifest.py`, `generate_credits.py`
  and `SOURCES.csv`. Current fetch script only has `--offline` and `--dest`, a fixed
  query list, and rewrites metadata for files it skips. Do not run it unchanged on the
  curated manifest, and do not assume a `--count` flag exists.
- [ ] Make fetching incremental: load existing rows by filename, retain every existing
  author/license/source URL, fetch only new names, merge rows, and write atomically.
  Keep old files unchanged unless a separately justified correction is needed.
- [ ] Record failed fetches honestly. Locally drawn placeholders must use
  `source=team_fixture` and a clear placeholder note. Never label them as Wikimedia
  downloads or count them as new photographic diversity.
- [ ] Add a tested `--target 150` option that grows only background queries, preserves
  hero pairs and stops at the target. Add tests in `data/demo/test_fetch_public.py`
  using mocked network metadata for existing-row preservation, missing-license
  rejection, duplicate handling, local fixture labelling and target bounds.
- [ ] Add license/source filtering and safe image checks before saving. Wikimedia
  Commons official API and Open Images only. Allowed: CC0, CC-BY, CC BY-SA and public
  domain. No identifiable people, real IDs or private text. Inspect each accepted image.
- [ ] Keep source, creator, license and per-image source link in `SOURCES.csv`. Preserve
  derived links and modifications. Cap around 300 files and 150 MB after resizing;
  current validator expects a 320 px short edge and at most 1024 px long edge.

Permission was granted only for 19 to 20 October 2026. If execution is later than that,
ask the owner to renew image-download permission before any network collection. Continue
the UI batches while that answer is pending. Do not infer extended permission from this
playbook or substitute other sources.

### Fill sparse categories first

| Phase 1 starting count | Categories | Priority |
|---|---|---|
| 1 each | mouse, kabel-charger, kotak-pensil, sepatu, topi | First bring each to at least 6 distinct usable object photos |
| 2 | jam-tangan | Next bring to at least 6 |
| 3 each | kalkulator, payung, jaket, kacamata, flashdisk, power-bank, helm | Add diversity in shape, color and background |
| 4 each | earphone, dompet, kunci-motor, tumbler, buku, ransel | Fill remaining target with diverse examples, not repeated crops |

Count original photos separately from derived pairs and fixtures. Report actual counts
instead of claiming every manifest row is a distinct real object.

### Integrate the pool into the seed

- [ ] Inspect `api/internal/demo/stories.go` and `demo.go`. The seed currently uses
  hardcoded photo filenames; adding files to the manifest does not enlarge search data.
  Phase 1 reset produced 21 found reports, not a 150-report search pool.
- [ ] Add deterministic background discovery from eligible manifest rows in that demo
  package. Keep the existing hero story, actor accounts, claims and history intact.
  Use stable keys and realistic existing categories/locations. Skip derived lost-photo
  rows and explicitly reserved holdout images. Never override model scores or embeddings.
- [ ] Add a seed test for eligible-row selection, stable keys, missing files and
  preservation of hero references. This is the necessary API-directory exception;
  avoid OpenAPI/SQL/schema changes for a data-seeding task.
- [ ] Run the following from the repository root, after the downloader repair:

```powershell
python -m unittest discover -s data/demo -p 'test_*.py'
python data/demo/fetch_public.py --target 150
python data/demo/validate_manifest.py
python data/demo/generate_credits.py
make test-api
make check-data
make dev
make demo
make demo-check
```

The network command requires the date-valid permission above. `make dev` rebuilds the
changed seed image; `make demo` alone may otherwise run the old binary. Inspect the
manifest, generated credits and photo bytes before staging. Run Gate G and the real
demo suites too. Commit D01 separately as `feat: expand the licensed demo search pool`.

### Verify a new, unseeded query photo

1. Reserve at least one licensed original photo as a holdout in its manifest notes and
   ensure seed selection excludes it. Verify its filename/hash is not among seed media.
2. Keep same-category but distinct-object examples in the search pool. Sign in as a
   member and create a new lost report using the holdout photo and natural Indonesian
   details through the normal form. Do not pre-create its match in the seed.
3. Wait for the real worker. Open resulting candidates and inspect actual score,
   category, four reasons and missing signals. Also try without a photo and through
   semantic search. Record top candidates and implausible results; do not promise a
   high match merely because a category is the same.
4. Edit one suggestion manually and submit; verify the edit persists. Test model failure
   and unavailable candidates separately. Capture both widths and axe results.
5. Record the sample filename, source, actual outputs, latency and limitations in
   `demo-polish.md`. Demo samples are not an official accuracy evaluation. Run
   `make demo` again before preparing the presentation story.

## 7. Documentation and final checks

- [ ] `docs/project/demo-script.md`: actual current labels and routes, both member and
  staff flows, honest model mistakes and retry instructions. Keep Indonesian speaker
  quotes, write new explanatory documentation in English.
- [ ] `docs/project/demo-polish.md`: mark each batch complete only with checked evidence;
  list route-specific issues left over and the actual pool counts.
- [ ] `docs/ux/ui-guidelines.md`: update only for a genuinely reusable decision. New
  dependencies require measured size and CSS alternatives in `design-research.md`.
- [ ] `CHANGELOG.md`: concise English entries under `[Unreleased]`.
- [ ] Compare reference screenshots with phase 1; all actions and protected data rules
  still hold. Verify current build is in the container before screenshots.
- [ ] Confirm ignored screenshots and videos stay out of git. No generated API artifacts
  should change for presentation work. If OpenAPI or sqlc input changes were unavoidable,
  run Docker sqlc generation, `go generate ./internal/openapi/`, then `npm run gen` in
  `web/`, and verify no generated delta remains uncommitted.
- [ ] Before each commit, run Gate G and the semester rule checker. No `--no-verify`.
  No `CLAUDE.md`, `agent-notes/`, `automation/`, secrets, agent credit or attribution.
- [ ] No em dash, en dash, emoji, remote fonts, runtime CDN or copyright asset copying.
  Code comments are sparse, one line, lowercase first letter, no final period.
- [ ] Demo mode must be `NEXT_PUBLIC_DEMO_MODE=true` at build time; never flip its default.
  Keep `proxy.ts`; never add `web/src/middleware.ts` beside it.
- [ ] npm locally, existing pnpm lockfile in CI. Do not create a second lockfile. The
  clock runs during tests, one Playwright worker, no parallel time-travel journeys.
- [ ] Use the existing photo helper's hydration retry. Avoid long shell heredocs on
  Windows. Ensure Git Bash is on the current process PATH when running make scripts.
- [ ] Finish at a clean, tested local commit. Never push, create or merge a PR.

## 8. Open questions for the owner

None blocks the UI batches. Resolve these before their dependent work:

1. If D01 runs after 20 October 2026, may the bounded Commons/Open Images download
   permission be renewed, and until which date?
2. Keep the original Lucide landing composition, or later supply two team-owned campus
   photographs? The current composition requires no new images and remains the default.
3. When will the team supply distinct second photos of its own hero items and review
   the misclassified earbuds example? Do not correct the screenshot by inventing output.
4. Which laptop, presentation date and speakers will be used for the final rehearsal?

## 9. Phase 2 final report format

Write a short Indonesian report containing:

1. Local commit hash and one-line result for each batch. State that nothing was pushed.
2. Screens completed, any incomplete screen and the concrete blocker.
3. Actual results for formatting, typecheck, lint, unit counts, build, Playwright and axe.
4. Screenshot links for desktop and phone, what improved and what still looks weak.
5. New dependencies and measured raw/gzip client bundle delta, or explicitly none.
6. Photo counts by original/derived/fixture, source/license validation, seeded report
   count, holdout result and actual model limitations. Do not call demo output accuracy.
7. Workflow, deploy, e2e or out-of-scope files changed and why.
8. Remaining owner questions, clean working tree status and the exact next action.
