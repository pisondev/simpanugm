# SimpanUGM Live Demo Script and Actor Accounts

This document outlines the demonstration story, actor roles, demo credentials, and walkthrough steps for the SimpanUGM system evaluation.

## 1. Quick Start and Reset

To prepare a clean, fully populated demonstration environment:

```bash
make demo
```

This target runs migrations, seeds master data, loads the realistic demo scenarios, computes real neural embeddings, scores matches, populates claims across lifecycle stages, and refreshes all analytics materialized views.

## 2. Demo Accounts and Passwords

All demo accounts share the standard password below.

Standard Password: `kata sandi panjang 2026`

| Role | Actor | Name | Email | Post Assignment | Purpose |
|---|---|---|---|---|---|
| Member | Finder | Budi Santoso | anggota.penemu@demo.simpanugm.my.id | None | Files found reports, receives finder hints |
| Member | Owner | Siti Rahma | anggota.pemilik@demo.simpanugm.my.id | None | Files lost reports, reviews match candidates, submits claims |
| Member | General | Anggota Demo | anggota@demo.simpanugm.my.id | None | Secondary member account for exploring listing |
| Officer | Pos 1 | Agus Prasetyo | petugas.pos1@demo.simpanugm.my.id | Pos Klaster Perpustakaan | Manages intake, custody, and verification at Library Post |
| Officer | Pos 2 | Dewi Lestari | petugas.pos2@demo.simpanugm.my.id | Pos Klaster Saintek | Manages intake and claims at Science Post |
| Officer | General | Petugas Demo | petugas@demo.simpanugm.my.id | All Posts | Cross post oversight, review board, duplicate merges |
| Admin | Administrator | Admin Demo | admin@demo.simpanugm.my.id | All Posts | AI monitoring, fusion activation, spatial analytics |

## 3. Demo Flow by Actor

### Scene 1: Finder (Budi Santoso)
- **Goal**: Demonstrate AI assisted found report submission and finder hint.
- **Login**: `anggota.penemu@demo.simpanugm.my.id`
- **Action**:
  1. Navigate to `/lapor/ditemukan`.
  2. Upload photo of found earbuds.
  3. Notice Vision AI assistant auto populating Category ("Elektronik & Gadget / Earphone & Headphone"), Colours, and Brand.
  4. Notice OCR masking of any sensitive serial numbers or private text.
  5. Submit report. Finder hint banner alerts that a student recently reported matching lost earbuds.

### Scene 2: Owner (Siti Rahma)
- **Goal**: Demonstrate candidate matching tiers, score breakdown, and category specific verification claim.
- **Login**: `anggota.pemilik@demo.simpanugm.my.id`
- **Action**:
  1. Open active lost report for brown leather wallet.
  2. Review Match Candidate list sorted by multi modal score.
  3. Inspect match explanation: High match tier, percentage breakdown (Visual, Text, Location, Time).
  4. Click claim button on candidate card.
  5. Answer category specific verification questions (e.g., hidden card details).

### Scene 3: Officer (Agus Prasetyo / Petugas Demo)
- **Goal**: Demonstrate review board, priority queue, duplicate warnings, and code based handover.
- **Login**: `petugas.pos1@demo.simpanugm.my.id` or `petugas@demo.simpanugm.my.id`
- **Action**:
  1. Navigate to Officer Review Board (`/petugas/papan`).
  2. Check duplicate link suggestions flagged with high text/visual confidence.
  3. Review claim under verification (`/petugas/klaim`).
  4. Tick verification checklist items and approve claim.
  5. Complete physical handover using the 6 digit handover code and recipient identity verification.

### Scene 4: Administrator (Admin Demo)
- **Goal**: Demonstrate AI monitoring, HDBSCAN cluster maps, and zero downtime fusion model lifecycle.
- **Login**: `admin@demo.simpanugm.my.id`
- **Action**:
  1. Navigate to AI Monitoring (`/admin/ai/monitoring`).
  2. Review AI assistant acceptance rate metrics and match outcome distributions.
  3. View HDBSCAN spatial clusters on the campus map (`/admin/analitik`).
  4. Navigate to Model Versions (`/admin/model-versions`).
  5. Demonstrate active fusion model version rollback and activation without container deployment.
