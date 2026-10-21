# Dashboard penemu barang

Referensi: slide 5, "Fitur untuk Penemu Barang", pada
[presentasi SimpanUGM](https://docs.google.com/presentation/d/1GJvs2N0-C1RR6-5N7AFZt-phUa9ahCwaljLQA13qhes/edit#slide=id.h37bd994acf507109_0_19).

| Fitur pada slide dan antarmuka | Alur yang digunakan |
| --- | --- |
| Found Item Reporting | Formulir temuan dengan foto, lokasi, waktu dan detail barang. |
| AI-Assisted Item Description | Analisis foto dan template deskripsi yang sudah ada, dapat diperiksa dan diedit pada langkah Detail. |
| Instant Match Suggestions | Endpoint `match-hint` pada laporan milik penemu. Menampilkan nama barang dan kategori laporan kehilangan yang mirip. |
| Duplicate / Related Report Detection | Pemeriksaan laporan serupa pada langkah Tinjau sebelum mengirim formulir temuan. |
| Handover to Officer | Kode laporan, status penitipan dan petunjuk pos pada detail temuan; penerimaan fisik dicatat oleh petugas. |
| Case History & Notifications | Riwayat temuan, linimasa kasus dan notifikasi. |

Keenam fitur tersedia. Ilustrasi slide menampilkan persentase kecocokan untuk
penemu, sedangkan API sengaja hanya memberikan nama barang dan kategori demi
privasi. Dashboard mengikuti proyeksi API tersebut tanpa menambahkan skor,
foto laporan kehilangan, kontak atau identitas pemilik. Saran belum merupakan
bukti kepemilikan. Pemeriksaan duplikat memerlukan foto dan detail, sehingga
pintasan dashboard membuka formulir dengan penjelasan prasyarat, bukan hasil
pemeriksaan buatan.

Pemilik dan penemu tetap satu peran teknis `member`. `/beranda?mode=penemu` dan
`/beranda?mode=pemilik` memilih pengalaman penggunaan, dengan pemilih tampilan di
atas header. Tanpa parameter, akun yang hanya memiliki temuan aktif diarahkan
ke tampilan penemu; akun lain memakai tampilan pemilik. Tidak ada pemeriksaan
alamat email khusus untuk menentukan tampilan. Tombol masuk cepat penemu memilih
`mode=penemu` secara eksplisit, sehingga tetap sesuai adegan walaupun akun demo
juga dipakai membuat laporan kehilangan. Hak akses API tetap berlaku.

Nama enam fitur terlihat dalam satu halaman, dengan karakter perempuan di kiri
pada desktop, header kampus, menu berwarna dan pemilih temuan. Petunjuk kecocokan
dan detail penitipan dimuat terpisah dari API agar kegagalan petunjuk tidak
menghalangi akses penyerahan. Data diperbarui setiap 30 detik dan saat tab kembali
aktif. Kondisi kosong, pemuatan dan kegagalan ditampilkan apa adanya.

Ilustrasi berasal dari `Cheerful Girl Tapping Smartphone.png` di `transit-assets`
yang disediakan pemilik, dioptimalkan menjadi WebP 47.816 byte. Aset kampus dan
warna memakai implementasi dashboard pemilik. Tidak ada dependensi atau kontrak
API baru. Dukungan `prefers-reduced-motion` tetap memakai aturan global.

Pemeriksaan e2e dibatasi pada dua skenario `demo-finder-dashboard.spec.ts` dan
satu skenario navigasi pemilik yang terdampak pemilih tampilan bersama, termasuk
axe dan pemeriksaan desktop serta ponsel. Skenario demo hanya berjalan ketika
`DEMO_E2E=1`, mengikuti konfigurasi proyek yang sudah ada.

Hasil pemeriksaan lokal 21 Oktober 2026:

- Satu run gabungan dua skenario penemu dan satu regresi navigasi pemilik:
  3 lulus, 0 gagal, 0 dilewati.
- Axe pada dashboard desktop, ponsel dan kondisi kosong: 0 pelanggaran.
- Prettier, typecheck, lint dan build web: lulus.
- Pemeriksa aturan proyek dan `git diff --check`: bersih.

Suite e2e lengkap dan tes backend tidak dijalankan ulang untuk perubahan antarmuka ini.
