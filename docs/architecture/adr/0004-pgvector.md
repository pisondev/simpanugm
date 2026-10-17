# ADR-0004 Embeddings in PostgreSQL with pgvector

Status: Accepted, 2026-10-17

## Context

Each report has one or more 512 dimension image vectors and two text vectors. Retrieval
must combine vector similarity with relational filters: report status, type, time window,
hidden flag and model version. Expected volume is at most a few hundred thousand vectors.

## Decision

Store vectors in PostgreSQL with the pgvector extension, version 0.8 or newer, using HNSW
indexes with cosine distance and iterative index scans so filters apply during the search.

## Consequences

- Vectors commit in the same transaction as the rows they describe, and deleting a photo
  deletes its vector by cascade. Retention is one rule, not two.
- One backup covers everything.
- Filtered nearest neighbour search is one SQL query, tested like any other.
- At our scale HNSW search costs milliseconds. If volume grew by orders of magnitude,
  partitioning by report type and year would come before a separate store.

## Alternatives

- **A dedicated vector database such as Qdrant or Milvus.** Another stateful service to
  run and back up, and a consistency gap between reports and their vectors.
- **Brute force cosine in Go.** Viable for the demo volume, but it moves filtering and
  ranking into application memory and does not hold at NFR-PERF-06 sizes.
