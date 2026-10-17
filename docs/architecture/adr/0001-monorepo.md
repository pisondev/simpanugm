# ADR-0001 One monorepo with a folder per area

Status: Accepted, 2026-10-17

## Context

Three services in three languages (`web` TypeScript, `api` Go, `ml` Python) plus data
pipelines and deployment files. Three people. The services share contracts: the public
OpenAPI spec, the inference API, the database schema and the seed files. A change to a
contract usually touches two services at once.

## Decision

One repository, `pisondev/simpanugm`, with top level folders `web/`, `api/`, `ml/`,
`data/`, `deploy/`, `docs/`. Each folder has an owner (CONTRIBUTING section 1) and its own
toolchain files. CI runs per folder with path filters.

## Consequences

- A contract change and both of its consumers land in one pull request and one review.
- One issue tracker, one board, one changelog, one version number for the product.
- CI must be path filtered so a docs change does not run the Python suite.
- Toolchains coexist at the root: `.editorconfig` and `.gitattributes` cover all three
  languages.

## Alternatives

- **A repository per service.** Clean ownership, but every contract change becomes three
  coordinated pull requests, and the version of "the system" is no longer a single commit.
  Too much ceremony for three people.
- **A workspace tool such as Nx or Turborepo.** Useful at a larger scale. With one
  JavaScript package the benefit does not cover the learning cost.
