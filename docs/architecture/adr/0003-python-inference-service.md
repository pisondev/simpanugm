# ADR-0003 Python inference service as the only non Go backend

Status: Accepted, 2026-10-17

## Context

The agreed stack is Next.js, PostgreSQL and Go. The matching engine needs image and text
encoders, OCR and clustering. The mature implementations of CLIP, multilingual sentence
encoders, EasyOCR and HDBSCAN are Python packages built on PyTorch and scikit-learn. Go
bindings exist for ONNX Runtime, but OCR pipelines, tokenisers and preprocessing would have
to be reimplemented and kept bit compatible with the Python reference, which is where
evaluation happens.

## Decision

A separate service `ml/` in Python 3.12 with FastAPI, internal only, stateless, holding no
database connection and no storage access. It receives bytes and text, returns vectors and
attributes, and names the model revision in every response. All state, all scoring and all
decisions stay in Go.

## Consequences

- Python appears exactly where the ecosystem demands it, and the stack constraint holds
  for everything a user touches.
- The evaluation harness in `ml/eval/` uses the same code that serves production, so
  offline numbers describe the deployed behaviour.
- One more image to build and operate. Its interface is small, six endpoints, and its
  failure is tolerated by design (NFR-AVAIL-02).
- The inference API is a contract with its own OpenAPI file and tests on both sides.

## Alternatives

- **ONNX models called from Go.** Possible for the two CLIP encoders, and kept as a v1.1
  speed option inside `ml`. Not possible for OCR without a large rewrite.
- **A hosted AI API.** Violates CON-05: report text and photos would leave our server,
  latency and cost depend on a third party, and results are not reproducible by revision.
