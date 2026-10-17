# Roles and Permissions | SimpanUGM

Role based access control with ownership scopes. A role grants a set of permissions. Each
permission is then narrowed by a scope that the service layer checks against the record.

## 1. Roles

| Role | Granted to | Notes |
|---|---|---|
| `member` | Self registration with an allowed email domain | Acts as owner or finder depending on the report |
| `officer` | Invitation by an administrator | Assigned to one or more posts |
| `admin` | Invitation by an administrator | Superset of `officer`, acts at every post |

A user has exactly one role. An officer who loses an umbrella uses the same account to
file a lost report, and the ownership scope keeps the two hats apart.

Two account flags further restrict a role: `unverified` (BR-ACC-02) and
`claim_restricted` (BR-CLM-04).

## 2. Scopes

| Scope | Meaning |
|---|---|
| `own` | The record was created by the user, or the user is its reporter, finder or claimant |
| `matched` | The user owns a lost report that has an active candidate of tier medium or high with this found report, not dismissed by the owner |
| `public` | The public listing projection only, BR-VIS 5.2 first row |
| `post` | The found report is held at, or intended for, a post the officer is assigned to |
| `all` | Any record |

## 3. Matrix

`M` member, `O` officer, `A` admin. A cell names the scope. Empty means denied.

### 3.1 Accounts

| Permission | M | O | A |
|---|---|---|---|
| `account.self.read`, `account.self.update` | own | own | own |
| `account.self.delete` | own | | |
| `account.sessions.manage` | own | own | own, all |
| `user.read` | | | all |
| `user.invite`, `user.update_role`, `user.assign_post`, `user.suspend`, `user.delete` | | | all |
| `user.clear_claim_restriction` | | | all |

### 3.2 Reports

| Permission | M | O | A |
|---|---|---|---|
| `report.lost.create` | own, verified | proxy | proxy |
| `report.lost.read` | own | all | all |
| `report.lost.update` | own, while editable | all, audited | all, audited |
| `report.lost.close` | own | duplicate only | all |
| `report.lost.reopen` | own | | all |
| `report.found.create` | own, verified | proxy | proxy |
| `report.found.read` | own, matched, public | all | all |
| `report.found.update` | own, BR-FND-06 | all until returned | all until returned |
| `report.found.close` | own, BR-FND-05 | duplicate only | all |
| `report.private.read` | own | all, access logged | all, access logged |
| `report.mark_duplicate` | | all | all |
| `report.moderate` | | | all |
| `report.history.read` | own | all | all |

### 3.3 Media

| Permission | M | O | A |
|---|---|---|---|
| `media.upload` | own | own | own |
| `media.read` | per BR-VIS | all variants | all variants |
| `media.delete` | own, while editable | all | all |

### 3.4 Matching and AI

| Permission | M | O | A |
|---|---|---|---|
| `match.read` | own lost report, tier medium or high. Finder sees the hint only | all tiers | all tiers |
| `match.feedback` | own | all | all |
| `match.dismiss` | own, owner scope only | all, global | all, global |
| `match.rematch` | | all | all |
| `assist.analyze` | own, verified | all | all |
| `search.semantic` | | all | all |
| `ai.monitoring.read` | | | all |
| `ai.clusters.read`, `ai.clusters.run` | | | all |
| `ai.models.read`, `ai.models.activate` | | | all |

### 3.5 Claims, custody, handover

| Permission | M | O | A |
|---|---|---|---|
| `claim.create` | own, verified, not restricted | proxy | proxy |
| `claim.read` | own | all | all |
| `claim.withdraw` | own | | all |
| `claim.message` | own | all | all |
| `claim.review` (pick up, request info, reject, reopen) | | all | all |
| `claim.approve`, `claim.revoke` | | post | all |
| `claim.internal_note` | | all | all |
| `custody.receive`, `custody.transfer`, `custody.extend` | | post | all |
| `custody.handover` | | post | all |
| `custody.dispose` | | post | all |
| `custody.inventory.read` | | post, switch to all | all |

### 3.6 Operations and administration

| Permission | M | O | A |
|---|---|---|---|
| `search.reports` | own and public | all | all |
| `officer.queue.read`, `officer.board.read` | | post, switch to all | all |
| `masterdata.read` | all | all | all |
| `masterdata.manage` | | | all |
| `settings.read` | | | all |
| `settings.manage` | | | all |
| `audit.read`, `audit.export` | | | all |
| `analytics.read` | | all | all |
| `analytics.export` | | | all |

## 4. Enforcement

1. **Middleware** resolves the session, loads the user's role, flags and post ids once per
   request, and rejects a request whose route permission the role lacks with
   `403 forbidden`.
2. **Service layer** checks the scope against the record before any read or write. A
   scope failure on a read returns `404 not_found`, not `403`, so the existence of
   another person's record is never confirmed.
3. **Query layer** applies the scope as a `WHERE` clause for every list query, so a
   listing can never include rows outside the scope even if the service check is
   forgotten. Integration tests assert this per role.
4. **Field projection** for found reports and lost reports follows BR-VIS. Each role and
   scope has its own response type in the OpenAPI spec, so an owner response type simply
   has no `hidden_details` field to leak.
5. `GET /api/v1/me` returns the effective permission list so the web client can hide
   actions. The client is never trusted for enforcement.

## 5. Tests that guard this page

- A table driven test walks every cell of section 3 for each role and scope and asserts
  allow or deny.
- For each list endpoint, a test seeds records for three users and asserts each user sees
  exactly their scoped subset.
- Response snapshot tests assert that member responses never contain the field names
  `hidden_details`, `private_details`, `ocr_text`, `finder`, `reporter_contact`.
