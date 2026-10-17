# Risk Register | SimpanUGM

Reviewed at every sprint review. Probability and impact on a scale of 1 to 3, score is
their product. A risk with score 6 or more has an owner who reports on it each review.

| Id | Risk | P | I | Score | Owner | Mitigation | Trigger to act |
|---|---|---|---|---|---|---|---|
| R-01 | Matching quality on real campus photos is below the gates | 2 | 3 | 6 | Irfan | baseline first, ablations show where it fails, error analysis drives fixes, the gates are honest and a miss is reported with its cause | first gate run in Sprint 2 below any gate |
| R-02 | CPU inference too slow on the VPS, OCR above all | 2 | 2 | 4 | Irfan | latency spike in Sprint 0, OCR at 1024 px, River `ml` concurrency, OCR switch, ONNX option | p95 match pipeline above 60 s in the first load test |
| R-03 | Scope too large for nine weeks next to six other courses | 3 | 3 | 9 | Pison | scope control order of project-plan section 6, velocity checked at each review, Should stories scheduled late | velocity below 80 percent of plan in Sprint 1 or 2 |
| R-04 | One person carries most of `api` and `web` | 3 | 2 | 6 | Pison | work packages 9 and 14 move to Axelle, Irfan pairs on scoring, generated code reduces boilerplate | Pison's open issues above 8 at mid sprint |
| R-05 | Evaluation dataset late or too small | 2 | 3 | 6 | Axelle | collection starts in Sprint 0, all three photograph, minimum viable set of 100 items with 250 distractors if needed, documented as such | fewer than 100 items by 6 November |
| R-06 | Leakage between `val` and `test` inflates results | 1 | 3 | 3 | Irfan | grouped split by item, automated leakage test, test touched only at gates | leakage test failing |
| R-07 | Personal data exposure through a projection bug | 2 | 3 | 6 | Pison | role specific response types, snapshot tests per role, security review before v1.0 | any snapshot test failing or a review finding |
| R-08 | Fraudulent claims in the pilot | 2 | 2 | 4 | Pison | hidden details, checklist, ID in person, code, claim limits | any rejected claim with reason `suspected_fraud` |
| R-09 | Time zone bugs, UTC shown as local time | 2 | 2 | 4 | Pison | UTC everywhere, one formatter pinned to `Asia/Jakarta`, tests across midnight UTC | a displayed time differs from the stored instant by seven hours |
| R-10 | Go, Python and TypeScript integration drifts | 2 | 2 | 4 | Pison | OpenAPI contracts both ways, generated clients, shared fixtures for fusion | generated code diff in CI or a fixture mismatch |
| R-11 | VPS or domain not available in time | 1 | 2 | 2 | Pison | decided in Sprint 0, staging up in Sprint 1, laptop demo stack as fallback | no staging by 6 November |
| R-12 | Model weight downloads fail or a revision disappears | 1 | 2 | 2 | Irfan | weights cached in a volume and mirrored in the team Drive with their hashes | download failure in CI or on a server |
| R-13 | Lecturer expectations of "production ready" differ from our checklist | 2 | 3 | 6 | Pison | prd section 9 made explicit, progress shown in class with the dashboard and the pipeline, adjust after feedback | lecturer feedback in any class session |
| R-14 | HEIC and large photos from phones break uploads | 2 | 1 | 2 | Pison | client side conversion and compression, server signature checks, tested on an iPhone and an Android phone | upload error rate above 2 percent on staging |
| R-15 | Spam or abusive reports on a public deployment | 1 | 2 | 2 | Pison | university email registration, rate limits, report limits, moderation | any hidden report |
| R-16 | OpenStreetMap tile usage policy limits | 1 | 1 | 1 | Axelle | low traffic, proper attribution, a tile provider with a free tier as fallback | tile errors in the browser console |
| R-17 | Team member unavailable for a week | 2 | 2 | 4 | Pison | ownership documented, every area has a second reviewer who can continue, no knowledge outside the docs | anyone announces an absence |

## Closed risks

None yet. A risk moves here with the date and the reason when it can no longer happen.
