# Notification Catalog | SimpanUGM

Every event that notifies someone. Channels: **App** is stored in `notifications` and
pushed over Server Sent Events. **Email** is a queued job. Copy is Indonesian and final
wording lives in the templates in `api/internal/notify/templates/`, using the text below
as the baseline.

Mandatory events ignore user preferences. Optional events can be switched off per channel
in Profil, Notifikasi.

## 1. Account and security

All mandatory, email only, never delayed by quiet hours.

| Event | Recipient | Subject | Body baseline |
|---|---|---|---|
| `account.verify_email` | new user | Verifikasi email SimpanUGM | Klik tautan berikut untuk mengaktifkan akunmu. Tautan berlaku 24 jam. |
| `account.password_reset` | user | Atur ulang kata sandi | Ada permintaan atur ulang kata sandi. Tautan berlaku 30 menit. Abaikan jika bukan kamu. |
| `account.password_changed` | user | Kata sandi diubah | Kata sandi akunmu baru saja diubah. Semua sesi lain telah dikeluarkan. |
| `account.invite` | invited officer or admin | Undangan akun SimpanUGM | Kamu diundang sebagai petugas. Atur kata sandi dalam 72 jam. |
| `account.email_change` | new address | Konfirmasi email baru | Konfirmasi alamat ini untuk akun SimpanUGM. |
| `account.suspended` | user | Akun dinonaktifkan | Akunmu dinonaktifkan oleh admin. Hubungi pos keamanan untuk informasi. |

## 2. Matching

| Event | Recipient | App | Email | Optional | Text baseline | Link |
|---|---|---|---|---|---|---|
| `match.high_for_owner` | owner of the lost report | yes | yes | yes | Ada barang temuan yang sangat mirip dengan laporanmu {lost_code}. Lihat kandidatnya. | lost report candidates |
| `match.medium_digest_for_owner` | owner | yes | no | yes | {n} kandidat baru untuk {item_name}. | lost report candidates |
| `match.hint_for_finder` | finder | yes | no | yes | Laporan temuanmu {found_code} mungkin sudah dicari pemiliknya. Pastikan barang diserahkan ke pos. | found report |
| `match.officer_invites_claim` | owner | yes | yes | no | Petugas menilai salah satu kandidat sangat mungkin milikmu. Ajukan klaim untuk melanjutkan. | candidate detail |
| `match.high_for_officer` | officers of the holding or intended post | yes | no | yes | Kandidat kuat: {lost_code} dan {found_code}, skor {score}. | review board |

Throttled by BR-NTF-01. Owner emails respect quiet hours (BR-NTF-03).

## 3. Found items and custody

| Event | Recipient | App | Email | Optional | Text baseline | Link |
|---|---|---|---|---|---|---|
| `found.duplicate_hint` | finder, inline on submit | inline | no | no | Sepertinya barang ini sudah dilaporkan: {found_code} di {location}. | existing report |
| `found.custody_reminder_24h` | finder | yes | yes | no | Barang temuan {item_name} belum diserahkan ke pos. Serahkan ke {post_name} agar pemilik bisa mengambilnya. | found report |
| `found.custody_reminder_48h` | finder | yes | yes | no | Batas waktu penyimpanan oleh penemu sudah habis. Mohon serahkan hari ini ke {post_name}. | found report |
| `found.custody_overdue` | officers of the zone's nearest post | yes | no | no | {found_code} belum diserahkan penemu selama 48 jam. | found report |
| `found.received` | finder | yes | yes | yes | Terima kasih, {item_name} sudah diterima di {post_name}. | found report |
| `found.returned_thanks` | finder | yes | yes | yes | Barang yang kamu temukan sudah kembali ke pemiliknya. Terima kasih sudah membantu. | found report |
| `custody.retention_warning` | post officers | yes | digest | no | {n} barang akan melewati masa simpan dalam 7 hari. | inventory |
| `custody.unclaimed` | post officers | yes | digest | no | {found_code} tidak diklaim hingga batas waktu. Tentukan tindak lanjut. | found report |

## 4. Lost reports

| Event | Recipient | App | Email | Optional | Text baseline | Link |
|---|---|---|---|---|---|---|
| `lost.stale_reminder` | owner | yes | yes | no | Masih mencari {item_name}? Pilih tetap buka atau tutup laporan. | lost report |
| `lost.expired` | owner | yes | yes | no | Laporan {lost_code} ditutup karena tidak ada aktivitas. Kamu bisa membukanya lagi dalam 30 hari. | lost report |
| `lost.confirm_receipt` | owner | yes | no | no | Sudah menerima barangmu? Konfirmasi agar kasus ditutup. | lost report |

## 5. Claims

| Event | Recipient | App | Email | Optional | Text baseline | Link |
|---|---|---|---|---|---|---|
| `claim.submitted_ack` | claimant | yes | no | no | Klaim {claim_code} terkirim. Petugas akan memeriksa. | claim |
| `claim.submitted` | post officers | yes | digest | no | Klaim baru {claim_code} untuk {found_code}. | claim review |
| `claim.info_requested` | claimant | yes | yes | no | Petugas membutuhkan info tambahan untuk klaim {claim_code}. | claim |
| `claim.info_provided` | reviewer | yes | no | no | Penjawab klaim {claim_code} sudah membalas. | claim review |
| `claim.message` | the other party | yes | no | yes | Pesan baru pada klaim {claim_code}. | claim |
| `claim.approved` | claimant | yes | yes | no | Klaim disetujui. Datang ke {post_name} dengan KTM atau KTP. Kode serah terima ada di aplikasi. | claim |
| `claim.on_hold` | claimant | yes | no | no | Klaimmu ditunda karena ada klaim lain yang sedang diproses. | claim |
| `claim.resumed` | claimant | yes | no | no | Klaimmu kembali diperiksa. | claim |
| `claim.rejected` | claimant | yes | yes | no | Klaim {claim_code} ditolak: {reason_label}. | claim |
| `claim.expiring` | claimant | yes | yes | no | Klaim disetujui akan kedaluwarsa besok. Segera ambil barang di {post_name}. | claim |
| `claim.expired` | claimant | yes | yes | no | Klaim {claim_code} kedaluwarsa. | claim |
| `claim.handed_over` | claimant | yes | yes | no | Serah terima selesai. Semoga barangmu aman. | lost report |

The approval email never contains the handover code (FR-CLAIM-12).

## 6. Delivery mechanics

| Aspect | Rule |
|---|---|
| Creation | The domain transaction inserts the `notifications` row and enqueues the email job, so a rolled back action notifies nobody |
| Live push | After commit, `api` issues `NOTIFY notifications, '<user_id>'`. Each `api` instance listens and pushes to that user's open SSE streams |
| Email retries | 5 attempts with exponential backoff from 30 s. After that the job is discarded and visible in River's failed list |
| Officer digest | One email per officer at 07:00 WIB listing everything marked `digest` from the previous 24 hours, sent only if non empty |
| Payload | `{event, title, body, link, entity_type, entity_id}`. Rendered text is stored, so a later template change does not rewrite history |
| Language | Indonesian only in v1.0 |
| Privacy | No email contains private details, hidden details, the handover code, or the identity of the other party |
