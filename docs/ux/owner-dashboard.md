# Dashboard pemilik barang

Referensi: slide 4, "Fitur untuk Pemilik Barang", pada
[presentasi SimpanUGM](https://docs.google.com/presentation/d/1GJvs2N0-C1RR6-5N7AFZt-phUa9ahCwaljLQA13qhes/edit#slide=id.h37bd994acf507109_0_13).

## Kesesuaian fitur dengan implementasi

| Nama pada presentasi dan antarmuka | Implementasi yang digunakan |
| --- | --- |
| Lost Item Reporting | `/lapor/hilang`: kategori, ciri, lokasi, waktu dan foto opsional. |
| Smart Report Assistant | Formulir kehilangan, unggah foto lalu lanjut ke detail: kategori, warna, kandidat merek dan draf deskripsi yang dapat dikoreksi. |
| Match Candidates | Kandidat dari `/lost-reports/{id}/matches`, dibatasi sesuai izin pemilik, menuju bagian kandidat pada detail laporan. |
| Explainable Match | Detail kandidat: alasan visual atau lintas modal, deskripsi, lokasi dan waktu. Ringkasan dashboard memakai tingkatan yang diberikan API. |
| Claim Submission | Formulir klaim pada pasangan laporan yang dipilih, bukti dan pertanyaan verifikasi. Klaim tetap diputuskan petugas. |
| Case History & Notifications | Linimasa laporan, daftar klaim dan notifikasi yang diperbarui melalui SSE. |

Keenam fitur tersedia. Sebelum penyesuaian ini, draf deskripsi otomatis hanya
diaktifkan pada formulir temuan; formulir kehilangan kini memakai helper yang sama.
Draf deskripsi menggunakan template dari atribut hasil
analisis, bukan LLM. Saran kategori, warna dan merek tidak dijamin benar. Skor
kandidat adalah skor pemeringkatan, bukan probabilitas kepemilikan.

## Perilaku dashboard

Anggota masuk ke `/beranda` melalui alur login yang sudah ada. Tampilan dapat
dipilih melalui pemilih dashboard. Akun dengan hanya laporan temuan aktif memakai
[dashboard penemu](finder-dashboard.md) secara otomatis. Pemilik dan penemu
tetap satu peran teknis `member`; dashboard menyediakan alur pemilik dan akses
laporan temuan dari akun yang sama. Hak akses petugas dan administrator tidak diubah.

Pada desktop, ilustrasi karakter ada di kiri, header memakai ilustrasi kampus,
dan enam fitur tersusun dengan warna yang mengikuti presentasi. Nama fitur juga
ditampilkan pada layar tujuan, dengan keterangan dan tombol dalam bahasa Indonesia.
Pada layar kecil, menu menjadi satu kolom dan ilustrasi karakter disembunyikan.

Pemilih laporan menghubungkan kandidat, alasan, klaim dan linimasa pada laporan
yang sama. Kandidat dimuat dari API nyata dan diperbarui setiap 30 detik serta saat
jendela kembali aktif. Tidak ada contoh skor atau kandidat buatan. Jika laporan
atau kandidat belum ada, nama fitur tetap tampil dengan penjelasan prasyarat dan
aksi yang belum tersedia dinonaktifkan. Kegagalan memuat kandidat memiliki tombol
coba lagi; pelaporan dan riwayat tetap dapat digunakan.

Klaim terbaru dan pintasan kode serah terima atau balasan informasi ditampilkan
jika keadaan tersebut ditemukan dalam lima klaim terbaru. Dashboard hanya
menampilkan ringkasan, dengan tautan menuju daftar lengkap.

Aset lokal berasal dari folder `transit-assets` yang disediakan pemilik. Catatan
asalnya ada pada `web/public/illustrations/README.md`. Ukuran gabungan dua WebP
adalah 155.306 byte. Tidak ada dependensi, layanan AI, atau kontrak API baru.

## Pemeriksaan

`demo-owner-dashboard.spec.ts` dijalankan dengan `DEMO_E2E=1`. Pemeriksaan mencakup
login pemilik, tautan ke fitur nyata, pemilihan laporan, aksesibilitas, tampilan
sempit, kegagalan kandidat, akun anggota tanpa laporan dan draf deskripsi dari
endpoint template yang sebenarnya. Jalankan pengujian
terhadap stack demo lokal yang sudah berjalan dan frontend yang sedang ditinjau.

Hasil pemeriksaan lokal 21 Oktober 2026:

| Pemeriksaan | Hasil |
| --- | --- |
| `pnpm exec prettier --check .`, `pnpm typecheck`, `pnpm lint`, `pnpm build` | Lulus |
| `pnpm test` | 75 tes lulus pada 16 berkas |
| `DEMO_E2E=1 pnpm exec playwright test demo-owner-dashboard demo-01-polish --project=desktop` | Satu run gabungan: 7 lulus, 0 gagal, 0 dilewati |
| `pnpm exec playwright test j1-register` tanpa `DEMO_E2E` | Satu run ponsel dan desktop: 8 lulus, 0 gagal, 0 dilewati |
| Axe WCAG A/AA pada layar yang diuji | 0 pelanggaran |
| `python deploy/scripts/demo_check.py` | 20 lulus, 0 gagal, 0 peringatan |
| Pemeriksa aturan proyek dan `git diff --check` | Bersih |

Pengujian e2e tersebut difokuskan pada dashboard dan alur yang terdampak perubahan.
Jalankan build sebelum pengujian inferensi nyata agar keduanya tidak berebut CPU.
