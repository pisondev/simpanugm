# Phase 1 validation

Environment: Windows, Node 22.23.2, npm, Docker Compose. Reviewed 20 October 2026.
Application entry: `http://localhost:8000`. Demo web built with
`NEXT_PUBLIC_DEMO_MODE=true`; default mode was not changed.

## Checks

| Check | Result |
|---|---|
| Baseline formatting, typecheck, lint, unit suite and production build | Passed before the research commit; 14 files, 69 tests |
| Foundation formatting, typecheck, lint, unit suite and production build | Passed; 15 files, 73 tests |
| Reference screen formatting, typecheck, lint, unit suite and production build | Passed; 15 files, 74 tests, including retry without overwriting manual input |
| `make demo` | Passed after adding Git Bash bin directories to the current PowerShell process PATH |
| Demo reset result | 7 users, 19 lost reports, 21 found reports, 22 matches, 5 claims, 1 duplicate pair |
| `make demo-check` | Passed: 17 checks, 0 failures, 0 warnings |
| `python data/demo/validate_manifest.py` | Passed: 52 images, 3.77 MB |
| Final combined Playwright run | 11 passed in 1.3 minutes: 4 real demo AI journeys, 3 reference/slow-failure scenarios and 4 recording journeys |
| axe WCAG 2.1 AA | 0 violations on all 11 reference captures and all scanned legacy demo/recording states |
| Runtime network and JavaScript errors | No external HTTP requests and no page errors in either successful reference journey |
| Rules and git hooks | `python automation/check_rules.py ai-model/simpanugm` passed; identity and guard current |

The browser plugin could not initialize because its execution runtime rejected a
missing sandboxPolicy field. Visual verification used the repository's Playwright suite
and direct inspection of generated PNGs. These are real application screenshots.

## Reproduce the reference checks

From the repository in PowerShell, if `sh` is not on PATH:

```powershell
$env:PATH = 'C:\Program Files\Git\bin;C:\Program Files\Git\usr\bin;' + $env:PATH
make demo
make demo-check
Set-Location web
$env:DEMO_E2E = '1'
npx playwright test demo-polish --project=desktop --output=test-results/phase1-playwright
```

The desktop project explicitly checks both 1366 by 900 and 360 by 800. Demo specs are
excluded from ordinary runs unless `DEMO_E2E=1`, and are not duplicated in the phone
project. The suite has one worker and never advances the shared clock. Run production
builds before browser tests to avoid memory contention with inference.

## Screenshot inventory and visual review

All PNGs are ignored under `web/test-results/screenshots/`; they are not committed.

| Stem after `phase1-` | View | Review |
|---|---|---|
| `landing-desktop`, `landing-phone` | Public hero, steps and privacy | Blue hierarchy, original local Lucide composition and clear CTAs. Phone action is visible before the decorative composition; no invented metrics |
| `candidate-list-desktop`, `candidate-list-phone` | Owner lost report with candidate | Tier, actual overall percentage, protected thumbnail, date, location and post. One candidate is sufficient for this fixture; larger pools remain phase 2 |
| `candidate-desktop`, `candidate-phone` | Owner evidence and claim actions | Entire item shown with contain-fit, four meters with reasons, honest score disclaimer and one primary claim action. Phone stacks evidence, so this is a long page |
| `found-photo-desktop`, `found-photo-phone` | Required photo step | Bounded upload area, explanation of editable assistance, existing camera/gallery and validation |
| `found-assist-desktop`, `found-assist-phone` | Real photo suggestion result | Ready state, alternative category chips, drafted text, editable fields and suggestion tags. Actual earbuds photo was classified as Charger, with Power Bank and Tumbler alternatives. This model mistake is preserved, not rewritten for a screenshot |
| `found-unavailable-reduced` | Deliberately failed analysis with reduced motion | Inline explanation, retry and manual fields stay usable; no fake successful result |

Visual changes after inspection: reduce competing primary actions, show the full item
rather than crop it, define the upload area, reveal mobile step labels, and remove excess
space below the stepper. Fixed mobile navigation and form controls remain viewport UI;
a full-page screenshot includes them at the captured viewport position.

The inference result quality needs separate model work. A correct-looking interface
does not establish classification accuracy. The seeded calculator's time reason also
has no level; the UI preserves the original missing level rather than invent a score.

## Motion and offline evidence

- Unit coverage checks reduced motion on first render, live preference change, frame
  cancellation, final accessible counts, visibility gating and observer cleanup.
- CSS suppresses movement, shimmer and native view transitions for reduced motion.
  The browser failure scenario checks the computed near-zero transition duration and
  edits the form while analysis is pending, then checks the edit survives failure.
- Native page transitions apply to same-origin full-document navigation. Next.js soft
  navigation remains immediate. No experimental framework flags or routing delay.
- Runtime request assertions cover the two reference journeys; model and photo data
  remain local. This is narrower than proving every route in the app has no external
  dependency. The admin map and other phase 2 screens need their own offline audit.
- No 60 fps performance trace was collected. Transform/opacity and a 200 ms maximum
  are implemented; 60 fps remains a budget, not a benchmark claim.

## Measured bundle baseline

Method: sum sizes of all `.next/static/chunks/*.js` and sum Python `gzip.compress` of
each file separately, on production builds from the same local npm installation. This
measures all emitted client chunks, not a per-route first-load or network transfer size.

| Measurement | Baseline | Reference build |
|---|---:|---:|
| Emitted JavaScript files | 69 | 71 |
| JavaScript bytes | 5,264,963 | 5,282,584 |
| JavaScript gzip bytes | 1,456,673 | 1,462,598 |

Final measured delta: 17,621 raw bytes, 5,925 gzip bytes (about 5.79 KiB, 0.41 percent).
Motion primitive source is 1,280 gzip bytes, below the 5 KiB authored-source budget.
Local font is 48,432 bytes, the same Latin subset already delivered before this change.
There are no new npm dependencies or lockfile changes. These measurements include the
post-review mobile stepper update.

## Issues found and addressed

- An existing asynchronous unit assertion checked the captured outcome before React
  committed it. The assertion now waits for the outcome as well as the generated text.
- The old phone notification accessible label did not contain the visible label and
  badge string in order. It now names the visible count, including the 99+ cap.
- Initial analysis exceeded the existing eight-second request cap while builds and
  inference competed. The manual fallback worked; explicit retry now recovers without
  overwriting user edits. No time budget was raised and no model result was mocked.
- One initial browser upload failed with `net::ERR_INSUFFICIENT_RESOURCES` during
  concurrent production builds. Sequential browser verification passed afterward.
- Vite reports an existing native-config-loader warning for the TypeScript config;
  tests pass. No runtime or package change was made solely to silence it.
- Legacy demo and recording specs referred to an absent `Earphone / Headphone`
  category and older officer, cluster and owner action labels. Their selectors now use
  current UI labels; assertions, actual model calls and screenshot captures remain.
- Repeated diagnostic runs reached the existing login throttle (HTTP 429, visible
  retry message). The local API is restarted before the final combined demo run to
  clear its in-memory test traffic counters; security settings are unchanged.
