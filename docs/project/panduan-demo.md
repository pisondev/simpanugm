# Panduan Demo SimpanUGM

Edisi UI baru, 22 Oktober 2026. Acuan kode antarmuka: `babf757`, branch `feat/web-login-demo-guide`.

## Panduan Demo SimpanUGM

UI baru | Demo utama 12 menit | Data dan model nyata di laptop

**Sistem Informasi Manajemen Penemuan dan Pengembalian Barang di lingkungan UGM.** Pemilik mencari barang, penemu melaporkan temuan, petugas memeriksa bukti dan mencatat serah terima, administrator memantau operasional serta kualitas pencocokan.

> **Pesan utama:** AI membantu mempersempit kandidat dan membaca alasan kecocokan. Keputusan kepemilikan serta serah terima tetap diverifikasi petugas.

| Tahap         | Yang dilakukan                                                                                       |
| ------------- | ---------------------------------------------------------------------------------------------------- |
| 1. Lapor      | Foto, deskripsi, kategori, lokasi dan waktu. Kehilangan boleh tanpa foto; temuan wajib memakai foto. |
| 2. Cocokkan   | CLIP dan E5 membantu pemeringkatan kandidat. Sinyal lokasi, waktu dan kategori ikut digabungkan.     |
| 3. Verifikasi | Pemilik memberi alasan dan jawaban kategori. Petugas memeriksa bukti serta checklist.                |
| 4. Kembalikan | Barang sudah di pos, klaim disetujui, identitas diperiksa dan kode aktif cocok.                      |

### Cara membaca panduan

| Bagian                                              | Halaman      |
| --------------------------------------------------- | ------------ |
| Persiapan, akun dan daftar demo                     | 2 sampai 4   |
| Fitur sesuai enam panel tiap dashboard              | 5 sampai 6   |
| Foto dan contoh isian latihan                       | 7 sampai 12  |
| Kandidat, klaim dan pekerjaan petugas               | 13 sampai 16 |
| Administrator, demo opsional dan penanganan kendala | 17 sampai 19 |
| Kredit foto dan acuan                               | 20           |

Nama fitur di panduan sama dengan yang terlihat pada dashboard. **Fitur AI dicetak tebal dan disorot kuning.** Ilustrasi karakter mewakili peran, bukan foto orang atau tim. Proyek ini bukan layanan resmi UGM.

## Gunakan versi UI yang benar

Selesaikan build dan latihan sebelum presentasi dimulai.

| Alamat                        | Kapan dipakai                                                                               |
| ----------------------------- | ------------------------------------------------------------------------------------------- |
| `http://localhost:3001/masuk` | Preview UI terbaru yang saat panduan ini dibuat sudah berjalan. Pilih karakter untuk masuk. |
| `http://localhost:8000/masuk` | Versi Docker. Tampilannya baru setelah web dibangun dari branch yang sama.                  |
| `http://localhost:8025`       | Mailpit untuk verifikasi email atau pemulihan akun uji.                                     |

### Jika menyiapkan ulang dari repository

```powershell
git switch feat/web-login-demo-guide
git pull --ff-only
make dev
make demo-check
```

Gunakan branch UI di atas. Jangan pindah ke main lama lalu mengira dashboard baru sudah ikut dibangun. Jika port 3001 sudah menampilkan UI baru, gunakan proses yang berjalan itu.

### Menjalankan preview port 3001 sendiri

```powershell
$env:API_INTERNAL_URL='http://127.0.0.1:8090'
$env:NEXT_PUBLIC_DEMO_MODE='true'
cd web
pnpm dev --port 3001
```

API, database, worker dan ML tetap berasal dari stack Docker. Bila make tidak menemukan sh, tambahkan Git Bash ke PATH terminal sesuai docs/project/demo-script.md. Jangan menjalankan build berat saat menunggu saran foto.

> **Reset hanya jika diperlukan:** `make demo` mengembalikan cerita demo dan dapat menghapus data latihan. Jalankan hanya saat data latihan boleh hilang, lalu ulangi `make demo-check`. Jangan reset di tengah presentasi.

**Snapshot kesiapan:** 20 pemeriksaan lulus, 0 gagal, 0 peringatan. Payung siap serah, kunci masih di penemu dan kalkulator meminta informasi. Periksa lagi sebelum tampil karena latihan dapat mengubah status. Model berjalan lokal; peta dasar Mapbox membutuhkan token dan internet.

## Pilih karakter, jalankan adegan

Kata sandi demo untuk semua akun: kata sandi panjang 2026

_Screenshot PDF halaman 3: Pilihan Masuk cepat mode demo pada halaman masuk. Login manual tersedia di atasnya._

| Tombol / akun                       | Email                                | Tujuan                                 |
| ----------------------------------- | ------------------------------------ | -------------------------------------- |
| Penemu (Mahasiswa); Budi Santoso    | anggota.penemu@demo.simpanugm.my.id  | Dashboard penemu; saran foto           |
| Pemilik (Mahasiswa); Siti Rahma     | anggota.pemilik@demo.simpanugm.my.id | Dashboard pemilik; kandidat dan klaim  |
| Petugas (Pos Perpus); Agus Prasetyo | petugas.pos1@demo.simpanugm.my.id    | Pos Perpus; checklist dan serah terima |
| Administrator; Admin SimpanUGM      | admin@demo.simpanugm.my.id           | Dashboard administrator                |
| Manual: Dewi Lestari                | petugas.pos2@demo.simpanugm.my.id    | Tindak lanjut kalkulator               |
| Manual: Anggota Demo                | anggota@demo.simpanugm.my.id         | Penemu kunci; akun tambahan            |

## Urutan presentasi 12 menit

Utamakan cerita seed; latihan formulir baru tersedia sebagai adegan tambahan.

| Waktu              | Adegan utama                       | Titik selesai                                         |
| ------------------ | ---------------------------------- | ----------------------------------------------------- |
| 00:00 sampai 01:00 | Pengantar dan halaman masuk        | Jelaskan tujuan dan empat pilihan karakter            |
| 01:00 sampai 03:30 | Budi: dashboard dan foto dompet    | Perlihatkan hasil AI asli, koreksi dan tinjauan       |
| 03:30 sampai 06:00 | Siti: kandidat earbuds dan klaim   | Empat alasan kecocokan dan pertanyaan kategori        |
| 06:00 sampai 08:30 | Agus: dashboard, checklist, payung | Serah terima hanya bila sedang menjalankan sesi nyata |
| 08:30 sampai 11:00 | Admin: monitoring dan klaster      | Tunjukkan metrik dan batas interpretasinya            |
| 11:00 sampai 12:00 | Tanya jawab                        | Simpulkan bantuan AI dan keputusan manusia            |

**Demo opsional:** isi kehilangan baru (halaman 11 sampai 12), terima kunci, balas permintaan info kalkulator, cari semantik, tinjau duplikat, peta atau tren. Pemilik dan penemu tetap satu role teknis member. Keluar melalui menu akun sebelum mengganti karakter pada browser yang sama.

> **Objek berbeda untuk tiap adegan:** dompet untuk saran foto dan latihan formulir, earbuds untuk kandidat serta klaim, kunci untuk checklist, dan payung untuk serah terima yang sudah siap. Jelaskan perpindahan kasus ini; laporan baru tidak otomatis menjadi kasus siap serah.

### Titik berhenti saat latihan

Formulir laporan: berhenti pada Tinjau. Klaim: berhenti sebelum Kirim. Serah terima: cukup buka formulir tanpa Konfirmasi. Untuk menjalankan aksi sungguhan saat presentasi, ikuti pemeriksaan pada halaman adegannya. Status dan kode harus diambil dari aplikasi yang sedang berjalan.

## Pemilik dan penemu

Nama panel ditulis persis seperti pada UI. Sorotan kuning menandai fitur AI.

### Pemilik barang

- **Lost Item Reporting**: Buat laporan kehilangan, foto opsional, kemudian isi detail, lokasi/waktu dan ciri khusus.

- **Smart Report Assistant** [AI]: Mulai dari foto, periksa kategori, atribut dan draf deskripsi yang bisa diedit.

- **Match Candidates** [AI]: Lihat kandidat untuk laporan yang dipilih lewat Laporan yang ditinjau.

- **Explainable Match** [AI]: Periksa empat sinyal foto/teks-foto, makna deskripsi, lokasi dan waktu.

- **Claim Submission**: Ajukan klaim atas kandidat yang sesuai dan jawab pertanyaan kategori.

- **Case History & Notifications**: Ikuti riwayat kasus, pesan dan notifikasi; buka tindak lanjut klaim yang tersedia.

### Penemu barang

- **Found Item Reporting**: Laporkan temuan dengan foto wajib dan rencana penyerahan barang.

- **AI-Assisted Item Description** [AI]: Periksa saran atribut dan draf deskripsi dari foto, lalu koreksi.

- **Instant Match Suggestions** [AI]: Petunjuk nama/kategori kehilangan yang mirip, tanpa identitas pemilik atau skor persen.

- **Duplicate / Related Report Detection** [AI]: Periksa laporan serupa pada langkah Tinjau setelah detail cukup.

- **Handover to Officer**: Lihat kode laporan, pos tujuan dan petunjuk penerimaan fisik oleh petugas.

- **Case History & Notifications**: Lihat riwayat temuan dan perkembangan melalui notifikasi.

## Petugas dan administrator

Bantuan AI tidak menggantikan pemeriksaan bukti atau keputusan manusia.

### Petugas

- **AI-Assisted Priority Queue** [AI]: Prioritas menggabungkan skor kecocokan dan urgensi operasional.

- **AI Match Review Board** [AI]: Bandingkan laporan serta kandidat dan alasan kecocokannya.

- **Semantic Verification Checklist** [AI]: Periksa detail/jawaban dan saran checklist; hasil akhirnya dinilai petugas.

- **Semantic Search** [AI]: Cari berdasarkan makna deskripsi dengan mesin pencarian yang sudah ada.

- **Duplicate Review** [AI]: Tinjau pasangan serupa sebelum penggabungan yang memerlukan konfirmasi.

- **Handover to Owner**: Periksa klaim disetujui, barang di pos, identitas dan kode aktif.

### Administrator

- **AI & Matching Monitoring** [AI]: Baca versi aktif, distribusi skor dan peringkat kasus terkonfirmasi.

- **Drift Detection** [AI]: PSI dan peringatan pergeseran distribusi, dengan konteks jumlah dan periode data.

- **Fusion Control & Rollback** [AI]: Aktivasi atau kembali ke versi konfigurasi yang sudah tersedia.

- **HDBSCAN Case Clustering** [AI]: Kelompokkan kemiripan visual dan semantik untuk meninjau pola kasus.

- **Lost & Found Hotspot Analytics** [AI]: Agregasi lokasi serta deteksi lonjakan statistik; peta dasar memakai Mapbox.

- **Trend Analytics**: Volume, kategori dan pola waktu; ekspor CSV bulanan. Bukan fitur AI baru.

## Siapkan foto sebelum tampil

Seluruh path gambar berikut relatif terhadap folder repository simpanugm.

| Berkas dalam data/demo/images/                    | Dipakai untuk                                                                      |
| ------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `hero-wallet-found.jpg`                           | Foto wajib laporan temuan latihan. Objek hitam dengan tepi cokelat.                |
| `hero-wallet-lost.jpg`                            | Foto opsional laporan kehilangan latihan; turunan crop dari foto dompet yang sama. |
| `hero-earbuds-lost.jpg`; `hero-earbuds-found.jpg` | Sudah dipakai cerita seed untuk kandidat dan klaim; tidak perlu diunggah ulang.    |
| `hero-keys-found.jpg`                             | Cerita kunci yang belum diterima pos; bukan untuk membuat laporan baru.            |
| `hero-umbrella-found.jpg`                         | Cerita payung siap serah terima.                                                   |
| `hero-calculator-found.jpg`                       | Cerita kalkulator yang meminta informasi tambahan.                                 |

_Screenshot PDF halaman 7: wallet-found.jpg_

_Screenshot PDF halaman 7: earbuds-found.jpg_

_Screenshot PDF halaman 7: keys-found.jpg_

_Screenshot PDF halaman 7: umbrella-found.jpg_

> **Contoh latihan bukan keluaran AI.** Nama, deskripsi dan tulisan DEMO-SITI pada halaman formulir berikut adalah narasi contoh untuk latihan. Merek tidak ditebak dari foto. Nama/merek cerita seed juga tidak berarti merek asli foto Commons.

Foto kehilangan maksimal 5 dan opsional; foto temuan minimal 1, maksimal 5. Batas pemilih foto 10 MB per berkas. Gunakan JPG, PNG atau WebP; HEIC/HEIF dikonversi aplikasi. Bukti klaim opsional, maksimal 3 foto. Jangan mengunggah dokumen identitas asli; gunakan contoh bertanda CONTOH.

Sumber foto, pembuat dan lisensi ada pada halaman 20 serta data/demo/SOURCES.csv. Ilustrasi karakter berasal dari transit-assets yang disediakan pemilik proyek.

## Dashboard penemu dan saran foto

Klik karakter Penemu (Mahasiswa), Budi Santoso, pada halaman masuk.

_Screenshot PDF halaman 8: Dashboard penemu: pilih laporan pada Temuan yang ditinjau; enam panel tetap terlihat._

1. **Tunjukkan enam fitur penemu**: Panel **Found Item Reporting** memulai formulir. Panel **AI-Assisted Item Description** memiliki tombol **Mulai dari foto**. Pilih tombol itu.

2. **Unggah contoh dompet**: Pada langkah Foto, pilih `data/demo/images/hero-wallet-found.jpg`. Tunggu unggahan selesai, lalu tekan **Lanjut**.

3. **Periksa hasil model asli**: Tunjukkan status penyiapan saran, kategori, warna, merek bila ada dan draf deskripsi. Jangan menyebut hasil tertentu sebelum hasilnya terlihat. Koreksi bila tidak sesuai objek.

_Screenshot PDF halaman 8: Saran pada sesi pengambilan gambar. Hasil bisa berubah; pengguna tetap mengoreksi sebelum mengirim._

## Isi laporan temuan per langkah

Contoh latihan: Dompet lipat hitam cokelat demo. D-1 berarti kemarin dari hari demo.

| Langkah / kolom      | Wajib?   | Contoh yang diisi                                                                                   |
| -------------------- | -------- | --------------------------------------------------------------------------------------------------- |
| 1. Foto              | Ya       | Unggah hero-wallet-found.jpg. Tunggu unggahan selesai.                                              |
| 2. Kategori          | Ya       | Dompet. Pilih hasil daftar, bukan hanya mengetik pencariannya.                                      |
| Nama barang          | Ya       | Dompet lipat hitam cokelat demo                                                                     |
| Deskripsi            | Ya       | Dompet kulit lipat dua warna hitam dengan tepi cokelat dan jahitan krem. Ditemukan di meja belajar. |
| Warna                | Opsional | Hitam dan cokelat. Maksimal 2 warna.                                                                |
| Merek                | Opsional | Kosongkan bila tidak terbaca; jangan menganggap merek seed sebagai hasil foto.                      |
| 3. Tempat ditemukan  | Ya       | Cari perpus, lalu pilih Perpustakaan Pusat UGM.                                                     |
| Catatan lokasi       | Opsional | Lantai 2, area meja belajar.                                                                        |
| Tanggal dan Jam      | Ya       | D-1, pukul 10.00 WIB. Tidak boleh waktu mendatang.                                                  |
| 4. Penyerahan barang | Ya       | Saya serahkan ke pos. Untuk Dompet, jangan pilih simpan sementara.                                  |
| Pos tujuan           | Ya       | Pos Lost and Found Perpustakaan Pusat. Klik pilihan pos itu secara eksplisit.                       |
| Detail tersembunyi   | Opsional | Contoh latihan: ada tulisan DEMO-SITI di kantong bagian dalam.                                      |
| 5. Tinjau            | Ya       | Periksa foto, data, pos dan saran duplikat sebelum Kirim laporan.                                   |

> **Titik berhenti latihan:** cukup sampai Tinjau. Jika benar-benar ingin menambah laporan, pilih **Kirim laporan**, lalu simpan kode FND yang dihasilkan. Barang belum dianggap masuk pos hanya karena laporan terkirim; petugas tetap mencatat penerimaan fisik.

Deskripsi temuan 10 sampai 500 karakter; nama barang 3 sampai 80. Ciri rahasia jangan ditulis dalam deskripsi publik. Bila ada duplikat, buka dan periksa; jangan mengabaikan atau menggabungkan tanpa memastikan barangnya.

## Dashboard pemilik dan kasus seed

Klik karakter Pemilik (Mahasiswa), Siti Rahma. Dashboard membuka tampilan pemilik.

_Screenshot PDF halaman 10: Dashboard pemilik. Pilihan Laporan yang ditinjau menentukan kandidat, alasan dan pintasan klaim._

1. **Pilih kasus yang siap**: Pada **Laporan yang ditinjau**, pilih **TWS Earbuds Jabra Elite 4 Active**. Ini laporan seed yang sudah mempunyai kandidat; tidak perlu membuat kehilangan baru untuk adegan utama.

2. **Jelaskan susunan fitur**: **Lost Item Reporting** untuk laporan baru; **Smart Report Assistant** untuk opsi foto. **Match Candidates**, **Explainable Match** dan **Claim Submission** mengikuti laporan terpilih. Riwayat dan notifikasi ada pada panel terakhir serta header.

3. **Buka kandidat**: Pilih **Lihat kandidat**. Bandingkan foto dan nama barang, lokasi serta pos penyimpanan. Pilih kandidat untuk melihat alasannya. Lanjut ke halaman 13.

> **Jika ingin mendemokan laporan kehilangan baru:** gunakan resep lengkap pada halaman 11 sampai 12. Resep itu latihan formulir tersendiri; jangan menjanjikan kandidat atau klaim langsung disetujui setelah laporan baru dikirim.

## Foto dan detail kehilangan

Dari panel Lost Item Reporting pilih Laporkan kehilangan.

| Langkah / kolom | Wajib? | Isian latihan                                                                                                                |
| --------------- | ------ | ---------------------------------------------------------------------------------------------------------------------------- |
| 1. Foto         | Tidak  | Pilih Lewati, saya tidak punya foto untuk latihan manual. Bila ingin bantuan foto, gunakan hero-wallet-lost.jpg.             |
| 2. Kategori     | Ya     | Dompet. Buka pemilih Kategori, cari dompet, pilih Dompet.                                                                    |
| Nama barang     | Ya     | Dompet lipat hitam cokelat demo                                                                                              |
| Deskripsi       | Ya     | Dompet kulit lipat dua warna hitam dengan tepi cokelat dan jahitan krem. Terakhir dibawa di meja belajar Perpustakaan Pusat. |
| Warna           | Tidak  | Hitam dan cokelat; maksimal 2 warna.                                                                                         |
| Merek           | Tidak  | Kosongkan jika tidak yakin. Jangan menyalin Eiger dari kasus seed ke contoh foto ini.                                        |

_Screenshot PDF halaman 11: Langkah Detail setelah contoh isian manual dilengkapi. Merek sengaja kosong._

Kehilangan menerima deskripsi 20 sampai 1000 karakter, nama barang 3 sampai 80 karakter dan merek maksimal 40 karakter. Tekan **Lanjut** setelah kategori benar-benar dipilih.

## Lokasi, waktu dan ciri khusus

Tanggal disesuaikan dengan hari demo; jangan menyalin tanggal lama dari screenshot.

| Kolom / tindakan            | Wajib?          | Isian latihan                                                                                                     |
| --------------------------- | --------------- | ----------------------------------------------------------------------------------------------------------------- |
| 3. Tempat terakhir terlihat | Ya / alternatif | Cari perpus, pilih Perpustakaan Pusat UGM. Maksimal 3 tempat. Jika tidak tahu, centang Saya tidak tahu lokasinya. |
| Catatan lokasi              | Tidak           | Lantai 2, area meja belajar.                                                                                      |
| Tanggal                     | Ya              | D-1: satu hari sebelum hari demo.                                                                                 |
| Lebih dari satu hari        | Tidak           | Biarkan tidak dicentang untuk contoh ini.                                                                         |
| Waktu                       | Ya              | Pilih Pagi, rentang 06.00 sampai 11.00 WIB. Atur jam tersedia bila ingin rentang sendiri.                         |
| 4. Ciri khusus              | Tidak           | Contoh latihan: ada tulisan DEMO-SITI di kantong bagian dalam. Maksimal 500 karakter.                             |
| 5. Tinjau                   | Ya              | Cocokkan semua data. Pilih Ubah bagian bila perlu, atau Kembali untuk memperbaiki.                                |

_Screenshot PDF halaman 12: Langkah Lokasi dan waktu. Pilih hasil pencarian lokasi, bukan hanya isi teks pencarian._

> **Latihan aman:** berhenti pada Tinjau. **Demo pengiriman:** klik Kirim laporan, simpan kode LST yang dihasilkan, lalu tunggu pencocokan aktual. Jangan menggunakan kode LST/CLM dari PDF sebagai nilai tetap. Rentang kehilangan maksimal 7 hari; kejadian maksimal 90 hari lalu.

## Baca alasan sebelum mengklaim

Gunakan kasus earbuds seed, bukan mengharapkan contoh formulir langsung punya kandidat.

1. **Match Candidates**: Di dashboard Siti pilih TWS Earbuds Jabra Elite 4 Active, lalu **Lihat kandidat**. Periksa nama, foto, persentase gabungan, lokasi dan pos. Buka kandidat yang sesuai.

_Screenshot PDF halaman 13: Kandidat nyata untuk laporan earbuds pada sesi ini. Urutan dan skor tidak dijanjikan selalu sama._

2. **Explainable Match**: Tunjukkan keempat meter: foto atau deskripsi-foto, kemiripan makna deskripsi, lokasi dan waktu. Baca alasan yang tertulis, termasuk sinyal rendah atau data yang tidak tersedia.

_Screenshot PDF halaman 13: Empat alasan membantu membandingkan bukti. Persentase bukan probabilitas kepemilikan._

3. **Claim Submission**: Pilih **Ini milik saya** pada detail kandidat atau **Ajukan klaim** di dashboard untuk pasangan terpilih. Jangan menekan Bukan milik saya pada kandidat utama saat latihan.

## Isi bukti, bukan sekadar klik

Contoh berikut khusus cerita earbuds seed. Nama merek adalah narasi demo.

| Kolom / bagian                                                         | Wajib? | Contoh isian atau tindakan                                                                                                                                         |
| ---------------------------------------------------------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Laporan kehilangan terkait                                             | Ya     | Pastikan yang dipilih TWS Earbuds Jabra Elite 4 Active milik Siti. Jangan memilih kehilangan baru yang tidak terkait.                                              |
| Mengapa barang ini milikmu?                                            | Ya     | Earbuds ini milik saya. Case memiliki goresan kecil di sisi kiri dan nomor seri contoh SN-JBR4-9821. Terakhir dipakai di meja belajar lantai 2 Perpustakaan Pusat. |
| Sebutkan merek, warna, dan ciri casing atau earphone.                  | Ya     | Jabra, hitam, casing charging dengan goresan kecil di sudut kiri.                                                                                                  |
| Sebutkan aksesori atau kerusakan khusus yang tidak terlihat dari luar. | Ya     | Nomor seri contoh di case SN-JBR4-9821; goresan pada sisi kiri case.                                                                                               |
| Bukti kepemilikan                                                      | Tidak  | Boleh kosong untuk cerita seed. Bila perlu contoh foto: hero-earbuds-lost.jpg. Maksimal 3 foto; jangan unggah identitas asli.                                      |
| Rencana waktu pengambilan                                              | Tidak  | Besok pukul 10.00 WIB setelah petugas menyetujui.                                                                                                                  |
| Kirim klaim                                                            | Aksi   | Hanya klik saat adegan memang mengirim klaim baru. Petugas masih harus menilai dan mengambil keputusan.                                                            |

_Screenshot PDF halaman 14: Formulir yang sudah diisi sampai sebelum Kirim klaim. Tidak dikirim saat pengambilan screenshot._

> **Latihan:** cukup buka dan isi; jangan mengirim berulang. Jika aplikasi sudah menampilkan klaim yang ada atau melarang klaim ganda, buka klaim tersebut dan jelaskan statusnya. Klaim baru tidak otomatis disetujui atau siap serah.

## Prioritas, papan dan checklist

Klik Petugas (Pos Perpus). Cakupan awal Pos saya; kewenangan tetap mengikuti pos.

_Screenshot PDF halaman 15: Enam panel petugas. Pilih Kasus yang ditinjau dan klik Tinjau kasus untuk memperbarui pilihan._

1. **Pilih kasus yang mempunyai klaim**: Gunakan kasus **Kunci Motor Honda Gantungan Kulit** pada pemilih kasus. **AI Match Review Board** membuka laporan itu; **Semantic Verification Checklist** membuka klaimnya. Antrean lengkap tetap ada di bawah panel.

2. **Periksa bukti, bukan hanya skor**: Pada papan, bandingkan dua laporan dan alasan sinyal. Pada checklist, cocokkan saran dengan foto, barang fisik dan jawaban. Isi hasil Lulus/Gagal/Tidak berlaku sesuai pemeriksaan nyata, lalu simpan hanya jika sedang menjalankan adegan keputusan.

_Screenshot PDF halaman 15: Checklist pada klaim yang sedang ditinjau. Kunci masih di penemu; jangan menyetujui sebelum barang diterima._

Prioritas adalah urutan pekerjaan berbantuan skor dan urgensi, bukan kepastian pemilik. Semantic Search menerima frasa, misalnya **alat hitung scientific hitam**. Duplicate Review tetap memerlukan peninjauan sebelum penggabungan.

## Payung yang sudah siap di pos

Gunakan dua sesi browser atau keluar-masuk akun. Kode diambil dari layar Siti saat demo.

1. **Siti membuka klaim disetujui**: Di dashboard pemilik, buka daftar klaim melalui tindak lanjut atau `/klaim`. Pilih **Payung Lipat Hitam Otomatis** berstatus Disetujui. Catat kode serah terima 6 digit yang sedang aktif dan pos pengambilan.

2. **Agus membuka Handover to Owner**: Baca nama barang dan kode klaim pada panel sebelum membuka. Pintasan bisa menunjuk kasus berbeda dari pemilih kasus. Jika bukan payung, buka Semua klaim pos lalu pilih payung yang sama dengan Siti.

| Kolom / pemeriksaan     | Isian atau tindakan                                                                                    |
| ----------------------- | ------------------------------------------------------------------------------------------------------ |
| Kode serah terima       | Salin 6 digit yang sedang tampil pada akun Siti. Jangan menyalin angka dari screenshot lama.           |
| Jenis identitas         | KTM untuk contoh yang dipakai tim; gunakan kartu bertanda CONTOH.                                      |
| Empat digit terakhir    | 3107, hanya nomor contoh. Jangan masukkan NIK atau nomor identitas lengkap.                            |
| Nama sesuai identitas   | Centang setelah nama contoh dan orang yang mengambil telah diperiksa.                                  |
| Konfirmasi Serah Terima | Klik hanya setelah barang, klaim, identitas dan kode benar. Tunjukkan status serta riwayat sesudahnya. |

_Screenshot PDF halaman 16: Formulir serah terima di detail klaim. Kode aktif tidak dicetak dalam panduan ini._

> **Aksi sekali pakai:** konfirmasi menghabiskan cerita payung siap-serah. Untuk latihan tanpa mengubah keadaan, cukup buka formulir. Jika kode kedaluwarsa, pulihkan cerita di luar sesi dan jalankan demo-check. Jangan mencoba kode acak.

## Baca kualitas dan pola operasional

Klik Administrator pada /masuk. Enam panel memberi pintasan ke fitur yang sudah ada.

_Screenshot PDF halaman 17: Dashboard administrator dengan ringkasan nyata. Metrik demo selalu dibaca bersama jumlah sampelnya._

1. **Monitoring dan drift**: Buka **AI & Matching Monitoring**. Baca versi aktif, distribusi skor dan jumlah kasus terkonfirmasi. **Drift Detection** langsung menuju PSI. Nilai PSI nol atau tanpa peringatan bukan bukti akurasi umum.

2. **Tunjukkan kontrol, jangan ubah saat latihan**: **Fusion Control & Rollback** membuka bagian versi. Klik **Kelola & Rollback Model** untuk melihat dialog, lalu **Batal**. Ini memilih versi yang tersedia, bukan menyunting bobot bebas.

3. **Klaster, hotspot dan tren**: Buka **HDBSCAN Case Clustering** untuk anggota kelompok yang sudah ada. Gunakan **Lost & Found Hotspot Analytics** untuk sebaran lokasi dan **Trend Analytics** untuk pola waktu/kategori serta ekspor CSV.

> **Konfigurasi peta:** MAPBOX_ACCESS_TOKEN belum tersedia pada lingkungan lokal yang diperiksa. Peta menjelaskan kondisi itu; tabel lokasi tetap dapat dibaca. Isi token di web/.env.local untuk port 3001 atau deploy/env/web.env untuk Docker, lalu mulai ulang proses web. Jangan memasukkan token ke Git.

Peta memakai agregasi serta indikator lonjakan statistik. Klaster bukan bukti bahwa semua anggotanya barang yang sama. Model tidak dilatih ulang hanya dengan membuka dashboard.

## Terima kunci dan jawab permintaan info

Kedua adegan mengubah status. Jalankan setelah contoh utama atau sesuai kebutuhan presentasi.

### A. Penerimaan kunci dari penemu

| Langkah / kolom        | Isi atau tindakan                                                                                                       |
| ---------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| Cari kode kunci        | Buka klaim atau temuan Kunci Motor Honda Gantungan Kulit. Salin kode FND aktif; jangan mengandalkan nomor tetap di PDF. |
| Agus: Terima Barang    | Masukkan kode FND pada pencarian, buka hasil dan cocokkan foto dengan barang contoh.                                    |
| Pos penerima           | Pos Lost and Found Perpustakaan Pusat.                                                                                  |
| Rak / slot penyimpanan | Loker A3.                                                                                                               |
| Kondisi barang         | Kunci dan gantungan lengkap, contoh latihan.                                                                            |
| Konfirmasi penerimaan  | Klik hanya bila benar-benar menerima fisik barang. Pastikan status kemudian Disimpan di pos.                            |

### B. Pemilik menjawab permintaan informasi

1. **Siti membuka klaim kalkulator**: Buka `/klaim`, pilih **Kalkulator Ilmiah Casio fx-991EX** berstatus Perlu Informasi. Baca pertanyaan petugas dan isi kotak respons.

> **Jawaban contoh cerita seed:** Di balik cover ada stiker kartun One Piece dan tulisan nama Siti. Ini narasi contoh, bukan klaim bahwa detail tersebut terlihat pada foto Commons.

2. **Kirim dan tunjukkan tindak lanjut**: Kirim jawaban hanya sekali saat menjalankan adegan. Klaim kembali ke review. Dewi, petugas Pos Saintek, dapat membuka klaim yang sama untuk membaca jawaban. Gunakan login manual untuk akun Dewi.

Penemu kunci adalah akun tambahan Anggota Demo. Akun Budi tidak mempunyai setiap temuan seed. Untuk menunjukkan sisi penemu kunci, masuk manual sebagai anggota@demo.simpanugm.my.id dengan kata sandi demo.

## Jika tampilan atau hasil berbeda

Gunakan hasil nyata. Jangan memperbaiki cerita di tengah presentasi dengan reset mendadak.

| Gejala                               | Langkah yang dilakukan                                                                                                    |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------- |
| UI lama di port 8000                 | Buka localhost:3001/masuk untuk preview baru, atau bangun ulang Docker dari branch UI sebelum sesi.                       |
| Kartu karakter tidak tampil          | Periksa apakah mode demo diaktifkan saat build/proses web dan apakah make demo sudah menyiapkan akun.                     |
| Draf latihan muncul lagi             | Gunakan Buang draf pada toast saat pemulihan draf. Jangan mengirim draf lama tanpa meninjaunya.                           |
| Saran foto tidak tersedia            | Pilih Coba saran lagi sekali, lalu isi manual. Baca hasil asli; jangan mengganti skor atau kategori dengan hasil buatan.  |
| Kandidat kosong                      | Untuk adegan utama gunakan laporan earbuds seed yang sudah diperiksa. Laporan baru tidak dijamin mempunyai kandidat.      |
| Checklist / serah terima tidak aktif | Periksa apakah kasus punya klaim, statusnya sesuai, barang di pos dan akun berwenang. Pintasan tidak melewati aturan API. |
| Peta dasar kosong                    | Token Mapbox dan internet dibutuhkan. Baca tabel lokasi atau buka Trend Analytics.                                        |
| Kode payung tidak berlaku            | Ambil kode aktif dari Siti. Reset cerita atau regenerasi hanya di luar adegan sesuai kebutuhan, jangan menebak kode.      |

> **Kalimat presenter**
> "AI membantu mencari kandidat dan menjelaskan kemiripan. Skor bukan bukti kepemilikan. Petugas tetap memeriksa bukti dan memastikan serah terima benar."

**Checklist menjelang tampil:** pengisi daya terpasang; cegah laptop tidur; buka URL yang benar; siapkan foto; periksa empat login; baca kondisi payung, kunci dan kalkulator; panaskan saran foto sekali sebelum sesi; tutup build atau suite yang berat. Suite lengkap tidak perlu dijalankan saat presentasi.

## Sumber gambar dan batas demo

Foto dalam tabel berikut tetap mengikuti lisensi masing-masing sumber.

**hero-wallet-found.jpg**
Nafis Fuad Ayon | CC BY-SA 4.0 | [Sumber Commons](https://commons.wikimedia.org/wiki/File:Aarong_leather_wallet.jpg)

**hero-earbuds-found.jpg**
Wkoubnv | CC BY-SA 4.0 | [Sumber Commons](https://commons.wikimedia.org/wiki/File:MS-TW21NV%E3%81%AE%E7%94%BB%E5%83%8F_2.jpg)

**hero-keys-found.jpg**
DARMAS SB 9 | CC BY-SA 4.0 | [Sumber Commons](https://commons.wikimedia.org/wiki/File:Kunci_Kontak_Sepeda_Motor.jpg)

**hero-umbrella-found.jpg**
jam_232 | CC BY 2.0 | [Sumber Commons](<https://commons.wikimedia.org/wiki/File:Functional_%26_practical_(8451672559).jpg>)

**hero-calculator-found.jpg**
R. Henrik Nilsson | CC BY 4.0 | [Sumber Commons](https://commons.wikimedia.org/wiki/File:1980_to_1988_scientific_calculator_Casio_fx_180P_made_in_Japan.jpg)

hero-wallet-lost.jpg dan hero-earbuds-lost.jpg adalah turunan foto sumber yang sama, dengan crop, pencahayaan dan pergeseran perspektif. Kredit pembuat asal serta CC BY-SA 4.0 tetap dicatat. Foto yang terlihat pada screenshot juga berasal dari paket demo tersebut. Pemotongan screenshot hanya untuk tata letak.

**Kredit lengkap:** data/demo/SOURCES.csv dan docs/project/demo-credits.md. Ilustrasi kampus serta empat karakter disediakan pemilik di transit-assets dan dioptimalkan untuk web. Ilustrasi bukan foto tim; tidak ada identitas asli yang dipakai untuk latihan.

**Acuan antarmuka:** commit babf757, branch feat/web-login-demo-guide. Naskah singkat: docs/project/demo-script.md. Pemetaan fitur tersedia di docs/ux/owner-dashboard.md, finder-dashboard.md, officer-dashboard.md dan admin-dashboard.md. Resep pengisian ini merujuk formulir aktual serta cerita pada api/internal/demo/stories.go.

> **Batas interpretasi:** 14/14 foto hero pada pemeriksaan kategori bukan akurasi umum. Metrik monitoring pada data demo bukan hasil evaluasi populasi. Narasi merek dan ciri khusus seed adalah contoh cerita. Hasil inferensi foto, kandidat dan skor tetap ditampilkan asli.

Screenshot diambil dari UI baru pada localhost:3001. Selama pengambilan gambar tidak ada laporan, klaim, penerimaan atau serah terima yang dikirim. Formulir contoh berhenti sebelum tombol konfirmasi; akun dan status cerita utama dipertahankan.
