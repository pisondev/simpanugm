# Visual and interaction research

Reviewed 20 October 2026. Scope: the public landing, owner candidates, and found report
flow. Decisions extend the existing blue palette, Indonesian copy, 8 px controls and
12 px cards. They do not change model output, permissions, or report workflows.

## Evidence and decisions

| Pattern | Source | Fit for SimpanUGM | Decision and reason |
|---|---|---|---|
| Quiet navigation, strong task hierarchy, generous grouping | [Linear, March 2026](https://linear.app/now/behind-the-latest-design-refresh) | Members need one clear next step; officers need dense evidence | Adopt the hierarchy principle. Keep existing navigation and use spacing, headings and bordered evidence panels |
| Layered shadows, consistent surface roles, subtle borders and radius | [Geist materials](https://vercel.com/geist/materials) | Separates report evidence from page chrome | Modify. Three semantic elevation tokens, low opacity blue shadows, 8 px controls, 12 px cards, 24 px hero. No copied code or assets |
| Short purposeful feedback, optional motion | [Apple motion](https://developer.apple.com/design/human-interface-guidelines/motion) | Upload and AI completion need feedback without delaying typing | Adopt. 120 ms press, 160 ms reveal, 200 ms score growth. All fields and reasons exist immediately |
| Functional motion rather than spring overshoot | [Material motion physics, May 2025](https://m3.material.io/styles/motion/overview/how-it-works) | A lost item is a practical, sometimes stressful task | Modify to CSS easing under the project's 200 ms cap. Reject an expressive spring runtime and bouncing controls |
| Distinct display, title, body and label roles | [Material typography](https://m3.material.io/styles/typography/applying-type) | Landing can be expressive while forms stay readable | Adopt role separation with existing Inter. Local variable Latin font, 16 px inputs, tabular scores, restrained uppercase eyebrow only |
| Content-shaped skeletons | [NN/g skeleton screens](https://www.nngroup.com/articles/skeleton-screens/) | Predictable cards reduce uncertainty while loading | Adopt existing shapes; add a transform-only shimmer. Stop decorative looping after three cycles and keep text status available |
| Inspectable explanations and honest limitations | [NN/g explainable AI, December 2025](https://www.nngroup.com/articles/explainable-ai/) | Match scores must not imply proven ownership | Modify for matching: show real total, four evidence bars, missing signals and editable suggestions. Reject invented processing stages and fake confidence |
| Limited choices per decision | [Hick's law](https://lawsofux.com/hicks-law/) | Category suggestions should make the form easier | Adopt two alternative category chips. Keep complete category search and manual entry available |
| Transform and opacity animation | [web.dev animation guide](https://web.dev/articles/animations-guide) | Laptop CPU is also running inference | Adopt. Fixed bar layout, scaleX fills, no animated height, width, blur or shadow. No permanent will-change |
| Feature-detected page transitions | [web.dev SPA transitions](https://web.dev/learn/css/view-transitions-spas) | Orientation between list and candidate detail | Modify. An opt-in post-commit content transition uses native snapshots when supported, with immediate fallback. Do not intercept links, hold navigation, or depend on experimental React APIs |
| Reduced motion is a live preference | [web.dev reduced motion](https://web.dev/articles/prefers-reduced-motion) | Vestibular comfort and testable accessibility | Adopt CSS overrides plus a subscribed JS preference. Count and bars expose final values to assistive technology throughout |
| Familiar list rows, status and locally saved drafts | [APJII 2025 reports](https://survei.apjii.or.id/survei/2025), [2026 availability](https://survei.apjii.or.id/survei/2026) | Campus use includes phones and interrupted connectivity | Adopt as a product hypothesis, not a measured claim about UGM. Keep 360 px layouts, existing draft persistence and retry paths. WhatsApp-style scanable rows and marketplace-style step grouping are familiarity hypotheses to validate with classmates; no invented usage statistics. The 2026 portal has no uploaded report at review time |
| Soft gradients and a local patterned hero | [Geist materials](https://vercel.com/geist/materials) | Gives the product warmth without public item disclosure | Modify. Original CSS campus-like route pattern and Lucide composition, no external images, fake reports, fake totals or photographic backgrounds behind text |
| Toast versus actionable feedback | [Apple feedback](https://developer.apple.com/design/human-interface-guidelines/feedback) | Users must not miss a failed upload or inference | Keep neutral success toasts and inline recoverable errors. Ready suggestions get a polite confirmation and editing hint; unavailable models leave manual input usable |

The source principles inform original implementation. No proprietary assets, icons,
illustrations, UI source code, or image downloads are used. Refactoring UI, Stripe and
Notion are not dependencies or copied templates. This scope is deliberately narrower than
an exhaustive comparison of products.

## AI interaction contract

- Photo analysis: a real request drives the loading status. No fake percent or sequence
  claiming OCR or classification finished. At eight seconds use the existing unavailable
  state. Description generation has the same bounded request budget.
- Suggestions: populate empty fields immediately, show an editable-default label, and
  stagger only decorative highlights by at most 30 ms. Never wait to accept input.
- Candidates: real overall score counts visually on first visibility. Accessible text
  always contains the final score. Four meters retain exact values or existing level
  proxies, never claim that those proxies are exact numerical model signals.
- Review: reusable comparison panels label each side, stack on narrow screens, and keep
  reasons and actions independent. Phase 2 applies the panel to the officer board.
- Finder: a quiet search hint lists only the existing permitted item and category data.
  No ownership assertion, private evidence, or invented match count.

## Performance budget

- Target 60 fps on an ordinary laptop, approximately 16.7 ms per frame. This is a target,
  not a measured guarantee. Use transform/opacity, at most one observer per animated
  primitive, disconnect after first visibility, cancel frames and listeners on unmount.
- No new npm dependencies. Motion code budget: at most 5 KiB gzip of authored source;
  compare production client JS before and after in the phase 1 validation report.
- Entrance duration at most 200 ms; stagger delay at most 120 ms total. Count-up at most
  200 ms. Do not delay route navigation, API requests, form values or focus.
- Shimmer is decorative, clipped to a reserved content rectangle, and finite. Never use
  a skeleton as the only description of loading or expose decorative frames to readers.
- Keep images on their existing protected same-origin API paths with reserved aspect
  ratios, lazy load secondary photos, preserve blurred identity variants.
- No runtime font, image, CDN or analytics dependency. Ship the existing Inter Latin
  subset locally with its SIL OFL license. CSS patterns add no image requests.

## Motion accessibility checklist

- [ ] Reduced motion at initial render: no count-up, shimmer, reveal or page transition
- [ ] Change preference during animation: cancel frames and display final values
- [ ] No content or action withheld while animation runs or JavaScript is unavailable
- [ ] Final meter values, missing-signal text and explanation always accessible
- [ ] Keyboard focus visible; no focus moves caused by decorative reveals
- [ ] No animation moves layout; text and bars reserve their final dimensions
- [ ] High contrast semantic tokens; no status conveyed only through color or motion
- [ ] Scroll regions keyboard reachable; touch controls tested at 360 px
- [ ] axe WCAG 2.1 AA scans after loaded desktop and phone states

Validation results and measured bundle deltas belong in
`docs/ux/phase1-validation.md`; do not mark this checklist as verified before running it.
