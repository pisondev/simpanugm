# ml

The inference service and the offline evaluation harness. Stateless FastAPI service on the
internal network: image and text embeddings, image analysis (category, colour, visible
text, brand), clustering. Scoring and every decision stay in `api`.

Owner: Irfan Fadilah Rafif. Status: the three inference endpoints, the model lock with real
revisions, the report assistant analysis, the Python fusion reference, and the evaluation
harness with calibration are in place. The harness runs end to end on a synthetic dataset
in tests, the real run waits for the frozen `simpanugm-eval` dataset.

| Read first | For |
|---|---|
| [docs/ai/matching-engine.md](../docs/ai/matching-engine.md) | models, the inference API, the scoring this service feeds |
| [docs/ai/ai-features.md](../docs/ai/ai-features.md) | what each feature needs from this service |
| [docs/ai/evaluation.md](../docs/ai/evaluation.md) | the harness in `eval/`, metrics and gates |
| [docs/ai/model-lifecycle.md](../docs/ai/model-lifecycle.md) | pinning, versions, activation, rollback |

Rules that matter most here:

- Model ids and revisions live only in `models.lock.json`. Weights are never committed.
- `eval/fusion.py` must agree with the Go scoring on `data/fixtures/fusion_cases.json`.
- A number in a report comes from `eval/run.py`, never from a notebook.

Stack: Python 3.12, uv, FastAPI, PyTorch CPU, sentence-transformers, EasyOCR,
scikit-learn, MLflow for evaluation runs.

## Endpoints

`openapi.json` is the contract, regenerate it with `PYTHONPATH=. uv run python scripts/export_openapi.py`
after changing an endpoint, a test fails when it drifts.

| Endpoint | Does |
|---|---|
| `GET /v1/models` | what is loaded, with revision and dimension |
| `POST /v1/embed/image` | 1 to 8 JPEG, WebP or PNG files, unit vectors of 512 numbers |
| `POST /v1/embed/text` | up to 32 texts, `e5` (384) and `clip_text` (512), each answer echoes model and revision |
| `POST /v1/analyze/image` | zero shot categories, colours, masked visible text, brand candidates |
| `POST /v1/cluster` | HDBSCAN clustering on normalized vectors, labels and version |

Errors are problem details with the codes `invalid_image`, `too_large`, `too_many_items`,
`invalid_request` and `not_ready`. `/readyz` stays 503 until every component of the lock is
loaded, and a failed load never reports ready.

## Models

`models.lock.json` pins three Hugging Face revisions and, for EasyOCR, the SHA-256 of the two
weight files it downloads from release assets. The first start downloads about 1.5 GB into
`HF_HOME`, so the Compose volume `ml-models` keeps it. Golden tests against the real weights
run with `HF_HOME=... uv run pytest -m models` and are skipped in CI.

Docker builds need `data/seed`, Compose passes it as the build context `seed`:

```
docker build --build-context seed=../data/seed -t simpanugm-ml:local .
```

## Evaluation

```
make eval EVAL_SPLIT=val                  # B0 to B3 and ablations A1 to A7, components T4 to T8, gates
uv run python -m eval.run --dataset ... --calibrate fusion-linear-2026.11.1   # val only
make eval-latency                         # CPU latency per component
```

A run refuses a dirty tree, seeds everything with 2026, and writes `ml/eval/runs/<id>/`
(`metrics.json`, `per_query.csv`, `slices.csv`, `config.yaml`, `environment.json`). Runs are
not committed. Calibration reads `val` only and writes a new document under `eval/configs/`.

## Working on it

Requires Python 3.12 and uv.

```
uv sync                       # creates .venv from uv.lock
uv run pytest
uv run ruff check . && uv run ruff format --check .
uv run mypy                   # strict on app/
ML_INTERNAL_TOKEN=change-me-to-16-chars uv run uvicorn app.main:create_app --factory --port 8100
```

`/healthz` and `/readyz` are open on the internal network. Every `/v1` endpoint needs
`Authorization: Bearer $ML_INTERNAL_TOKEN`. `/readyz` turns green only when every component
in `models.lock.json` is loaded, and the lock refuses floating revisions such as `main`.

