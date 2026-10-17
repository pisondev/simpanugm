# ADR-0009 Contract first REST API with OpenAPI and generated code

Status: Accepted, 2026-10-17

## Context

`web` and `api` are written in different languages, often by different people on the same
day. Role specific projections (BR-VIS) mean the same resource has several response
shapes. Drift between client and server is the most likely integration bug.

## Decision

`api/openapi.yaml`, OpenAPI 3.1, is written first and is the contract. Go server
interfaces and request and response types are generated with `oapi-codegen` in strict
server mode. TypeScript types are generated with `openapi-typescript` and called through
`openapi-fetch`. Requests are validated against the spec at runtime with `kin-openapi`
middleware. CI fails when generated code is out of date.

## Consequences

- A field that is not in a role's response type cannot be sent to that role.
- Breaking changes are visible in the diff of one YAML file and reviewed as such.
- Writing the spec by hand costs time in Sprint 1, repaid every time the frontend calls a
  new endpoint.

## Alternatives

- **Code first with annotations generating the spec.** The spec becomes a by product and
  drifts from what reviewers read.
- **GraphQL.** Flexible queries that the product does not need, and authorisation per
  field becomes harder to reason about than per response type.
- **gRPC with a gateway.** Strong typing, but a browser first product gains little and the
  tooling cost is real.
