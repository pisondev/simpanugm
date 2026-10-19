# data

Master data seeds, fixtures shared by `api` and `ml`, the evaluation dataset pipeline, and
reference analytics queries.

Owner: Axelle Chandra. Status: MVP public-source baseline. Seeds and the private dataset are
built in Sprints 0 to 2.

| Folder | Holds | In git |
|---|---|---|
| `seed/` | campus master data loaded by `simpan seed` | yes |
| `fixtures/` | small test inputs and expected outputs used by both `api` and `ml` | yes |
| `eval/` | the evaluation dataset card and, after private freeze, its manifest | `README.md` now; `manifest.json` is added at freeze |
| `scripts/` | intake, quality checks, manifest fetch, OpenStreetMap extraction | yes |
| `analytics/` | reference SQL for the analytics views | yes |

How the evaluation dataset is collected: [docs/ai/dataset-collection.md](../docs/ai/dataset-collection.md).

## Current stage

The deterministic seed foundation and a small public-source MVP baseline are available in
[`seed/`](seed/) and are checked by [`scripts/check.py`](scripts/check.py). The baseline is
for the assignment/demo only; it is not an official production campus master-data approval.
It contains five project zones, 25 named locations selected from 392 OpenStreetMap
candidates, one officially published K5L emergency contact, and public model revisions or
package versions matching the AI documentation. Private trial photos, consent records, and
participant metadata remain outside Git.

The MVP location bbox is `-7.77525,110.36967,-7.76617,110.38783` (south, west, north,
east). It is a reproducible conversion of the UTM grid corners shown on the [official UGM
campus map](https://www.ugm.ac.id/downloads/PETA%20REVISI%204-1.pdf), not a claim that this
is the final production boundary. Zone centroids are arithmetic means of the selected
location rows. Location aliases are intentionally empty until a local review supplies them.
The sole post is the central K5L office; its emergency number comes from [UGM's official
contact page](https://ugm.ac.id/id/kontak/) and its hours are empty because no official
hours are published.

The location candidates came from the [OpenStreetMap Overpass API](https://overpass-api.de/).
The model seed records exact public revisions for [CLIP](https://huggingface.co/sentence-transformers/clip-ViT-B-32/commit/327ab6726d33c0e22f920c83f2ff9e4bd38ca37f),
[multilingual CLIP](https://huggingface.co/sentence-transformers/clip-ViT-B-32-multilingual-v1/commit/58edf8cada9e398793dca955574a48cbb7f18be2),
and [multilingual E5](https://huggingface.co/intfloat/multilingual-e5-small/commit/614241f622f53c4eeff9890bdc4f31cfecc418b3),
plus the documented [EasyOCR](https://pypi.org/project/easyocr/1.7.2/) and
[RapidFuzz](https://pypi.org/project/RapidFuzz/) package versions. This is an MVP baseline;
it does not replace a production `ml/models.lock.json`.

```text
python3 data/scripts/check.py --require-complete \
  --bbox=-7.77525,110.36967,-7.76617,110.38783
```

The foundation-only command is still useful for a checkout that has none of the four
complete seed files, but it intentionally rejects this branch because the MVP files are
present. Use the complete-mode check above for the current branch.

The foundation check also fails if any complete seed file appears by itself. This prevents
CI from accepting a partial location, post, or model-version seed without complete-mode
validation and a campus bbox. It also rejects symlinked seed roots and files so validation
cannot silently read data from outside the checkout, and reports malformed YAML as a seed
validation error.

Category validation requires non-empty parent and leaf names, positive parent ordering,
non-empty text prompts and question fields, boolean question requirements, and positive
optional retention overrides. Seed, trial, and auxiliary CSV readers also reject rows
whose field count does not match the declared header or required fields.
Brand names and aliases are checked case-insensitively with normalized whitespace, and
synonym terms use the same normalized uniqueness rule to prevent ambiguous lookup keys.
Allowed email domains must be valid, non-empty, and unique; reviewed location aliases
also reject blank entries and normalized duplicates.
Complete post seeds require string location references, non-empty phone strings, and
non-overlapping ordered opening intervals, while model-version components and statuses
must use their declared strings.

The data-specific history is kept in [`data/CHANGELOG.md`](CHANGELOG.md).

Shared contract fixtures are now available in [`fixtures/`](fixtures/). They contain only
synthetic scoring, masking, and name-comparison cases; participant data and collected
images remain outside the repository. Malformed JSON and case shapes are reported as
fixture validation errors. Validate them with:

```text
python3 data/scripts/check_fixtures.py
```

Synthetic image fixtures are available in `fixtures/images/` and can be recreated with
`python3 data/scripts/create_fixture_images.py`. CI also runs its `--check` mode to
compare committed bytes with their reviewed SHA-256 hashes, independent of local image
codecs. They contain no people, real documents, or participant data.

The repository data workflow runs the seed check, fixture check, and complete data-script
test suite on every change under `data/`. Private manifest verification still runs after
intake in the protected dataset workspace, because the manifest and its files are not in
Git before the v1 freeze. The manifest checker also rejects images whose short side is
under 1080 px, whose exact SHA-256 content is reused, or whose perceptual hash is within
four bits of an image from another item. Once the private dataset is approved, use
`data/scripts/publish_manifest.py` to copy only the validated manifest into `data/eval/`.
It also rejects symlinked dataset roots, manifest files, and any symlinked directory in
a listed path, so validation cannot read private files from outside the dataset root.
The private intake, fetch, manifest-publication, template-generation, and review-workspace
commands apply the same check to their source and output parent paths before resolving
them; the corresponding validators reject aliased roots as well.
The same output-root guard protects OSM review exports, promoted location seeds, review
summaries, agreement reports, evaluation splits, and synthetic fixture regeneration.

Prepared-dataset metadata readers reject rows with missing or extra fields before the
content quality checks run.

Reference analytics views are documented in [`analytics/README.md`](analytics/README.md)
and defined in [`analytics/views.sql`](analytics/views.sql). They are reviewed SQL, not
application migrations.

Sprint 0 collection preparation is in [`data/protocol/`](protocol/), including the
consent draft, mock identity-card rules, 20-item checklist, label template, and blank
trial review log template.

For an independent two-person label review, compare two private sheets and write only
aggregate agreement statistics:

```text
python3 data/scripts/check_agreement.py \
  --first /private/simpanugm-eval/trial-20/review/labels-a.csv \
  --second /private/simpanugm-eval/trial-20/review/labels-b.csv \
  --expected-count 20 \
  --output /private/simpanugm-eval/trial-20/review/agreement.md
```

The report contains observed agreement and Cohen's kappa for `leaf_category` and
`distinctiveness`; label sheets and the report remain outside git. The annotation and
review-log readers reject rows with missing or extra fields before the final trial gate
evaluates their statuses.

Prepare those sheets and the trial exit log atomically in a private directory with:

```text
python3 data/scripts/create_trial_review_workspace.py \
  --output /private/simpanugm-eval/trial-20/review
```

After collection and the manual privacy review, run the final trial gate:

```text
python3 data/scripts/check_trial_review.py \
  --workspace /private/simpanugm-eval/trial-20/review
```

For the location review, save an Overpass response and run:

```text
python3 data/scripts/osm_locations.py \
  --download \
  --bbox=SOUTH,WEST,NORTH,EAST \
  --raw-output /private/simpanugm-locations/overpass.json \
  --output /private/simpanugm-locations/locations-review.csv
```

Or, when the response has already been saved:

```text
python3 data/scripts/osm_locations.py \
  --input /path/to/overpass.json \
  --bbox=SOUTH,WEST,NORTH,EAST \
  --output /path/to/locations-review.csv
```

The bbox filters candidates by their extracted coordinates, including when a saved raw
response is processed. Download mode saves the raw response and review CSV atomically.
Generated rows have
empty `zone_slug` and `aliases`, and carry
`review_status=needs_review`. It retries transient Overpass errors twice by default.
They are review material, not production seed data.
Before a human starts editing the review, verify that the CSV still matches its raw
response:

```text
python3 data/scripts/check_location_review.py \
  --raw /private/simpanugm-locations/overpass.json \
  --review /private/simpanugm-locations/locations-review.csv \
  --bbox=SOUTH,WEST,NORTH,EAST
```

Location review and promotion readers reject rows with missing or extra fields before
provenance or zone approval checks run.

Create a private counts-only summary to plan the manual review without copying candidate
names into Git:

```text
python3 data/scripts/summarize_location_review.py \
  --review /private/simpanugm-locations/locations-review.csv \
  --output /private/simpanugm-locations/review-summary.md
```

After the private labels pass the dataset quality gate, create the reproducible,
item-grouped `val`/`test` assignment:

```text
python3 data/scripts/split_eval.py \
  --labels /private/simpanugm-eval/v1/labels.csv \
  --output /private/simpanugm-eval/v1/splits.csv \
  --distractors /private/simpanugm-eval/v1/distractors.csv \
  --distractor-output /private/simpanugm-eval/v1/distractor-splits.csv \
  --val-count 60 \
  --seed 2026
```

The splitter stratifies by parent category and distinctiveness, assigns each physical
item to exactly one split, and assigns distractors to the same `val`/`test` proportion.
Verification rechecks the expected `val` quota for every parent-category and
distinctiveness group, so manual edits cannot silently change the stratification. It can
verify both existing assignments with `--verify` and `--verify-distractors`. The
split files stay private with the dataset; only the script and its tests are committed.

Before publishing a manifest, run the read-only freeze gate so the relevant trial or full
profile checks cannot be skipped:

```text
python3 data/scripts/check_freeze.py \
  --profile trial \
  --dataset-dir /private/simpanugm-eval/trial-20/prepared \
  --review-workspace /private/simpanugm-eval/trial-20/review
```

For the full dataset, add `--split`, `--distractor-split`, and `--val-count 60`; the
command validates but does not publish or copy any private files. The profile routing and
private split-error handling are covered by `data/scripts/test_check_freeze.py`.

Once the full private dataset has its auxiliary reports, run the additional metadata
check:

```text
python3 data/scripts/check_auxiliary.py \
  --dataset-dir /private/simpanugm-eval/v1 \
  --expected-distractors 400 \
  --expected-duplicates 40 \
  --expected-queries 50
```

It validates the distractor IDs and photos, duplicate-pair references, and unique
paraphrase queries. The expected counts can be lowered for a local trial fixture.

For the full collection, create the private raw layout with
`data/scripts/create_eval_templates.py`; it generates stable IDs for all target rows
without adding any participant data to the repository. Run
`data/scripts/check_templates.py --profile full` before data entry to verify the blank
scaffold, including the exact field count of every row.

For the Sprint 0 trial, pass `--profile trial` to create only the 20-item workspace
needed by `check_dataset.py`. Verify it with
`data/scripts/check_templates.py --profile trial` before collection begins.

## Seed files

These formats are a contract with `api` (CONTRIBUTING section 2). The seed command upserts
by natural key and never deletes.

Run `python3 data/scripts/check.py --require-complete --bbox=-7.77525,110.36967,-7.76617,110.38783`
for the current MVP seed. In complete mode the checker validates CSV/YAML shape, zone and
location references, location coordinates inside the supplied campus bbox, post
opening-hour intervals, and the eight model components with exactly one active version
each.

| File | Format | Columns or keys |
|---|---|---|
| `zones.csv` | CSV, UTF-8, header | `slug`, `name`, `centroid_lat`, `centroid_lng` |
| `locations.csv` | CSV | `slug`, `name`, `zone_slug`, `kind`, `lat`, `lng`, `aliases` with aliases separated by `;` |
| `posts.yaml` | YAML list | `slug`, `name`, `location_slug`, `phone`, `opening_hours` per weekday as `[["07:00","16:00"]]` |
| `categories.yaml` | YAML tree | parents with `slug`, `name`, `children`. Leaves with `slug`, `name`, `sensitivity`, optional `retention_days`, `clip_prompts`, `verification_questions` of `key`, `question`, `required`, `check_hint` |
| `category_compatibility.csv` | CSV | `category_a`, `category_b`, `factor` |
| `colours.csv` | CSV | `name`, `hex`, `sort_order`, the 14 palette colours of matching-engine section 4 |
| `brands.csv` | CSV | `name`, `aliases` separated by `;` |
| `synonyms.csv` | CSV | `term`, `expansion` |
| `settings.yaml` | YAML map | the keys and defaults of business-rules section 10 |
| `model_versions.yaml` | YAML list | the initial version of each component with its identity and the first fusion document |

### Zones

The MVP groups campus locations into five project zones. Their stored centroids are the
arithmetic means of the five selected OSM locations in each zone:

| `slug` | `name` |
|---|---|
| `saintek` | Saintek |
| `soshum` | Soshum |
| `agro` | Agro |
| `medika` | Medika |
| `pusat` | Pusat |

These names and centroids are project-level MVP groupings, not an official UGM zoning
decision.

### Locations

Built with `scripts/osm_locations.py`, which pulled 392 named buildings and facilities
inside the MVP bbox from OpenStreetMap. Twenty-five rows were selected for the assignment
demo and promoted with their OSM names, coordinates, kinds, and provenance; aliases remain
blank because no student naming review was available. Every promoted row falls inside the
same bbox and is subject to the complete seed check. This small selection is not a claim
that it is the full campus inventory.

### Posts for the pilot

The MVP contains only the central K5L office and its official emergency phone. It is tied
to the OSM security-office location. Published opening hours were not found, so every day
is represented by an empty interval list; the post list must be replaced by the security
unit's reviewed inventory before adoption.

### Categories

| Parent | Leaves, with sensitivity |
|---|---|
| `identitas` Identitas dan Kartu | `ktm`, `ktp`, `sim`, `kartu-atm`, `kartu-lain`: identity |
| `dompet-tas-kecil` Dompet dan Tas Kecil | `dompet` valuable, `tempat-kartu` valuable, `pouch` normal |
| `elektronik` Elektronik | `ponsel`, `laptop`, `tablet`, `earphone` valuable. `charger`, `power-bank`, `flashdisk`, `kalkulator` normal |
| `kunci` Kunci | `kunci-motor` valuable, `kunci-lain` normal |
| `tas` Tas | `ransel`, `tote-bag`, `tas-laptop`: normal |
| `botol` Botol dan Tumbler | `botol-minum`, `tumbler`: normal |
| `payung` Payung | `payung`: normal |
| `pakaian` Pakaian | `jaket`, `topi`, `jas-almamater`, `pakaian-lain`: normal |
| `helm` Helm | `helm`: normal |
| `aksesori` Aksesori | `jam-tangan` valuable, `perhiasan` valuable, `kacamata` normal |
| `alat-tulis` Buku dan Alat Tulis | `buku`, `alat-tulis`: normal |
| `dokumen` Dokumen | `dokumen`: identity |
| `lainnya` Lainnya | `lainnya`: normal |

Compatibility pairs across or within parents: `dompet` and `tempat-kartu` 0.80, `dompet`
and `pouch` 0.70, `ransel` and `tas-laptop` 0.80, `botol-minum` and `tumbler` 0.90, `ktm`
and `kartu-lain` 0.70, `tote-bag` and `pouch` 0.50.

Verification question examples:

| Leaf | Questions | Check hint |
|---|---|---|
| `dompet` | Sebutkan isi dompet (required). Ciri khusus di luar atau dalam dompet (required) | Isi dompet yang disebut sesuai |
| `ponsel` | Merek, model, dan warna casing (required). Gambar layar kunci atau ciri casing (required) | Pengklaim dapat membuka kunci perangkat atau menunjukkan nomor IMEI |
| `ktm` | Nama dan NIM pada kartu (required) | Nama pada kartu sesuai identitas pengklaim |
| `kunci-motor` | Merek motor dan gantungan kunci (required) | Pengklaim menunjukkan STNK atau kendaraan yang cocok |
| `payung` | Warna, motif, dan bentuk gagang (required). Kerusakan atau tanda khusus (optional) | Ciri yang disebut terlihat pada barang |

## Fixtures

| File | Used by | Content |
|---|---|---|
| `fusion_cases.json` | `api` matching tests, `ml/eval` tests | component inputs and expected component scores, overall score and tier, to six decimals |
| `masking_cases.json` | `api` and `ml` | strings with identity and card numbers and their masked form |
| `fuzzy_cases.json` | `api` | name pairs and their rapidfuzz token set ratio |
| `images/` | `api` media tests, `ml` tests | tiny team made images: solid colour swatches, a GPS tagged photo of a desk, a mock card |
