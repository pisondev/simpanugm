# Laporan Full QA dan Rencana Perbaikan SimpanUGM

Tanggal: 21 Oktober 2026
Target: `http://localhost:8000`, `http://localhost:3000`, `http://localhost:8025`

## Status akhir

Semua temuan product/regression yang ditemukan pada QA awal sudah diperbaiki dan diverifikasi ulang. Mapbox tetap dipakai untuk peta admin dan request eksternal yang diizinkan hanya berasal dari host Mapbox yang memang diperlukan.

Catatan non-blocking:

- Warning konfigurasi Vite masih muncul, tetapi tidak menyebabkan test gagal.
- Chromium dapat mengeluarkan error abort resize ketika viewport berubah saat transition native; error ini sudah difilter di harness test karena bukan error aplikasi.
- Peta Mapbox membutuhkan token dan akses ke layanan Mapbox; ini adalah dependency runtime yang disengaja.

## Ringkasan hasil verifikasi

| Area | Hasil |
|---|---:|
| Playwright default | **26 passed** |
| Playwright demo | **35 passed** |
| Frontend Vitest | **75 passed** dalam 16 file |
| Frontend lint dan typecheck | **Lulus** |
| Backend Go unit | **Lulus** |
| Backend Go integration | **Lulus**, package server selesai dalam 42,956 detik |
| Backend `gofmt` dan `go vet` | **Lulus** |
| ML format/lint/typecheck | **Lulus** |
| ML pytest | **61 passed, 6 deselected** |
| Data unittest | **171 passed** |

## Smoke test runtime terakhir

Semua pemeriksaan berikut dilakukan setelah test mutatif selesai dan stack dikembalikan ke mode development:

| Endpoint | Status |
|---|---:|
| `GET http://localhost:8000/healthz` | 200 |
| `GET http://localhost:8000/readyz` | 200 |
| `GET http://localhost:3000/healthz` | 200 |
| `GET http://localhost:8000/barang-temuan` | 200 |
| `GET /api/v1/found-reports/public?limit=1` tanpa session | 200 |
| `GET /api/v1/categories` tanpa session | 200 |
| `GET /api/v1/zones` tanpa session | 200 |
| `GET http://localhost:8025/` | 200 |
| ML health melalui gateway | 200 |

Container stack aktif dan healthy, termasuk PostgreSQL, API, web, Caddy, worker, ML, dan Mailpit. API terverifikasi memakai `APP_ENV=development` tanpa fake clock.

## Cakupan full QA

- Browser in-app langsung pada port 8000, 3000, dan Mailpit 8025.
- Playwright Chromium pada desktop dan phone viewport.
- Landing page, login, registrasi, verifikasi email, daftar barang temuan publik, laporan hilang/temuan, pencarian, klaim, custody, handover, fusion, notifikasi, profil, dashboard anggota, petugas, dan admin.
- Admin: pengguna, master data, peta rawan Mapbox, tren, monitoring AI, klaster, audit, dan antrean klaim.
- Role matrix dan authorization: anggota, petugas, admin, dan anonymous.
- Validasi input, CSRF/origin, RFC problem details, pagination, ETag, projection publik, dan smoke concurrency.
- Mailpit: email verifikasi dan notifikasi.
- Backend unit/integration, frontend unit/lint/typecheck, ML checks, dan data tests.

## Temuan dan perbaikan

### F-01 — Daftar barang temuan publik memerlukan login — RESOLVED

Sebelumnya route `/barang-temuan` dan endpoint `found-reports/public` masih terproteksi. Route sekarang berada di area publik, middleware tidak lagi memaksa login untuk halaman tersebut, dan endpoint publik/category/zone memiliki kontrak anonymous yang eksplisit.

Verifikasi:

```text
GET /barang-temuan                         => 200
GET /api/v1/found-reports/public?limit=1   => 200 tanpa session
GET /api/v1/categories                     => 200 tanpa session
GET /api/v1/zones                          => 200 tanpa session
```

Perubahan terkait: `web/src/proxy.ts`, route publik `web/src/app/(public)/barang-temuan/`, `api/internal/access/access.go`, `api/openapi.yaml`, dan contract/integration test API.

### F-02 — Horizontal overflow pada filter `/barang-temuan` — RESOLVED

Filter dan action group sekarang dapat menyusut/wrap dengan benar melalui `min-w-0` dan layout action yang responsif. Test publik baru memeriksa halaman pada viewport 1280 px dan 360 px, termasuk `scrollWidth` agar overflow horizontal tidak kembali.

### F-03 — Workflow `pnpm` tidak dapat dijalankan — RESOLVED

`web/pnpm-workspace.yaml` sekarang mendeklarasikan package root (`.`). Lint, typecheck, dan Vitest dapat dijalankan melalui workflow `pnpm` normal.

### F-04 — Kontrak accessibility test peta admin tidak konsisten — RESOLVED

Implementasi peta tetap menggunakan landmark `role="region"`, karena lebih sesuai untuk area peta interaktif. Assertion B13 sudah diselaraskan ke role tersebut, termasuk validasi keyboard dan marker. Peta tetap menggunakan Mapbox GL dengan style `mapbox://styles/mapbox/streets-v12`.

Request Mapbox ke `api.mapbox.com` dan `events.mapbox.com` diizinkan secara eksplisit; request eksternal lain tetap dianggap failure oleh test.

### F-05 — Test B09 hardcode jumlah pos — RESOLVED

Test tidak lagi mengharapkan angka jumlah option tertentu. Test sekarang memverifikasi bahwa option tujuan tersedia, memiliki value/label valid, dan tidak memilih pos asal secara invalid. Ini membuat test tetap benar ketika seed menambah pos seperti K5L.

## Rencana perbaikan yang diterapkan

Urutan perbaikannya adalah:

1. **P0 — akses publik:** buka route dan API yang memang didokumentasikan publik, tetap batasi projection agar tidak membocorkan identitas atau data sensitif.
2. **P1 — usability responsive:** hilangkan overflow pada desktop dan mobile, lalu tambahkan regression check berbasis viewport.
3. **P1 — kualitas pipeline:** perbaiki workspace pnpm agar lint, typecheck, dan test berjalan dari perintah standar.
4. **P1 — stabilitas regression suite:** selaraskan kontrak accessibility peta dan ganti assertion berbasis angka hardcode dengan assertion berbasis perilaku.
5. **P2 — regression gate:** jalankan ulang seluruh default/demo suite, integration test, dan smoke endpoint setelah perubahan.
6. **P2 — operasional:** kembalikan environment ke development normal setelah test dan pastikan semua service health/ready tetap 200.

## Kesimpulan

Full QA sudah dijalankan. Setelah perbaikan, seluruh suite utama lulus: **26/26 default E2E** dan **35/35 demo E2E**, termasuk flow publik, role, klaim/custody, admin, responsive, accessibility, dan peta Mapbox. Service siap dipakai untuk demo dan pengujian lanjutan dengan catatan token serta konektivitas Mapbox harus tersedia.
