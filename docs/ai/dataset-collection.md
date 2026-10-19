# Evaluation Dataset Collection | SimpanUGM

| | |
|---|---|
| Owner | Axelle Chandra |
| Reviewer | Irfan Fadilah Rafif |
| Dataset | `simpanugm-eval`, version 1 |
| Freeze date | Sunday 15 November 2026 |

No public dataset pairs a lost item report with a found item report on an Indonesian
campus. We build one. It is small, but it is real: real objects, real phone photos in
real campus places, descriptions written the way owners and finders actually write.

## 1. Ethics and consent

| Rule | Detail |
|---|---|
| Only consenting contributors | Items belong to team members or to friends who fill the consent form (Google Form, linked from `data/eval/README.md`). Consent covers use for evaluation and the course report until 31 December 2027, and deletion on request |
| No people | No face, hand with a ring or tattoo, or reflection of a person in any photo. Retake instead of cropping |
| No real identity documents | Identity items are mock cards printed by the team with fictitious names, the word CONTOH across them and 16 digit numbers that start with `0000`. Never photograph a real KTM, KTP, SIM or bank card |
| No personal data on screens | Phones and laptops are photographed powered off or on a neutral lock screen |
| Metadata stripped | EXIF, including GPS, is removed before upload by the intake script, and the quality check refuses a file that still has it |
| Access | The team Drive folder is shared with the three members only |

## 2. Item plan

150 items, chosen to cover the categories that are lost most and to include both generic
and distinctive objects.

| Parent category | Leaf examples | Items | Of which generic |
|---|---|---|---|
| Botol dan tumbler | botol minum, tumbler | 20 | 10 |
| Payung | payung lipat, payung panjang | 15 | 9 |
| Dompet dan tas kecil | dompet, pouch, tempat kartu | 15 | 6 |
| Elektronik kecil | earphone, TWS dan casing, charger, power bank, flashdisk | 25 | 10 |
| Kunci | kunci motor, kunci kos, gantungan kunci | 10 | 5 |
| Tas | ransel, tote bag, tas laptop | 10 | 3 |
| Pakaian | jaket, hoodie, topi, jas almamater | 10 | 4 |
| Helm | helm half face, full face | 8 | 3 |
| Alat tulis dan buku | kalkulator, kotak pensil, buku catatan | 9 | 4 |
| Aksesori | jam tangan, kacamata, gelang | 8 | 2 |
| Identitas (mock) | KTM, KTP, kartu ATM | 10 | 0 |
| Lainnya | botol obat, mainan kunci, alat olahraga | 10 | 4 |
| **Total** | | **150** | **60** |

`generic` means a person could not tell it apart from similar items in a photo without
handling it, for example a plain black umbrella. Two members label distinctiveness
independently, disagreements are settled together, and Cohen's kappa is reported.

Distractors: 400 more found reports of items that have no lost counterpart, collected
with the same photo protocol, roughly following the same category mix. Duplicates: for 40
items, a second member files an independent found report with their own photos and text.

## 3. Photo protocol

| Photo | Who takes it | Where and how |
|---|---|---|
| Finder photos, 2 to 3 per item | a member who does not own the item | where it would be found on campus: a desk in a lecture room, a library table, a bench, a canteen table, a mosque shoe rack, a parking area. Natural light, phone camera default settings, item fills 30 to 70 percent of the frame, different angles |
| Owner photo, 0 or 1 per item, for about 60 items | the owner | at home or in a kos room, on another day, the way a person would photograph their own belongings. Different background and light from the finder photos |

Every photo is JPEG, at least 1080 px on the short side, unedited apart from EXIF removal.
Blurry or overexposed photos are retaken, except 10 percent of finder photos that are
deliberately taken in poor light to keep the set honest. Those carry the label
`low_light`.

## 4. Description protocol

| Text | Writer | Rules |
|---|---|---|
| Owner description | the owner, from memory, without looking at the item | 1 to 3 sentences in everyday Indonesian, slang and English brand names allowed. One private detail written separately, the kind of thing only the owner would know |
| Finder description | the finder, while looking at the item | 1 to 2 sentences, only what is visible. Any hidden detail noticed (contents, a name inside) written separately |
| Item name | each writer separately | 2 to 5 words, for example "botol minum biru" |
| Category | each writer separately, from the app's category list | disagreements are kept, they are part of the realism |

Writers do not read each other's text. A member never writes both texts for one item.

## 5. Labels

`labels.csv`, one row per item:

| Column | Values |
|---|---|
| `item_id` | `I001` to `I150` |
| `leaf_category` | slug from `data/seed/categories.yaml` |
| `primary_colour`, `secondary_colour` | palette names from `data/seed/colours.csv`, secondary may be empty |
| `brand` | brand as printed, empty if none visible |
| `distinctiveness` | `generic`, `distinctive` |
| `has_owner_photo` | `true`, `false` |
| `flags` | any of `low_light`, `small_in_frame`, `damaged` |

Colour and brand are labelled by one member and checked by a second. The paraphrase
queries for T8 are written by a member who did not write either description of the
target item.

## 6. Layout and versioning

The data lives in the team Drive folder `simpanugm-eval`. Only the manifest is in git.

```
simpanugm-eval/v1/
  manifest.json                 every file with relative path, size, sha256, generator parameters, seed
  items.csv                     item id, item names, descriptions, private details, owner category
  labels.csv
  distractors.csv
  duplicates.csv
  queries.csv
  images/I001/finder_1.jpg
  images/I001/finder_2.jpg
  images/I001/owner.jpg
  images/D001/finder_1.jpg
```

| Rule | Detail |
|---|---|
| In git | `data/eval/manifest.json` and `data/eval/README.md` only |
| Download | `make eval-data` pulls the files listed in the manifest into `data/eval/`, which git ignores, and verifies every hash |
| Version | the dataset version is the SHA-256 of `manifest.json` |
| Changes after freeze | a new version `v1.1` with a changelog entry in `data/eval/README.md`. Runs on different versions are never compared in one table |

## 7. Quality checks

`make eval-check` runs before freezing and in CI on the manifest:

- every manifest file exists and matches its hash
- no file has EXIF data
- every image is at least 1080 px on the short side
- no two images share a SHA-256, and no two images of different items are within 4 bits of
  perceptual hash
- every label value is in its allowed domain
- every item has at least two finder photos, an owner description and a finder description
- split balance per parent category and distinctiveness within 10 percent of the target

## 8. Schedule

| Week | Target |
|---|---|
| 19 to 23 Oct | consent form live, protocol trial with 20 items, mock identity cards printed |
| 24 Oct to 6 Nov | 150 items photographed and described, labels done |
| 7 to 13 Nov | 400 distractors, 40 duplicates, 50 queries, quality checks green |
| 15 Nov | freeze `v1`, publish manifest hash in `data/eval/README.md` |

All three members photograph. Axelle runs intake, labels and checks. Irfan reviews a 20
percent sample before the freeze.

## 9. Master data from the same effort

While walking campus for photos, the team records the locations used and checks them
against OpenStreetMap: official name, common aliases students use, zone, and the
coordinate of the main entrance. That list becomes `data/seed/locations.csv`, see
[data/README.md](../../data/README.md).

## 10. Human collection checklist and freeze protocol

The physical dataset collection and private labeling require physical human presence on campus.
The team tracks collection progress against the following checklist before executing the freeze:

### 10.1 Human collection checklist

- [ ] **150 Physical items**: gathered across 12 parent categories (60 generic, 90 distinctive).
- [ ] **Finder photos**: 2 to 3 photos per item taken in realistic campus locations (at least 1080 px short side, natural light, no people or reflections, EXIF stripped).
- [ ] **Owner photos**: 1 photo each for approximately 60 items taken in residential/private settings.
- [ ] **400 Distractor found reports**: single finder photos with campus locations and metadata, matching the target category distribution.
- [ ] **40 Duplicate pairs**: independent found reports for 40 items filed by two distinct team members.
- [ ] **50 Paraphrase queries**: natural officer query formulations targeting specific items, written by a member who did not draft the item descriptions.
- [ ] **Metadata files prepared**: `items.csv`, `labels.csv`, `distractors.csv`, `duplicates.csv`, `queries.csv` in `/private/simpanugm-eval/v1/`.
- [ ] **Consent forms**: signed contributor forms archived in private Google Drive.

### 10.2 Post-collection freeze commands

Once data collection in `/private/simpanugm-eval/v1` is complete, execute the validation and freeze pipeline in order:

1. **Verify manifest formatting and image resolution**:
   ```bash
   python3 data/scripts/check_manifest.py --dir /private/simpanugm-eval/v1
   ```
2. **Verify inter-annotator agreement (distinctiveness and category kappa)**:
   ```bash
   python3 data/scripts/check_agreement.py --first /private/sheet_member1.csv --second /private/sheet_member2.csv
   ```
3. **Generate and verify deterministic stratified splits (60 val, 90 test)**:
   ```bash
   python3 data/scripts/split_eval.py \
     --labels /private/simpanugm-eval/v1/labels.csv \
     --output /private/simpanugm-eval/v1/splits.csv \
     --distractors /private/simpanugm-eval/v1/distractors.csv \
     --distractor-output /private/simpanugm-eval/v1/distractor-splits.csv \
     --val-count 60 \
     --seed 2026
   ```
4. **Execute auxiliary file checks and generate freeze verification report**:
   ```bash
   python3 data/scripts/check_auxiliary.py --dir /private/simpanugm-eval/v1
   python3 data/scripts/check_freeze.py \
     --dataset-dir /private/simpanugm-eval/v1 \
     --profile full \
     --split /private/simpanugm-eval/v1/splits.csv \
     --distractor-split /private/simpanugm-eval/v1/distractor-splits.csv \
     --val-count 60 \
     --output /private/simpanugm-eval/v1/freeze_report.json
   ```
5. **Publish public manifest to git repository**:
   ```bash
   python3 data/scripts/publish_manifest.py --source /private/simpanugm-eval/v1 --output data/eval/manifest.json
   ```
6. **Fetch files and run official benchmark evaluation**:
   ```bash
   python3 data/scripts/fetch.py --manifest data/eval/manifest.json --source /private/simpanugm-eval/v1 --output data/eval
   cp /private/simpanugm-eval/v1/freeze_report.json data/eval/freeze_report.json
   make eval-real
   ```
   The `make eval-real` command automatically verifies `data/eval/freeze_report.json`,
   executes fusion calibration on `val`, evaluates acceptance gates on `test`, and
   generates `ml/eval/runs/<run-id>/REPORT.md` with status `RESMI`. If the freeze report
   is missing or indicates validation failures, execution is immediately aborted.
