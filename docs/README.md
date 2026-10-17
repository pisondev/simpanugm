# SimpanUGM documentation

Internal specification for the team. Everything here is the single source of truth: if
code and a page disagree, one of them is a bug and the pull request that fixes it updates
both.

## Map

| Folder | Page | Answers |
|---|---|---|
| `product/` | [prd.md](product/prd.md) | Why the product exists, who it serves, scope per release, success metrics |
| | [user-stories.md](product/user-stories.md) | Every story with acceptance criteria, priority, estimate, sprint |
| `requirements/` | [srs.md](requirements/srs.md) | Every functional and non functional requirement, with ids |
| | [business-rules.md](requirements/business-rules.md) | Rules the system enforces: visibility, limits, deadlines, retention |
| | [state-machines.md](requirements/state-machines.md) | Lifecycle of lost reports, found reports, custody, matches, claims |
| | [permissions.md](requirements/permissions.md) | Role and permission matrix, ownership scopes |
| | [notifications.md](requirements/notifications.md) | Every event, recipient, channel and message |
| | [traceability.md](requirements/traceability.md) | Concept document item to requirement to story to module to test |
| `architecture/` | [overview.md](architecture/overview.md) | Context, containers, components, key runtime flows, deployment view |
| | [data-model.md](architecture/data-model.md) | Entity relationship diagram and every table |
| | [api.md](architecture/api.md) | REST conventions and the endpoint catalog |
| | [security-privacy.md](architecture/security-privacy.md) | Threat model, controls, personal data handling, retention |
| | [adr/](architecture/adr/) | Architecture decision records |
| `ai/` | [matching-engine.md](ai/matching-engine.md) | Models, features, retrieval, score fusion, tiers, explanations |
| | [ai-features.md](ai/ai-features.md) | Every AI feature per role, inputs, outputs, fallbacks |
| | [evaluation.md](ai/evaluation.md) | Offline protocol, metrics, acceptance gates, online metrics |
| | [dataset-collection.md](ai/dataset-collection.md) | How the evaluation dataset is built, labelled and versioned |
| | [model-lifecycle.md](ai/model-lifecycle.md) | Versioning, reproducibility, activation, monitoring, rollback |
| `ux/` | [information-architecture.md](ux/information-architecture.md) | Sitemap per role, screen inventory, key flows |
| | [ui-guidelines.md](ux/ui-guidelines.md) | Design tokens, components, copy rules, accessibility |
| `engineering/` | [tech-stack.md](engineering/tech-stack.md) | Every runtime, library and tool, pinned, with the reason |
| | [repository-structure.md](engineering/repository-structure.md) | Planned layout inside each service folder |
| | [coding-standards.md](engineering/coding-standards.md) | Go, TypeScript, Python and SQL conventions |
| | [testing-strategy.md](engineering/testing-strategy.md) | Test levels, tools, coverage targets, what CI runs |
| | [local-development.md](engineering/local-development.md) | Prerequisites, services, ports, environment variables |
| `operations/` | [deployment.md](operations/deployment.md) | Environments, CI/CD, release process, server layout |
| | [observability.md](operations/observability.md) | Logs, metrics, health checks, SLOs, alerts |
| | [runbook.md](operations/runbook.md) | Incident procedures, backup and restore, model rollback |
| `project/` | [project-plan.md](project/project-plan.md) | Timeline, sprints, milestones, work breakdown |
| | [team.md](project/team.md) | Roles, RACI, rituals, communication |
| | [risks.md](project/risks.md) | Risk register with owners and mitigations |
| | [definition-of-done.md](project/definition-of-done.md) | Definition of ready and definition of done |
| | [glossary.md](glossary.md) | Every domain term, English and Indonesian |

## Reading order by role

| You are | Read first |
|---|---|
| Everyone | prd, glossary, project-plan, CONTRIBUTING |
| Building `api/` | srs, state-machines, permissions, data-model, api, security-privacy, adr |
| Building `web/` | user-stories, information-architecture, ui-guidelines, api, permissions |
| Building `ml/` | matching-engine, ai-features, evaluation, model-lifecycle |
| Building `data/` | dataset-collection, evaluation, data-model, business-rules (retention) |

## Id scheme

| Prefix | Meaning | Defined in |
|---|---|---|
| `FR-<MODULE>-nn` | Functional requirement | srs |
| `NFR-<QUALITY>-nn` | Non functional requirement | srs |
| `BR-<AREA>-nn` | Business rule | business-rules |
| `US-nn` | User story | user-stories |
| `AI-nn` | AI feature | ai-features |
| `ADR-nnnn` | Architecture decision | adr |
| `R-nn` | Risk | risks |

## Conventions for these pages

- English. UI copy quoted inside a page is Indonesian, exactly as it will appear.
- Diagrams are Mermaid so they render on GitHub and diff as text.
- Numbers that the system enforces (limits, thresholds, deadlines) appear once in
  business-rules or matching-engine and are referenced elsewhere by id, so they change in
  one place.
