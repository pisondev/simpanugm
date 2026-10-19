# Model Cards | SimpanUGM

Model cards for locked models specified in `ml/models.lock.json` per the lifecycle protocol
in [model-lifecycle.md](model-lifecycle.md).

Evaluation metrics requiring real campus data are marked TERTUNDA until the dataset freeze on 15 November 2026.

---

## 1. Image Encoder (`image_encoder`)

| Attribute | Specification |
|---|---|
| Model ID | `sentence-transformers/clip-ViT-B-32` |
| Component Kind | Neural visual encoder |
| Revision | `327ab6726d33c0e22f920c83f2ff9e4bd38ca37f` |
| Output Dimension | 512 (float32, L2 normalized) |
| License | MIT |
| Source | Hugging Face Hub / OpenAI CLIP architecture |
| Input Format | JPEG/WebP, resized to 224x224, bicubic interpolation, normalized with CLIP mean/std |
| Measured Latency | p50: 72 ms (1 image), 117 ms (2 images); p95: 78 ms (1 image), 129 ms (2 images) |
| Hardware Profile | Reference laptop (Windows 11, Intel 12 logical CPUs, 4 PyTorch threads) |

### Intended Use
Extract dense 512-dimensional visual feature vectors for:
1. Pairwise cosine similarity between finder photos and owner photos in matching pipeline (AI-01).
2. Zero-shot category suggestion prompt scoring against candidate prompt sets (AI-03).
3. Visual similarity clustering (AI-11).

### Limitations and Failure Modes
- Low contrast, severe blur, or heavy occlusions degrade embedding quality.
- Small objects occupying less than 20% of frame blend into background features.
- Generic items (e.g. plain black umbrellas, standard clear bottles) produce near-identical embeddings across distinct instances.
- Background bias: photos taken in distinct environments can reflect scene differences rather than item identity.

### Evaluation Data and Gate Metrics
- Synthetic Validation: Recall@1: 0.400, Recall@5: 0.400, MRR: 0.400 (SINTETIS stand in).
- Real Dataset Gate: TERTUNDA (pending `simpanugm-eval` v1 freeze).

---

## 2. Cross-Modal Text Encoder (`xmodal_text_encoder`)

| Attribute | Specification |
|---|---|
| Model ID | `sentence-transformers/clip-ViT-B-32-multilingual-v1` |
| Component Kind | Multilingual cross-modal text encoder |
| Revision | `58edf8cada9e398793dca955574a48cbb7f18be2` |
| Output Dimension | 512 (float32, L2 normalized) |
| License | Apache-2.0 |
| Source | Hugging Face Hub / Sentence-Transformers (multilingual distillation of CLIP text tower) |
| Input Format | Raw text string, tokenized with multilingual tokenizer, max 128 tokens |
| Measured Latency | p50: 24 ms; p95: 28 ms (1 text) |
| Hardware Profile | Reference laptop (Windows 11, Intel 12 logical CPUs, 4 PyTorch threads) |

### Intended Use
Encode textual descriptions of lost reports directly into CLIP visual embedding space when
the lost report does not have an attached photo (cross-modal matching T2).

### Limitations and Failure Modes
- Performance drops on highly local Indonesian campus slang unless normalized via the synonym table.
- Abstract or non-visual text (e.g. emotional remarks, generic statements) projects poorly into visual space.
- Shorter descriptions (under 5 words) lack distinctive features.

### Evaluation Data and Gate Metrics
- Synthetic Validation: Recall@10 without lost photo: 0.571 (SINTETIS stand in).
- Real Dataset Gate: TERTUNDA (pending `simpanugm-eval` v1 freeze).

---

## 3. Semantic Text Encoder (`text_encoder`)

| Attribute | Specification |
|---|---|
| Model ID | `intfloat/multilingual-e5-small` |
| Component Kind | Dense semantic sentence encoder |
| Revision | `614241f622f53c4eeff9890bdc4f31cfecc418b3` |
| Output Dimension | 384 (float32, L2 normalized) |
| License | MIT |
| Source | Hugging Face Hub / Microsoft |
| Input Format | Structured text with prefix `query: ` or `passage: `, max 512 tokens |
| Measured Latency | p50: 19 ms; p95: 23 ms (1 text) |
| Hardware Profile | Reference laptop (Windows 11, Intel 12 logical CPUs, 4 PyTorch threads) |

### Intended Use
Compute dense semantic representations of report titles, descriptions, categories, colors,
and brands for text similarity matching and semantic search (AI-01, AI-13).

### Limitations and Failure Modes
- Requires asymmetric prefixing: `query: ` for queries/lost descriptions and `passage: ` for found item passages.
- Sensitive to sensitive number leakage: PII (credit cards, KTM IDs) must be masked before embedding.
- Context window truncated at 512 tokens (descriptions are bounded at 2000 characters).

### Evaluation Data and Gate Metrics
- Synthetic Validation: T8 Semantic Recall@5: SINTETIS stand in.
- Real Dataset Gate: TERTUNDA (pending `simpanugm-eval` v1 freeze).

---

## 4. Optical Character Recognition (`ocr`)

| Attribute | Specification |
|---|---|
| Model ID | `JaidedAI/EasyOCR` |
| Component Kind | Text detection and recognition pipeline |
| Revision | `craft_mlt_25k+latin_g2` |
| Checksums | `craft_mlt_25k.pth` (SHA-256: `4a5efbfb48b4081100544e75e1e2b57f8de3d84f213004b14b85fd4b3748db17`), `latin_g2.pth` (SHA-256: `aaa95be1c4a9cb3496879bed7c520886ce1164f89e026f0c54488394e74e8c55`) |
| Output Format | Bounding boxes, recognized string tokens, confidence scores |
| License | Apache-2.0 |
| Source | JaidedAI EasyOCR |
| Input Format | Image scaled to short side 1024 px |
| Measured Latency | p50: 737 ms; p95: 747 ms (mock identity card) |
| Hardware Profile | Reference laptop (Windows 11, Intel 12 logical CPUs, 4 PyTorch threads) |

### Intended Use
Detect printed and card text on items (e.g. KTM, name tags, student cards) to:
1. Extract candidate text for PII masking (AI-04).
2. Compute `name_hit` matching signal when the owner's name matches text detected on the item.

### Limitations and Failure Modes
- Poor performance on cursive handwriting or stylized artistic fonts.
- Degrades on reflective surfaces, glossy cards with glare, and low-light or angled photos.
- Non-Latin scripts are not supported in the default language pack.

### Evaluation Data and Gate Metrics
- Synthetic Validation: Masking regex and test card verification tested in `ml/tests/test_ocr.py`.
- Real Dataset Gate: TERTUNDA (pending `simpanugm-eval` v1 freeze).

---

## 5. Multi-Modal Fusion (`fusion`)

| Attribute | Specification |
|---|---|
| Model ID | `simpanugm-fusion` |
| Component Kind | Multi-modal score aggregation and tier calibration |
| Active Variant | B3 (Calibrated linear weighted mean, configuration `fusion-linear-2026.11.0.json`) |
| Experimental Variant | B4 (Learned logistic regression, `ml/eval/learned_fusion.py`) |
| Input Format | 13 normalized component similarity scores and missingness indicators |
| Output Format | Continuous score [0, 1] and calibrated tier (`high`, `medium`, `low`, `not_stored`) |
| Latency | < 1 ms per pair (vectorized NumPy / pure Python) |

### Intended Use
Aggregate visual, semantic, attribute (colour/brand), location, time, and category signals into
a unified candidate match score and assign operational confidence tiers.

### Current Status
- Active in production: Calibrated linear weighted mean (B3).
- Experimental (B4): Logistic regression trained via grouped cross-validation (`ml/eval/learned_fusion.py`).
- Synthetic Validation: B4 achieves Recall@5 1.0000 vs B3 0.4000, MRR 0.9500 vs B3 0.4000, high tier precision 1.0000 (SINTETIS).
- Real Dataset Gate: TERTUNDA. Per `matching-engine.md` section 7.2, learned fusion will only be activated if verified on the frozen real campus dataset.

