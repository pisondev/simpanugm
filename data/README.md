# data

Master data seeds, fixtures shared by `api` and `ml`, the evaluation dataset pipeline, and
reference analytics queries.

Owner: Axelle Chandra. Status: foundation in progress. Seeds and the dataset are built in
Sprints 0 to 2.

| Folder | Holds | In git |
|---|---|---|
| `seed/` | campus master data loaded by `simpan seed` | yes |
| `fixtures/` | small test inputs and expected outputs used by both `api` and `ml` | yes |
| `eval/` | the evaluation dataset card and manifest | only `README.md` and `manifest.json` |
| `scripts/` | intake, quality checks, download, OpenStreetMap extraction | yes |
| `analytics/` | reference SQL for the analytics views | yes |

How the evaluation dataset is collected: [docs/ai/dataset-collection.md](../docs/ai/dataset-collection.md).

## Current stage

The deterministic seed foundation is available in [`seed/`](seed/) and is checked by
[`scripts/check.py`](scripts/check.py). The foundation check covers the category tree,
verification questions, colour palette, compatibility factors, brand aliases, synonyms,
and runtime defaults. The campus location/post list and model-version seed remain pending
the OpenStreetMap review, security-unit input, and the inference-service lock file.

```text
python3 data/scripts/check.py
```

The complete-mode check is intentionally expected to fail until those reviewed files are
added:

```text
python3 data/scripts/check.py --require-complete
```

The data-specific history is kept in [`data/CHANGELOG.md`](CHANGELOG.md).

Sprint 0 collection preparation is in [`data/protocol/`](protocol/), including the
consent draft, mock identity-card rules, 20-item checklist, and label template.

For the location review, save an Overpass response and run:

```text
python3 data/scripts/osm_locations.py \
  --input /path/to/overpass.json \
  --output /path/to/locations-review.csv
```

The generated rows have empty `zone_slug` and `aliases`, and carry
`review_status=needs_review`. They are review material, not production seed data.

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
It can verify both existing assignments with `--verify` and `--verify-distractors`. The
split files stay private with the dataset; only the script and its tests are committed.

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

## Seed files

These formats are a contract with `api` (CONTRIBUTING section 2). The seed command upserts
by natural key and never deletes.

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

The campus clusters as the university names them, plus central facilities:

| `slug` | `name` |
|---|---|
| `saintek` | Klaster Sains dan Teknologi |
| `soshum` | Klaster Sosial Humaniora |
| `agro` | Klaster Agro |
| `medika` | Klaster Kesehatan |
| `pusat` | Fasilitas Pusat |

Faculty to cluster assignment follows the university's official list and is checked
before the seed is merged.

### Locations

Built with `scripts/osm_locations.py`, which pulls named buildings inside the campus
boundary from OpenStreetMap, then reviewed by hand: official name, the names students
actually use as aliases, zone, kind, and the coordinate of the main entrance. Every
location must be marked `approved` and fall inside the campus bbox passed to
`scripts/promote_locations.py`. The promotion step checks zone and kind values and
writes only the seven columns accepted by the seed contract. Candidates still marked
`needs_review` cannot be promoted.

### Posts for the pilot

One post per zone plus one at the central library, each tied to a location. The list is
replaced by the security unit's actual posts when the system is adopted.

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
