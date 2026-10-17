# ml

The inference service and the offline evaluation harness. Stateless FastAPI service on the
internal network: image and text embeddings, image analysis (category, colour, visible
text, brand), clustering. Scoring and every decision stay in `api`.

Owner: Irfan Fadilah Rafif. Status: specified, implementation starts in Sprint 1.

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
