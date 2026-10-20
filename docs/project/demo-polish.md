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
The photo pool remains at 52 files and requires the guarded D01 work in the playbook.

## Guardrails

Do not replace real model output, expose private fields, weaken the blurred identity
photo rule, fake data totals, add a runtime CDN, or delay editable values for animation.
Keep untranslated user copy out of components. Keep all existing actions and route
permissions. Hosting remains on hold.
