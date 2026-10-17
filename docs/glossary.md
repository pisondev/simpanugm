# Glossary | SimpanUGM

Domain terms with the Indonesian word used in the interface and the identifier used in
code.

| Term | In the interface | In code | Meaning |
|---|---|---|---|
| Lost report | Laporan kehilangan | `LostReport`, `lost_reports` | A member's report of something they lost, with a time window and up to three places |
| Found report | Laporan temuan | `FoundReport`, `found_reports` | A report of something found, with at least one photo and exactly one place |
| Owner | Pemilik, pelapor | `reporter_id` | The member who filed a lost report |
| Finder | Penemu | `finder_id` | The member who filed a found report |
| Proxy report | Laporan atas nama | `proxy_*` columns | A report an officer files for a walk in visitor without an account |
| Category | Kategori | `Category` | Two level classification. Leaves carry sensitivity, retention, verification questions and prompts |
| Sensitivity | Sensitivitas | `sensitivity` | `normal`, `valuable` or `identity`. Drives custody rules, retention, photo visibility and priority |
| Zone | Klaster, zona | `Zone` | A campus cluster grouping locations |
| Location | Lokasi | `Location` | A building, outdoor area, facility, transport point or parking area with a coordinate |
| Post | Pos | `Post` | A security or Lost and Found desk where items are kept and handed over |
| Custody | Penyimpanan | `custody_status` | Where a found item physically is: with the finder, at a post, released, disposed, cancelled |
| Storage slot | Lokasi simpan | `storage_slot` | Free text shelf or box reference inside a post |
| Retention | Masa simpan | `retention_due_at` | How long a post keeps an unclaimed item before a disposition |
| Disposition | Tindak lanjut | `Disposition` | What happened to an unclaimed item: donated, destroyed, transferred to a unit, returned to the finder |
| Match | Kecocokan | `Match`, `matches` | A scored pair of one lost report and one found report |
| Candidate | Kandidat | a `Match` seen from one side | A match shown on a report, ranked by score |
| Overall score | Skor kecocokan | `overall` | The fused score in `[0, 1]`, shown as a percentage |
| Component score | Komponen skor | `s_visual`, `s_text`, `s_location`, `s_time` | The parts the overall score is built from |
| Category factor | Faktor kategori | `category_factor` | Multiplier for how related the two categories are |
| Visual mode | Mode visual | `visual_mode` | `image` when both sides have photos, `cross_modal` when only one side does |
| Tier | Tingkat | `tier` | `high` Sangat mirip, `medium` Mirip, `low` Kurang mirip |
| Fusion configuration | Konfigurasi skor | `fusion` model version | The versioned document holding every scoring constant |
| Embedding | Representasi vektor | `*_embeddings` | A vector produced by an encoder, compared by cosine similarity |
| Cross modal | Lintas modal | `cross_modal` | Comparing a text embedding with an image embedding in a shared space |
| Report assistant | Saran otomatis | `assist` | Suggestions of category, colour, brand and description from a photo |
| Visible text | Teks pada foto | `visible_text` | Text read from a photo by OCR, masked before storage |
| Hidden details | Detail tersembunyi | `hidden_details` | What a finder noticed but must not publish, such as contents |
| Private details | Ciri khusus | `private_details` | What an owner knows about their item that proves ownership |
| Claim | Klaim | `Claim`, `claims` | An owner's request to receive a found item, reviewed by an officer |
| Verification checklist | Daftar periksa verifikasi | `claim_checks` | The AI-08 items an officer resolves before approving |
| Handover | Serah terima | `handover` | The recorded moment an officer gives an item to an approved claimant |
| Handover code | Kode serah terima | `handover_code_hmac` | Six digit one time code shown to the approved claimant |
| Hint | Petunjuk pemilik | `match-hint` | What a finder sees when a likely owner exists, without identity |
| Duplicate | Duplikat | `duplicate_links`, `duplicate_of` | Two reports of the same type about the same item |
| Priority queue | Antrean prioritas | `officer/queue` | Officer work items ordered by the AI-09 score |
| Review board | Papan tinjauan | `officer/board` | Side by side view of a report and its candidates |
| Hotspot | Titik rawan | `analytics/hotspots` | Locations and times with the most reports |
| Model version | Versi model | `model_versions` | An identified, evaluated version of one AI component |
| Gate | Ambang penerimaan | evaluation section 8 | The metric thresholds a version must pass before activation |
| WIB | WIB | `Asia/Jakarta` | Western Indonesia Time, UTC+7, the display time zone |
