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
