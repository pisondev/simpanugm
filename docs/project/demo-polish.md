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
  field highlights. Analysis failure keeps manual entry available.

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

## Guardrails

Do not replace real model output, expose private fields, weaken the blurred identity
photo rule, fake data totals, add a runtime CDN, or delay editable values for animation.
Keep untranslated user copy out of components. Keep all existing actions and route
permissions. Hosting remains on hold.
