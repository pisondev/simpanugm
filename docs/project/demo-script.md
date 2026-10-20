# SimpanUGM live demo script

Twelve-minute presentation kit, updated 20 October 2026. Operational instructions are
in English; quoted speaker notes are Indonesian. The demonstration runs the real API,
worker and local models. Anyone can explore the application outside the script.

## Preparation

From the repository root, build the current source and reset the demo story:

```powershell
make dev
make demo
make demo-check
```

On PowerShell, if make cannot find `sh`, first add Git Bash to this process's PATH:

```powershell
$env:PATH = 'C:\Program Files\Git\bin;C:\Program Files\Git\usr\bin;' + $env:PATH
```

Open `http://localhost:8000`. Mailpit is at `http://localhost:8025`. The web container
must be built with `NEXT_PUBLIC_DEMO_MODE=true`; do not change its default in source.
Finish builds before interacting with the models. Verify model readiness, accounts,
local photos and the actual process memory through `make demo-check`. Rehearse the
photo step once to warm inference, then clear the rehearsal draft through the UI.

Demo data and scores are labelled as such. The landing footer explains this is a
student project, not an official UGM service. Do not present demo outputs as an official
accuracy evaluation. No hosting or external network is required for the reference flows.
The local pack has 168 attributed images. The seed-only account `Petugas Foto Demo`
supports background report distribution and is not a primary presentation actor.
Before a demo, `/bantuan` lists current post locations and opening hours; `/privasi` and
`/ketentuan` keep the full student-project disclosures available from the public footer.
At `/masuk`, use a demo role button for a seeded actor. Registration and expired-link
recovery remain available for the full account lifecycle demonstration.
For password recovery, use a fresh test account so the seeded demo credentials remain
unchanged. Invitation acceptance is covered through the real Mailpit link flow.

### Accounts

Demo-only password: `kata sandi panjang 2026`. One-click demo actor buttons are also
available on `/masuk`. These accounts are refused by production seeding.

| Actor | Name | Email | Focus |
|---|---|---|---|
| Finder | Budi Santoso | anggota.penemu@demo.simpanugm.my.id | Photo assistance, draft and finder hint |
| Owner | Siti Rahma | anggota.pemilik@demo.simpanugm.my.id | Candidates, explanations and claim |
| Library officer | Agus Prasetyo | petugas.pos1@demo.simpanugm.my.id | Queue, board, verification and handover |
| Science officer | Dewi Lestari | petugas.pos2@demo.simpanugm.my.id | Secondary post and custody |
| Admin | Admin SimpanUGM | admin@demo.simpanugm.my.id | Monitoring, clusters, maps, trends and fusion |

## Schedule

| Time | Scene | Suggested presenter |
|---|---|---|
| 00:00 to 01:00 | Public introduction and privacy | Pison |
| 01:00 to 03:30 | Finder and photo assistance | Pison or the team |
| 03:30 to 06:00 | Owner evidence and claim | Axelle or the team |
| 06:00 to 08:30 | Officer review and handover | Irfan or the team |
| 08:30 to 11:00 | Admin AI operations | Pison or the team |
| 11:00 to 12:00 | Open exploration and questions | Team |

## Scene 0, public introduction

Show the blue demo banner, the "Yang hilang, bisa kembali." hero, report entry,
three-step explanation and privacy note. Registration and sign-in stay available.

> SimpanUGM membantu mempertemukan laporan kehilangan dan temuan di lingkungan UGM.
> Sistem ini berjalan di laptop dengan model nyata. Skor membantu memilih kandidat,
> sedangkan kepemilikan tetap diperiksa petugas.

## Scene 1, finder

1. Sign in as Budi, open `/lapor/temuan`, and choose a local photo from the demo pack
   or a permitted team-owned object. The photo is required for a found report.
2. Show "Mulai dari satu foto" and the real "Menyiapkan saran dari foto..." status.
   Continue to Detail while analysis is pending. Inputs remain editable.
3. When "Saran dari fotomu sudah siap" appears, inspect the fields labelled
   "Saran otomatis", the two alternative category chips and drafted description.
   Do not promise a particular category, brand or color. The earbuds photo was
   classified as Charger in phase 1, so demonstrate manual correction to Earphone.
   In the D01 holdout run, the flash drive photo ranked Earphone at 0.8996 and
   Flashdisk at 0.0327. The report was corrected manually, and the real worker found
   no candidate. Both are single demo examples, not model accuracy results.
4. If analysis times out, show the inline unavailable explanation, try "Coba saran
   lagi" once, then continue manually if needed. Do not replace real output with a mock.
5. Enter the item name and correct public description, select the found location and
   time, choose the nearest appropriate post, review custody and duplicate suggestions,
   and submit. Keep hidden details out of the public description.
6. On `/laporan/temuan/FND-...`, show post instructions and the finder hint if a real
   candidate exists. A hint is not guaranteed for every uploaded object.

> Foto membantu menyiapkan detail agar pelaporan lebih cepat. Ini tetap saran yang
> bisa keliru. Saya periksa dan koreksi sebelum mengirim. Saat layanan lambat, draf
> dan foto tetap ada, dan saya bisa melanjutkan sendiri.

For identity examples, use only mock cards marked CONTOH. Masking and blurred public
variants must remain intact; never upload a real student's identification document.

## Scene 2, owner

1. Sign in as Siti and open `/laporan`. Choose a seeded lost report with candidates.
   For a cross-modal example, select a lost report without an owner photo.
2. Open the candidate list, show its tier, actual overall percentage, location and
   storage post, then open the candidate detail.
3. Show the entire found-item photo, four labelled evidence meters and their reasons.
   Explain missing levels honestly. Member bars summarize bands; they are not exact
   component probabilities or proof of ownership.
4. Point out the primary "Ini milik saya" link, secondary "Sepertinya milik saya"
   feedback button, and "Bukan milik saya" dismissal action. Do not dismiss the
   presentation's hero candidate during a rehearsal that must be reused.
5. Follow "Ini milik saya" to `/klaim/baru?pasangan=...`, answer the real category
   questions, add allowed evidence, submit and show the claim's next action.

> Kandidat ini muncul karena beberapa cirinya mirip. Persentasenya bukan kepastian
> bahwa barang ini milik saya. Alasan tiap sinyal bisa diperiksa, lalu saya membuktikan
> kepemilikan melalui pertanyaan kategori dan verifikasi petugas.

## Scene 3, officer

`make demo` prepares these states, and `make demo-check` verifies all three through
authenticated API requests. No rehearsal or manual approval is needed:

| Story | Owner account | Officer account | Steps |
|---|---|---|---|
| Approved umbrella at library | Siti, `anggota.pemilik@demo.simpanugm.my.id` | Agus, `petugas.pos1@demo.simpanugm.my.id` | Siti opens `/klaim`, selects the approved umbrella and shows its current six-digit code. Agus opens that claim in `/petugas/klaim`, checks the mock identity, enters only the last four characters `3107`, confirms the name and submits the code. |
| Keys still with finder | Anggota Demo, `anggota@demo.simpanugm.my.id` | Agus | Open the key report from `/laporan` as its finder and copy its FND code. Agus enters that code at `/petugas/terima`, selects the library post and storage slot, then confirms receipt. The finder chose the library despite finding it near BioGeo. |
| Calculator needs information | Siti | Dewi, `petugas.pos2@demo.simpanugm.my.id` | Siti opens the calculator claim, reads the officer's question and answers with the sticker and handwritten-name details. The claim returns to review. |

The keys claim remains under review for checklist demonstrations after receipt. Completed jacket
and rejected tumbler claims show the terminal states. Codes are generated by the real
claim service on each reset, never hardcoded. Reset again after consuming these stories.

1. Sign in as Agus. `/petugas` contains "Antrean Prioritas Kerja" and post-scoped
   counts. Show how priority helps choose a task. Do not claim this is a separate
   `/petugas/antrean` route.
2. Open a real report pair through its "Papan tinjauan" action to `/petugas/papan`.
   Compare both reports, private evidence available to the officer and actual signal
   reasons. Keep pin, rematch, invite and dismissal controls visible.
3. Show `/petugas/duplikat` and semantic search on `/petugas/cari`. Query a natural
   phrase; describe actual results rather than promising a particular rank.
4. Open the seeded claim from `/petugas/klaim`, inspect the verification checklist,
   request more information or approve only after checking the required evidence.
5. Use the handover section inside that claim detail. Check the person against the
   allowed identification procedure and enter the owner's six-digit handover code.
   Do not show or record real identity numbers. Demonstrate locked-code state only
   on a disposable demo claim, not the hero claim needed later.

> Antrean membantu menentukan prioritas. Petugas tetap membandingkan bukti dan
> memeriksa pemiliknya. Serah terima baru selesai setelah verifikasi dan kode cocok.

## Scene 4, admin

1. Sign in as Admin. Open `/admin/ai/monitoring`, inspect current model/version,
   assist outcomes, score distributions, dismiss rates and drift indicators.
2. Show existing fusion activation and rollback controls on that page. Confirm a
   change only if the presentation calls for it; capture current state first so it
   can be restored. Do not invent a separate model-management route.
3. Open `/admin/ai/klaster`, headed "Pengelompokan Kasus & Deteksi Duplikat". Inspect
   actual HDBSCAN groups or explain a real empty state. Show report links.
4. Show `/admin/peta-rawan` and `/admin/tren`, their real filters, privacy suppression
   and table/export alternatives. Do not infer exact incident locations from a
   privacy-protected aggregate.

> Tampilan ini membantu memantau operasi dan perubahan hasil model. Angka yang
> terlihat berasal dari data demo. Hasil ini belum menjadi evaluasi akurasi resmi.

## Close and free exploration

> Alur dapat dicoba di luar skenario ini. Silakan gunakan akun demo, atau daftar
> dengan email kampus dan buka verifikasi melalui kotak masuk lokal. Kami ingin
> melihat apakah pengguna dapat melapor, menilai kandidat dan menyelesaikan klaim
> dengan jelas, termasuk ketika saran model keliru.

## Recording and reference captures

From `web/`, with the demo stack running:

```powershell
$env:DEMO_E2E = '1'
npx playwright test demo-polish --project=desktop --output=test-results/phase1-playwright
npx playwright test demo-record --project=desktop --output=test-results/demo-videos
```

`make demo-test` and `make demo-record` remain available from the repository root.
The explicit commands above make artifact paths predictable:

- Reference desktop/phone PNGs: `web/test-results/screenshots/phase1-*.png`
- Recording scene PNGs: `web/test-results/demo-record/scene*.png`
- Recording videos: `web/test-results/demo-videos/<test-folder>/video.webm`

All are gitignored. Open the actual PNGs/videos before using them as a fallback. The
recording suite captures UI journeys; the recording alone does not establish that every
scene submitted a claim or completed a handover. Perform those actions in rehearsal.

Run one Playwright worker, keep the shared clock running, and do not mix time-travel
tests with the demo. Restore demo configuration and rerun `make demo` after destructive
rehearsal or claim tests. Check [demo-polish-playbook.md](demo-polish-playbook.md) for
phase 2 work and [phase1-validation.md](../ux/phase1-validation.md) for measured results.
