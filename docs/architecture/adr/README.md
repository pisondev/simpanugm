# Architecture Decision Records

One file per decision that is expensive to reverse. Format: context, decision,
consequences, alternatives. A superseded record stays and links to its successor.

| Id | Decision | Status | Date |
|---|---|---|---|
| [ADR-0001](0001-monorepo.md) | One monorepo with a folder per area | Accepted | 2026-10-17 |
| [ADR-0002](0002-go-modular-monolith.md) | Go modular monolith with a separate worker process | Accepted | 2026-10-17 |
| [ADR-0003](0003-python-inference-service.md) | Python inference service as the only non Go backend | Accepted | 2026-10-17 |
| [ADR-0004](0004-pgvector.md) | Embeddings in PostgreSQL with pgvector | Accepted | 2026-10-17 |
| [ADR-0005](0005-river-job-queue.md) | PostgreSQL backed job queue with River | Accepted | 2026-10-17 |
| [ADR-0006](0006-same-origin-cookie-sessions.md) | Same origin behind Caddy with server side cookie sessions | Accepted | 2026-10-17 |
| [ADR-0007](0007-media-storage.md) | Storage interface, filesystem by default, S3 compatible as an option | Accepted | 2026-10-17 |
| [ADR-0008](0008-explainable-fusion.md) | Explainable weighted fusion first, learned fusion only through the evaluation gate | Accepted | 2026-10-17 |
| [ADR-0009](0009-contract-first-api.md) | Contract first REST API with OpenAPI and generated code | Accepted | 2026-10-17 |
| [ADR-0010](0010-multilingual-clip.md) | Multilingual CLIP for text to image matching | Accepted | 2026-10-17 |

New record: copy the newest file, take the next number, open a pull request labelled
`adr`, get approval from every member.
