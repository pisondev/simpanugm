# Team | SimpanUGM

Grup A, PACS262527 Rekayasa Model Kecerdasan Artifisial, Semester Gasal 2026/2027.
Lecturers: Dr. Muhammad Ardi Putra, S.Kom. and Prof. Dr. Drs. Azhari, MT.

## 1. Roles

| Member | Role | Owns | Also does |
|---|---|---|---|
| Pison Golda | Software Engineer, tech lead | `api/`, `web/`, `deploy/`, CI, root files, architecture, release | final call on cross area technical disputes, with the reason written down |
| Irfan Fadilah Rafif | AI Engineer | `ml/`, matching engine design, evaluation harness, model lifecycle, AI-12 review | pairs on the Go scoring, presents the AI engineering in the demo |
| Axelle Chandra | Data Engineer | `data/`, evaluation dataset, seeds, synonyms and brands, analytics views, schema review | admin and analytics pages in `web` in Sprint 4, retention rules review |

## 2. Responsibility matrix

R does the work, A answers for it, C is consulted, I is informed.

| Activity | Pison | Irfan | Axelle |
|---|---|---|---|
| Product and requirements docs | A, R | C | C |
| Architecture and ADRs | A, R | C | C |
| Database schema and migrations | A, R | I | C |
| `openapi.yaml` contract | A, R | I | C |
| `api` modules | A, R | C for matching | C for analytics |
| `web` member and officer flows | A, R | I | C |
| `web` admin and analytics pages | A | I | R |
| Inference service | C | A, R | I |
| Matching engine and fusion | C, R for Go | A, R | C |
| Evaluation dataset | I | C | A, R |
| Evaluation runs and report | I | A, R | C |
| Seeds and master data | C | C | A, R |
| CI, deployment, observability | A, R | C for `ml` | I |
| Backups and restore drill | A, R | I | C |
| Security checklist | A, R | C | C |
| Usability tests | A | R | R |
| Demonstration | A, R | R | R |

## 3. Rituals

| Ritual | When | Length | Output |
|---|---|---|---|
| Sprint planning | first Saturday of the sprint, 09:00 WIB, online | 60 min | sprint goal, issues moved to Ready with an assignee |
| Daily check in | by 21:00 WIB, written in the team chat | 2 min | three lines: kemarin, hari ini, hambatan |
| Mid sprint sync | Wednesday 19:30 WIB, online | 30 min | blockers cleared, scope adjusted |
| Review and retrospective | last Friday of the sprint, 19:30 WIB | 60 min | demo of merged work on staging, one thing to keep and one to change, recorded in `docs/project/retros.md` |
| Weekly AI review | Monday, async | 15 min | Irfan posts AI-12 observations from Sprint 3 onwards |

## 4. Communication

| Channel | For |
|---|---|
| Team chat group | daily check ins, quick questions, alerts |
| GitHub issues and the project board | every unit of work, its discussion and its status |
| Pull requests | code and doc review |
| This `docs/` folder | every decision that outlives a conversation |
| Team Drive folder | evaluation dataset, consent records, lecturer documents |

A decision made in chat that changes the specification is not real until a pull request
updates the page.

## 5. On call

From Sprint 2 for staging and from the production deployment in Sprint 4, one member per
week is on call: watches alerts during waking hours, follows the runbook, writes a short
note for anything that paged. Rotation: Pison, Irfan, Axelle, starting the week of
9 November.

## 6. Working agreements

1. Work happens on issues. No issue, no branch.
2. A pull request is reviewed within 24 hours. Review comments are answered within 24
   hours.
3. Nobody merges their own pull request without an approval.
4. Blocked for more than half a day means saying so in the chat.
5. Estimates are the team's, not one person's. Re estimate in the mid sprint sync if
   reality differs.
6. Exams and other deadlines are announced at sprint planning so capacity is planned, not
   discovered.
