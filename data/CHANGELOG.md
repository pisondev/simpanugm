# Data changelog

## Unreleased

- Added the first deterministic master-data seed foundation for categories, colours,
  compatibility, brands, synonyms, and runtime setting defaults.
- Added a dependency-light command entry point for validating the seed foundation and
  reporting the still-deferred location, post, and model-version files.
- Added an OpenStreetMap/Overpass extractor that writes named location candidates to an
  explicit human-review CSV without filling zones or aliases automatically.
