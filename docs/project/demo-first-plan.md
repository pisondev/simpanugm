# Demo first plan | SimpanUGM

Decided on 19 October 2026 by Pison. Read by everyone before picking up work.

## 1. Decision

The course is graded mostly on the live demonstration: how the product feels to each kind
of user, how it looks, and whether the AI features visibly do something for each actor.
Technical depth and documentation weigh less. From now on the remaining time goes to a
demo that is smooth, good looking and convincing, on top of the platform that is already
built and tested. Hosting (VPS and domain) is on hold until Pison says otherwise, the demo
runs from a laptop with Docker Compose.

Nothing built so far is removed. Items that no longer have priority are parked, not
deleted.

## 2. What changes

| Raise | Keep as is | Park |
|---|---|---|
| One scripted demo story per actor, with realistic data and photos | API, web, ml, CI, e2e as they are today | VPS, domain, TLS, restore drill on a server |
| Visual and UX polish on every screen a demo touches | Real AI behind every demo screen, no mocked outputs | Formal usability study with 10 participants |
| AI made visible: suggestions, match explanation, finder hint, review board, monitoring, clusters, hotspots | Honest labels on data and numbers | Official evaluation on a large real dataset (`make eval-real` stays ready) |
| `make demo` that resets and seeds the whole story in one command | Security and privacy rules in the code | Traceability and definition of done polish |
| Fallback recording and a pre flight check for the live run | Master data seeds | Load test beyond the existing budgets |

## 3. The demo story

Four actors, each with one moment where the AI is visibly useful.

| Actor | Story | AI moment that must be visible |
|---|---|---|
| Finder (member) | Reports earbuds found in the library with one photo | The assistant fills category, colours and brand from the photo, drafts the public description, masks visible text, and a hint says an owner may be looking |
| Owner (member) | Reports a lost wallet without a photo, later finds it in the candidate list | Candidates with a tier, a percentage, four score bars and reason lines in Indonesian, the "maybe mine" answer, then claim with the category questions |
| Officer | Works the queue, compares lost and found side by side, verifies the claim, hands over with the code | Priority queue, review board with component scores, verification checklist suggestions, duplicate warning, semantic search over reports |
| Admin | Looks at the week | Matching monitoring with drift, clusters of related reports, hotspot map, trends, fusion version activation and rollback |

## 4. Data for the demo

- A curated set of about 12 hero scenarios, each a lost report and a found report of the
  same kind of object so that a visible match exists, plus about 40 background reports so
  lists, the map and the charts look alive.
- Photos come from two places: photos the team takes of its own belongings, and public
  images with a free licence. Every file is listed with source and licence in
  `data/demo/SOURCES.csv`. Nothing from social media, nothing with a recognisable person
  or a real identity document. Identity items are mock cards marked CONTOH.
- The second photo of a hero pair is a real second photo when the team has one. Otherwise
  it is a derived image of the same object (crop, angle, light, background) and the
  manifest says `derived`.
- Embeddings, analysis and scores are computed by the real models. Demo data is data, the
  AI is not faked.
- Demo accounts and demo data exist only where `simpan seed --demo` is allowed, it is
  refused in production already.
- Numbers shown in the demo are labelled with what they are. Results from synthetic or demo
  data say so on the slide and in the docs. An official figure is shown only after
  `make eval-real` produced one.

## 5. Workstreams

Each one has its done condition. Work is done in order, one commit per stage.

| Id | Work | Done when |
|---|---|---|
| W1 | `make demo`: reset, migrate, seed master data, load the demo story, compute embeddings and matches | One command gives a stack where every actor can follow section 3 without preparation. Re running it gives the same story. Accounts per actor are documented in `docs/project/demo-script.md` |
| W2 | Demo photo pack and manifest in `data/demo/` | 12 hero pairs and about 40 background items with `SOURCES.csv`, small file sizes, a script that validates licence fields and image sizes |
| W3 | UX and visual polish pass per actor | A checklist per screen is ticked: empty and loading states, mobile width, Indonesian copy, spacing, icons, focus states, consistent status colours. axe stays at zero violations |
| W4 | AI made visible | Every AI moment of section 3 is clear on screen, has a graceful state when the service is slow, and is covered by a Playwright journey with screenshots |
| W5 | Demo run kit | `docs/project/demo-script.md` with minutes, clicks and Indonesian speaker notes, a Playwright run that records video and screenshots of the happy paths as a fallback |
| W6 | Light usability pass | A one page kit for 3 to 5 classmates, 10 minutes each, the findings are fixed or listed. Optional, replaces the formal study |
| W7 | Live demo reliability | `make demo-check` verifies services, models loaded, seeded accounts, an end to end smoke over the story, and memory use on a laptop. No external network call is needed during the demo |

## 6. Who does what

| Person | Focus |
|---|---|
| Pison | Owns the demo, decides the story and the order, reviews W3 and W5 |
| Irfan | The AI moments in W4 and the numbers shown about them. If unreachable, the work is done by the team and written down here |
| Axelle | Demo data curation in W2, including photos of her own items, review of synonyms and brands, reading this plan |
| Everyone | W1, W3, W4, W5 and W7 are split by area as in section 5 and reviewed by Pison |

## 7. Minimum the team must still do by hand

- Photograph about 12 own items twice (finder style, owner style), 30 minutes in total.
- Walk through the demo script once as each actor and say what feels wrong.
- Decide the demo date, the machine it runs on and who speaks when.

## 8. Rules that do not change

Privacy and security rules in the code, honest labelling of data and figures, English docs,
Indonesian user interface, no secrets in the repository, tests before commits.
