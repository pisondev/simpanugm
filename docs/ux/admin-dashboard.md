# Dashboard administrator

Referensi: slide 7, "Fitur untuk Administrator", pada
[presentasi SimpanUGM](https://docs.google.com/presentation/d/1GJvs2N0-C1RR6-5N7AFZt-phUa9ahCwaljLQA13qhes/edit#slide=id.h37bd994acf507109_0_31).

| Nama fitur pada slide dan antarmuka | Implementasi yang digunakan                                                                                          |
| ----------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| AI & Matching Monitoring            | `/admin/ai/monitoring`: distribusi skor, hasil bantuan foto, umpan balik kandidat dan peringkat kasus terkonfirmasi. |
| Drift Detection                     | `#drift-detection` pada monitoring: PSI dan status peringatan yang dihitung sistem.                                  |
| Fusion Control & Rollback           | `#fusion-control` pada monitoring: versi aktif dan dialog aktivasi atau rollback versi yang sudah tersedia.          |
| HDBSCAN Case Clustering             | `/admin/ai/klaster`: proses pengelompokan visual dan semantik, status proses dan anggota klaster.                    |
| Lost & Found Hotspot Analytics      | `/admin/peta-rawan`: peta koordinat kampus, filter waktu dan jenis laporan, tabel lokasi serta indikator lonjakan.   |
| Trend Analytics                     | `/admin/tren`: volume laporan, kategori, pola waktu kejadian dan ekspor CSV bulanan.                                 |

Keenam fitur sudah tersedia sebelum perubahan dashboard. Lima panel pertama diberi
label AI Feature sesuai presentasi. Analitik hotspot memakai agregasi dan deteksi
lonjakan statistik, bukan model prediksi kehilangan baru. Analitik tren tidak diberi
label AI. Kontrol fusi mengaktifkan atau mengembalikan versi konfigurasi yang telah
terdaftar; dashboard tidak menambahkan penyunting bobot bebas atau pelatihan model.

Administrator masuk ke `/admin`, dengan header kampus, karakter perempuan berkacamata
di kiri pada desktop dan enam panel berwarna. Header menyediakan pengguna, data
master dan pengaturan. Menu Lainnya mempertahankan seluruh navigasi administrasi,
ruang petugas dan akun. Ringkasan petugas, administrator, jumlah pos dan cakupan
penugasan tetap terlihat di bawah panel. Sidebar pada halaman kerja lain tetap ada.

Ringkasan monitoring, klaster dan analitik dimuat dari API di server. Recall@1 hanya
ditampilkan jika ada kasus terkonfirmasi; kondisi tanpa sampel dijelaskan. Nilai
PSI nol tidak disebut bukti kestabilan model. Tidak adanya peringatan ditampilkan
sebagai status sistem dengan pengingat untuk membaca periode dan jumlah datanya.
Kegagalan memuat ringkasan ditampilkan sebagai kondisi tidak tersedia, sementara
tautan halaman fitur tetap dapat dibuka. Tidak ada skor, grafik atau hasil AI buatan.

Halaman tetap dilindungi `requireAdmin` dan otorisasi API. Membuka dashboard tidak
mengaktifkan model, menjalankan clustering, atau mengubah penugasan. Dukungan ponsel
dan `prefers-reduced-motion` memakai pola serta aturan global yang sudah tersedia.

Ilustrasi dari `Cheerful UGM Tech Learner.png` di `transit-assets` milik pemilik
proyek dikompresi menjadi `admin-student.webp` transparan, 94.190 byte. Aset kampus
dan warna mengikuti dashboard sebelumnya. Implementasi dashboard tidak menambah
dependensi atau endpoint. Dependensi peta dan landing page berasal dari pembaruan
Axelle yang digabungkan lebih dahulu dalam commit terpisah.

Peta dasar Mapbox membutuhkan `MAPBOX_ACCESS_TOKEN`. Pemeriksaan lokal mendapati
token belum diisi di `deploy/env/web.env` maupun container web. Tampilan peta
menjelaskan keadaan tersebut; tabel lokasi tetap dapat dibaca. Untuk dev port 3001,
isi variabel itu di lingkungan proses atau `web/.env.local`, lalu mulai ulang dev
server. Untuk Docker, isi `deploy/env/web.env` dan buat ulang service web. Jangan
memasukkan kredensial ke Git. Contoh konfigurasi ada di `deploy/env/web.env.example`.

Pemeriksaan lokal 21 Oktober 2026:

- Satu run regresi setelah penggabungan: 3 skenario dashboard pemilik, penemu dan
  petugas lulus.
- Satu run gabungan setelah implementasi: 4 lulus, 0 gagal, 0 dilewati. Dua skenario
  baru administrator, satu regresi dashboard petugas dan satu regresi monitoring
  serta dialog kontrol model.
- Axe pada dashboard administrator dan monitoring, desktop serta ponsel: 0 pelanggaran.
- Prettier, typecheck, lint dan build web: lulus.
- Spec baru `demo-admin-dashboard.spec.ts` hanya berjalan dengan `DEMO_E2E=1`.
- Pengujian hanya membaca data dan membuka dialog, tanpa mengaktifkan versi model,
  menjalankan pengelompokan atau mengubah cerita demo.

Suite e2e lengkap dan tes backend tidak dijalankan ulang untuk perubahan antarmuka ini.
