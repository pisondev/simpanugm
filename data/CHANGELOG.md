# Data changelog

## Unreleased

- Added the first deterministic master-data seed foundation for categories, colours,
  compatibility, brands, synonyms, and runtime setting defaults.
- Added a dependency-light command entry point for validating the seed foundation and
  reporting the still-deferred location, post, and model-version files.
- Added an OpenStreetMap/Overpass extractor that writes named location candidates to an
  explicit human-review CSV without filling zones or aliases automatically.
- Restricted the Overpass query to named campus-relevant location features to keep review
  output focused.
- Added the Sprint 0 collection protocol, consent draft, mock identity-card specification,
  20-item label template, and trial-label validator.
- Added private-dataset intake and manifest verification with EXIF stripping, filename
  normalization, SHA-256 hashes, and tamper detection.
- Added dataset quality gates for metadata completeness, photo coverage, owner-photo
  consistency, and duplicate descriptions.
- Added a deterministic evaluation splitter that stratifies by parent category and
  distinctiveness while preventing physical-item leakage between `val` and `test`.
- Added auxiliary dataset checks for distractor metadata and photos, duplicate-pair
  references, and unique paraphrase queries.
- Extended the evaluation splitter to assign and verify distractors at the same split
  proportion as the physical-item set.
- Added a guarded location-review promotion step that converts only approved OSM rows
  into the production location seed after zone and campus-bbox checks.
- Extended complete seed checking to validate zones, locations, posts, model components,
  cross-file references, opening-hour intervals, and active-version uniqueness.
- Added a safe private-template generator for the 150-item collection and its auxiliary
  metadata, refusing to overwrite non-empty directories.
- Added a trial profile that creates only the 20-item metadata files and image directory.
- Added canonical reference analytics materialized-view SQL for daily reports, hourly
  patterns, return times, and match outcomes, with concurrent-refresh indexes and
  suppression delegated to the API query layer.
- Added a no-placeholder handoff checklist for the deferred zone, location, security
  post, and model-version seed files.
- Made private dataset intake atomic so invalid source files cannot leave a partial
  prepared dataset directory.
- Hardened intake and manifest checks against symlink inputs, invalid manifest kinds or
  paths, malformed hashes, and disabled normalization policy flags.
- Extended dataset quality checks to reject normalized description reuse across owner and
  finder roles as well as across items.
- Restricted evaluation splitting to the stable one-based item and distractor ID formats.
- Hardened auxiliary metadata checks for one-based IDs and duplicate label references.
- Made full and trial template generation atomic so failed writes do not leave partial
  private collection layouts.
- Extended private intake and manifest checks to reject symlink directories as well as
  symlink files.
- Hardened location review promotion against non-finite bboxes and duplicate OSM
  provenance rows.
- Made location review and promotion CSV writes atomic to prevent partial output files.
- Made evaluation split CSV writes atomic to protect existing private assignments from
  partial output.
- Added shared synthetic fixture contracts for fusion scores, BR-PRV-03 masking, and
  rapidfuzz token-set name matching, with a dependency-light validator and test coverage.
- Added deterministic synthetic image fixtures for colour extraction, EXIF removal, OCR,
  and masking while keeping participant media outside the repository.
- Added a reviewed SHA-256 integrity check for committed synthetic image fixtures and wired
  it into the data CI workflow without relying on runner-specific image codecs.
- Added an intake integration test proving the shared GPS-tagged fixture is stripped before
  it reaches a prepared private dataset.
- Added a private template checker for stable IDs, exact columns, blank pre-entry fields,
  and empty image directories in the full and trial collection scaffolds.
- Clarified that the private evaluation manifest is committed only after the real intake
  and v1 freeze, so no empty or fabricated hash is published beforehand.
- Added a dedicated data CI workflow that installs the declared validator dependencies,
  checks the seed foundation and shared fixtures, and runs all data-script tests.
- Added an atomic manifest publisher that validates the private prepared dataset and
  copies only `manifest.json` into `data/eval/` at the reviewed freeze.
- Added an atomic Overpass raw-response output so location review rows retain an auditable
  source alongside the generated CSV without promoting unreviewed values.
- Added a pre-review location checker that verifies the generated CSV against its raw OSM
  response and rejects premature manual fields.
- Added bounded retries for transient Overpass timeouts and server errors while leaving
  client errors fail-fast.
- Restricted generated location-review candidates to the requested bbox, including when
  processing a saved raw response.
- Classified named public amenities and institutional buildings as `facility` in the
  review CSV instead of leaving them as generic `building` rows.
- Added a counts-only private location-review summary to help plan manual review without
  copying candidate names into the repository.
- Extended private manifest quality checks to reject images below the 1080 px short-side
  minimum and exact duplicate image content before dataset freeze.
- Added a cross-item perceptual-hash gate that flags visually near-identical images within
  four bits before dataset freeze.
- Hardened split verification to reject manual edits that change the stratified `val`
  allocation for a parent-category and distinctiveness group.
- Added a private two-annotator agreement checker that reports aggregate Cohen's kappa
  without copying label sheets or participant data into the repository.
- Hardened location promotion to require valid OSM provenance and at least one approved
  location before writing a production seed.
- Added a blank 20-item trial exit-review log template for consent, privacy, retake, and
  disagreement tracking outside git.
- Added an atomic private trial-review workspace generator for two annotation sheets and
  the exit-review log.
- Added a final private trial-review gate for valid annotations, pass statuses, disagreement
  fields, and the recorded private Drive path.
- Bound the final trial gate to the generated agreement report so stale or missing kappa
  results cannot pass review.
- Added a read-only freeze orchestrator that runs the trial/full dataset, auxiliary, split,
  and review gates before manifest publication.
- Extended complete seed validation to require the reviewed campus bbox and reject locations
  outside it.
- Made private image intake canonicalize accepted image inputs to JPEG before hashing.
- Added a local-only evaluation fetcher that copies manifest-listed files from a private
  Drive sync or mount, verifies hashes, rejects symlinks and stale files, and preserves
  the repository dataset card.
- Hardened manifest publishing and private fetches so their output cannot be placed inside
  the source dataset directory.
- Made foundation seed checks reject partial additions of deferred complete-seed files
  until the explicit complete-mode bbox gate is used.
- Made manifest publication print the final SHA-256 needed for the dataset card and
  reproducible evaluation records.
- Hardened manifest checks to reject symlink dataset roots and symlink manifests before
  reading private files.
- Hardened seed validation against symlink roots and seed files that could redirect checks
  outside the repository checkout.
- Bound location promotion to the saved raw Overpass response so approved review fields
  cannot detach from their source OSM element.
- Made negative campus-bbox examples portable across supported Python CLI parsers by
  using the `--bbox=...` form in tests and handoff documentation.
- Rejected manifest-listed paths that traverse symlinked directories, keeping private
  dataset validation inside the dataset root.
- Applied symlink-parent checks to private dataset intake, fetch, and manifest
  publication before any source or output path is resolved.
- Extended symlink-parent checks to private template/workspace generation and their
  validators, keeping all collection roots tied to the requested directory.
- Tightened category-seed validation for empty trees, field types, prompt/question text,
  and optional positive retention overrides.
- Rejected normalized collisions between brand names/aliases and between synonym terms,
  preventing ambiguous master-data lookups.
