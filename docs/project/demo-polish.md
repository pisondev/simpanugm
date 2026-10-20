# Demo polish status

Phase 1 reference implementation, 20 October 2026.

## Reference screens

- Public `/`: patterned Lucide hero, direct found report entry, existing registration
  and sign-in, three steps, privacy and unofficial student-project disclosure.
- Owner lost detail candidate list and candidate detail: fixed-ratio protected photos,
  animated final overall score, four evidence meters, missing-signal text, tier, claim,
  positive feedback and dismissal. A score is a similarity signal, not proof of ownership.
- `/lapor/temuan`: existing validation, draft, duplicate check and custody flow with an
  AI explanation before upload, real analysis status, editable suggestions and subtle
  field highlights. Analysis failure keeps manual entry and an explicit retry available.

## Foundation ready for phase 2

Semantic elevation and surface tokens, `auth-texture`, reusable empty states, motion
utilities, count-up, visible score fills, native cross-document transitions and
`ComparisonPanel`. The auth texture and officer comparison wrapper are available for
adoption; full auth and officer screen redesigns remain phase 2 work.

Reference implementation: `web/src/components/landing/landing-page.tsx`,
`web/src/components/matching/`, `web/src/components/report/form/` and
`web/src/components/motion/primitives.tsx`.

Validation evidence: [phase1-validation.md](../ux/phase1-validation.md).
Execution plan: [demo-polish-playbook.md](demo-polish-playbook.md).

## Verified phase 1 evidence

Formatting, typecheck, lint, 74 unit tests and production build pass. The combined
`demo-ai`, `demo-polish` and `demo-record` run passes all 11 tests, including axe scans.
Eleven reference PNGs cover desktop, 360 px phone and reduced-motion failure states.
Demo preflight passes 17 checks; no npm dependency was added. The emitted client chunk
delta is 5,925 gzip bytes across the whole build, not a per-route first-load estimate.

The actual photo model misclassified the earbuds reference as Charger. Correction is
manual and stays editable. Broader model accuracy work is outside this visual phase.

## Phase 2 screen work

B01 polishes `/bantuan`, `/privasi` and `/ketentuan` with the existing legal content,
the subdued public shell, grouped post hours and responsive information cards. Sign-in,
registration, phone links when a post supplies a number, and the unofficial project notice
remain available. Desktop and 360 px Playwright captures passed axe with no external
requests or page errors. Captures are in the ignored `web/test-results/screenshots/`
directory with the `phase2-B01-` prefix.

B02 applies the local auth texture behind the white auth card and preserves editable form
validation, password visibility, demo role entry and token recovery. Desktop and 360 px
captures cover sign-in, registration, missing and expired verification links, and missing
and expired email confirmation links. The batch axe and external-request checks pass, and
the existing `j1-register` suite passes all four tests. Captures use the `phase2-B02-`
prefix in the ignored screenshot directory.

B03 reuses that auth surface for password recovery and invitation acceptance. Desktop and
360 px captures cover missing and usable reset links, missing invitations and the password
setup form. A real Mailpit reset link changed a newly registered test account only after
the API accepted it; mismatch and invalid-token feedback remained inline. The existing
officer-invitation journey passed, as did all three B03 axe and offline-request cases.
Captures use the `phase2-B03-` prefix in the ignored screenshot directory.

B04 gives the member home a clear lost-report action, improves report-card scanning and
surfaces the lost form and public found filters on semantic cards. Category filtering,
cursor pagination and reset navigation work against the seeded API. The public endpoint
and rendered cards contain no photos, descriptions, private details or finder identity.
The lost-report flow still accepts no photo; a real report was submitted and its draft
fields survived reload. Desktop and 360 px captures passed axe, keyboard visibility,
overflow and no-external-request checks. The test also found and fixed mobile grid overflow
and action-bar overlap on the review step. Captures use the `phase2-B04-` prefix in the
ignored screenshot directory. The B04 case uses a newly verified test member so repeated
demo runs do not consume the seeded owner's open-report limit.

B05 groups report details and history into quieter surfaces, keeps owner-only evidence
visually distinct, and adds page transitions to lost/found detail and edit routes. The
existing candidate detail is untouched. Playwright verifies seeded lost and found records,
owner-only private fields, denied cross-member API reads, photo zoom at desktop and phone
sizes, a two-tab edit conflict that retains the unsaved review values, and closing with an
explicit selected reason. Both B05 scenarios pass axe, horizontal overflow and offline
request checks; screenshots use the `phase2-B05-` prefix. The local API limiter required
restarting the API between the registration/edit and seeded-account scenarios.

B06 groups claim status, evidence, secure handover code, and the officer conversation into
clear surfaces, and adds visible retry feedback when notifications fail to load. The
desktop and 360 by 800 Playwright journey covers claim details and permitted code access,
the new-claim form, notification links, mark-read, and mark-all-read. All four routes pass
axe, viewport overflow, and no-external-request checks. Mobile screenshots use the actual
360 by 800 viewport; the detail conversation and photo-evidence control are scrolled above
the fixed bottom navigation before capture. The full claim lifecycle suite passes 7/7 in
the E2E stack, including reply, withdraw, receipt, and retention disposition. Its fake
clock test requires `deploy/compose.e2e.yaml`; the ordinary demo stack correctly returns
404 for that test-only endpoint. The demo was reset afterward and `make demo-check` passed
17/17. Captures use the `phase2-B06-` prefix.

B07 groups profile contact details, password and active sessions, and notification choices
into consistent surfaces. Its desktop and phone journey saves profile fields, rejects an
invalid email change without replacing the current address, checks short and mismatched
password errors, revokes a second session, and changes then restores a real notification
preference. Sign-out-all and account deletion dialogs both show their confirmation copy and
are canceled; the deletion action is never submitted. All three routes pass axe, overflow
and no-external-request checks. Phone screenshots use the 360 by 800 viewport and place
the profile save and preference save controls above the fixed navigation. The test writes
only to the demo owner and restores the preference value. Captures use the `phase2-B07-`
prefix.

B08 refines the officer priority queue, review board, semantic search and duplicate
workspace. Queue summary counters are quieter, report codes in the board's search field
resolve to the selected record, and side-by-side evidence uses the shared comparison
panel. Search guidance is visible to assistive technology; duplicate rows wrap without
horizontal overflow at 360 px. The E2E case opens a real seeded candidate pair, checks
the four evidence meters and invite action, pins and unpins it, runs keyboard semantic
search, and verifies duplicate rejection, merge controls and manual merge fields. All
four screens have desktop and 360 px captures, axe and overflow checks, with no external
requests or page errors. The B08 case passes 1/1; web Prettier, typecheck, lint, 74 unit
tests and production build pass. Captures use the `phase2-B08-` prefix.

B09 makes the intake lookup scanner-friendly while retaining manual typing/paste and
visible error feedback. Inventory keeps its post and sort filters, uses a focused
horizontal table on desktop and scanable item cards with custody actions on phones.
Transfer to another active post and one-time retention extension now call the existing
API with required confirmation context and validation; eligible unclaimed items retain
the disposition dialog. The B09 browser case passes 1/1 with axe and overflow checks at
both widths, exercises invalid manual lookup, inventory sorting and guarded action
dialogs, and uses no external requests or page errors. This reset demo has no report
awaiting physical receipt, so the happy-path intake was verified in the real Sprint 1
browser journey (7/7). B09 Prettier, typecheck, lint, 74 unit tests and production build
pass. Captures use the `phase2-B09-` prefix.

The real-model reference gate initially exceeded the eight-second assist bound. ML metrics
isolated EasyOCR as the main cost: 1,549 seconds across 258 analyses (6.0 seconds each),
versus 1.65 seconds total for category classification. Capping OCR input at 768px reduced
the OCR detector area by 44% while retaining the real model and its output. A real-weight
OCR check still reads the synthetic card title and masks its full number at that size. In
the rebuilt service, 134 analyses averaged 3.17 seconds in OCR and 3.46 seconds end to end.
The phase 1 reference and reduced-motion tests pass 3/3, B01-B09 pass 14/14, demo-ai passes
4/4, and demo-record passes 4/4. A combined run hit the in-memory login limiter (HTTP
429 confirmed in API logs); all four affected tests passed after restarting the local API
and rerunning the affected suites separately. No model output, UI timeout or production
rate limit was changed.

The post-B04 reference, AI and recording checks passed 19/19 across two runs. The local
API's in-memory sign-in limiter required an API restart before the four recording scenes;
the cold model's first desktop suggestion also exceeded its existing eight-second bound,
then passed when rerun with the model warm. No production rate limit or AI timeout was
changed.

## Phase 2 data work completed

D01 expanded the manifest from 52 to 159 local images: 140 Wikimedia Commons originals,
12 derived hero pairs and 7 team fixtures. The verified pack is 15.01 MB. The seeder
adds 90 distinct eligible background photos after excluding the reserved holdout and
duplicate image bytes, producing 111 found reports and 19 lost reports. The successful
demo reset computed 72 matches, 5 claims, 1 duplicate and 18 clusters. One additional
demo-only officer account distributes proxy uploads within the existing per-account
limit; API rate limits were not changed.

The reserved holdout `bg-flashdisk-011.jpg` stays out of the seeded reports. Through the
normal lost-report form, the live assistant suggested Earphone at 0.8996 and Flashdisk
at 0.0327. The report was corrected to Flashdisk and the visible blue and white colors.
The real matching worker returned no candidate for the holdout. This is one demo sample,
not an accuracy evaluation. On the report detail, the searching message remains until a
page refresh even after the two-minute window; handle this with the member detail batch.

The verified D01 assets and attribution are in [demo-credits.md](demo-credits.md), and
live image model outputs are recorded in [demo-photo-check.md](../ai/demo-photo-check.md).
The holdout browser run passed axe and used no external requests. Its desktop capture is
`web/test-results/screenshots/d01-holdout-desktop.png`.

## Guardrails

Do not replace real model output, expose private fields, weaken the blurred identity
photo rule, fake data totals, add a runtime CDN, or delay editable values for animation.
Keep untranslated user copy out of components. Keep all existing actions and route
permissions. Hosting remains on hold.
