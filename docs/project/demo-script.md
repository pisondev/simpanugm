# SimpanUGM Live Demo Script and Presentation Kit

Dokumen ini adalah panduan resmi alur demonstrasi langsung (live demo) berdurasi 12 menit untuk evaluasi dosen dan penguji, dilengkapi alokasi menit, urutan klik antarmuka, dan catatan pembicara berbahasa Indonesia.

## 1. Persiapan Awal dan Reset Sistem (Pre-Demo)

Satu perintah untuk menyiapkan lingkungan demo yang bersih, lengkap dengan data master, akun per aktor, skenario hero, embedding model nyata, dan histori klaim:

```bash
make demo
```

Perintah ini bersifat idempoten: dapat dijalankan ulang sebelum demo dimulai untuk memastikan kondisi data kembali ke titik awal yang konsisten.

### Akun dan Kata Sandi Demo

Semua akun demo menggunakan kata sandi standar berikut:

Kata Sandi: `kata sandi panjang 2026`

| Peran | Aktor | Nama | Email | Pos Penugasan | Fokus AI Utama |
|---|---|---|---|---|---|
| Anggota | Penemu | Budi Santoso | anggota.penemu@demo.simpanugm.my.id | - | Asisten pelaporan (AI-03, AI-04), petunjuk pemilik (US-14) |
| Anggota | Pemilik | Siti Rahma | anggota.pemilik@demo.simpanugm.my.id | - | Kandidat cross-modal, penjelasan skor 4 bilah (US-20), klaim terarah |
| Petugas | Pos Perpustakaan | Agus Prasetyo | petugas.pos1@demo.simpanugm.my.id | Pos Klaster Perpustakaan | Antrean prioritas (AI-06), papan tinjauan (AI-07), serah terima kode (US-31) |
| Petugas | Pos Saintek | Dewi Lestari | petugas.pos2@demo.simpanugm.my.id | Pos Klaster Saintek | Manajemen pos dan klaim sekunder |
| Admin | Administrator | Admin SimpanUGM | admin@demo.simpanugm.my.id | Semua Pos | Monitoring drift PSI (AI-11), klaster HDBSCAN (AI-12), peta rawan (AI-10), rollback fusion (US-49) |

---

## 2. Jadwal dan Alokasi Waktu (Total 12 Menit)

| Waktu | Durasi | Bagian | Aktor / Fokus | Pembicara Utama |
|---|---|---|---|---|
| 00:00 - 01:00 | 1.0 mnt | Pembuka & Arsitektur Solusi | Ikhtisar SimpanUGM dan integrasi AI multimodal | Pison |
| 01:00 - 03:30 | 2.5 mnt | Adegan 1: Alur Penemu | Pelaporan foto, asisten AI, masking data pribadi, petunjuk pemilik | Budi (Pison / Tim) |
| 03:30 - 06:00 | 2.5 mnt | Adegan 2: Alur Pemilik | Laporan kehilangan tanpa foto, pencocokan cross-modal, bilah skor, klaim | Siti (Axelle / Tim) |
| 06:00 - 08:30 | 2.5 mnt | Adegan 3: Alur Petugas Pos | Antrean prioritas cerdas, papan tinjauan, checklist AI, verifikasi kode | Agus (Irfan / Tim) |
| 08:30 - 11:00 | 2.5 mnt | Adegan 4: Alur Administrator | Monitoring model aktif, deteksi drift PSI, klaster HDBSCAN, peta rawan | Admin (Pison / Tim) |
| 11:00 - 12:00 | 1.0 mnt | Penutup & Tanya Jawab | Rangkuman keandalan teknis dan sesi interaktif | Tim SimpanUGM |

---

## 3. Naskah Alur dan Catatan Pembicara per Adegan

### Bagian 0: Pembuka (00:00 - 01:00)

- **Layar**: Beranda publik SimpanUGM (`http://localhost:3000`).
- **Tindakan**: Tunjukkan banner mode demo di bagian atas dengan tautan kotak masuk Mailpit.
- **Catatan Pembicara (Indonesia)**:
  "Selamat pagi Bapak/Ibu dosen penguji. Kami mempersembahkan SimpanUGM, platform manajemen kehilangan dan penemuan barang terintegrasi untuk kampus UGM. Sistem yang berjalan saat ini adalah sistem utuh di laptop lokal menggunakan Docker Compose, dengan inferensi AI nyata menggunakan model CLIP untuk visual dan multilingual E5 untuk teks. Kami akan mendemonstrasikan bagaimana AI secara nyata membantu empat peran pengguna: penemu barang, pemilik yang kehilangan, petugas pos pengamanan, dan administrator kampus."

---

### Adegan 1: Penemu Barang / Budi Santoso (01:00 - 03:30)

- **Tujuan**: Memperlihatkan asisten pengisian otomatis dari foto, penyembunyian teks sensitif, dan petunjuk keberadaan pemilik.
- **Masuk**: Halaman `/masuk`, klik tombol cepat "Penemu (Mahasiswa) Budi Santoso".
- **Langkah Navigasi & Klik**:
  1. Klik tombol "Lapor Temuan" di menu navigasi atas (mengarah ke `/lapor/temuan`).
  2. Unggah foto contoh earbuds (`hero-earbuds-found.jpg` atau ambil foto langsung).
  3. Klik "Lanjut". Perhatikan indikator Asisten AI "Menyiapkan saran dari foto...".
  4. Perhatikan bidang Kategori ("Earphone / Headphone"), Warna ("Hitam"), dan Merek ("Jabra") yang terisi otomatis dengan label "Saran otomatis".
  5. Perhatikan bahwa teks publik terhindar dari nomor seri pribadi.
  6. Pilih lokasi penemuan: "Perpustakaan Pusat UGM", lalu pilih Pos Perpustakaan sebagai pos serah terima tujuan.
  7. Klik "Kirim laporan".
  8. Pada halaman detail laporan (`/laporan/temuan/FND-...`), tunjukkan kotak info biru "Kemungkinan pemiliknya sedang mencari" yang menyebutkan adanya mahasiswa lain yang kehilangan barang serupa di lokasi yang sama.
- **Catatan Pembicara (Indonesia)**:
  "Di sini Budi baru saja menemukan earbuds di perpustakaan. Begitu Budi mengunggah foto, model AI kami langsung mengekstraksi fitur visual dalam waktu kurang dari 4 detik. Asisten mengisi kategori, warna, dan merek secara otomatis sebagai draf yang tetap dapat diubah oleh pengguna. Jika ada teks sensitif seperti nomor identitas, asisten secara otomatis menyamarkannya demi privasi. Setelah laporan terkirim, mesin pencocokan di latar belakang langsung mendeteksi bahwa ada laporan kehilangan yang berpasangan, sehingga penemu langsung diberi petunjuk ramah untuk segera menyerahkan barang ke pos terdekat."

---

### Adegan 2: Pemilik Barang / Siti Rahma (03:30 - 06:00)

- **Tujuan**: Memperlihatkan kandidat pencocokan cross-modal (teks ke foto), bilah transparansi 4 dimensi skor, dan pengajuan klaim berbasis pertanyaan kategori.
- **Masuk**: Keluar lalu masuk sebagai "Pemilik (Mahasiswa) Siti Rahma".
- **Langkah Navigasi & Klik**:
  1. Buka menu "Laporan Saya" (`/laporan`).
  2. Klik pada laporan kehilangan dompet kulit hitam ("Dompet Kulit Lipat Eiger Hitam"). Perhatikan laporan ini dibuat tanpa foto.
  3. Gulir ke bagian "Kandidat barang temuan".
  4. Tunjukkan kartu kandidat dengan tier kepercayaan "Sangat mirip" dan persentase skor keseluruhan (misalnya 92%).
  5. Klik kartu kandidat untuk membuka detail pencocokan (`/laporan/hilang/.../kandidat/...`).
  6. Tunjukkan 4 bilah meter penjelasan (Visual Cross-Modal "Kecocokan deskripsi dengan foto", Kemiripan teks, Kedekatan lokasi, Kedekatan waktu) beserta alasan tekstualnya dalam bahasa Indonesia.
  7. Klik tombol "Ajukan Klaim" (`/klaim/baru?pasangan=...`).
  8. Tunjukkan form pertanyaan verifikasi kategori: pertanyaan khusus kategori dompet (isi kartu, warna lapisan dalam) yang wajib dijawab pemohon klaim.
  9. Klik kirim klaim.
- **Catatan Pembicara (Indonesia)**:
  "Sekarang kita beralih ke Siti, pemilik dompet yang hilang di Gelanggang Mahasiswa. Siti melapor tanpa foto karena dompetnya sudah hilang. Di sinilah keunggulan model multimodal SimpanUGM: teks deskripsi Siti dicocokkan secara cross-modal dengan foto barang temuan yang ada di basis data. Siti melihat kandidat dengan transparansi penuh: ada persentase kecocokan dan empat bilah penjelasan yang mudah dipahami. Siti yakin ini miliknya dan mengklik 'Ajukan Klaim', di mana sistem meminta detail verifikasi yang hanya diketahui pemilik sah."

---

### Adegan 3: Petugas Keamanan Pos / Agus Prasetyo (06:00 - 08:30)

- **Tujuan**: Memperlihatkan antrean prioritas AI, papan tinjauan pasangan berdampingan, deteksi duplikat, dan penyelesaian serah terima dengan kode verifikasi 6 digit.
- **Masuk**: Masuk sebagai "Petugas (Pos Perpus) Agus Prasetyo".
- **Langkah Navigasi & Klik**:
  1. Buka beranda petugas (`/petugas`).
  2. Tunjukkan "Antrean Prioritas Tindakan": perhatikan badge skor prioritas gabungan (formula berbobot) dan chip alasan yang memprioritaskan barang dengan klaim aktif atau kemiripan tinggi.
  3. Klik "Papan Tinjauan" (`/petugas/papan`). Tunjukkan perbandingan berdampingan antara laporan hilang dan laporan temuan, lengkap dengan foto, detail tersembunyi, dan skor komponen.
  4. Klik menu "Pendeteksian Duplikat" (`/petugas/duplikat`). Tunjukkan peringatan laporan ganda untuk barang yang sama agar tidak terjadi pencatatan redundan.
  5. Buka daftar klaim masuk (`/petugas/klaim`). Pilih klaim Siti Rahma yang baru masuk.
  6. Tunjukkan checklist verifikasi AI (AI-08) yang mencocokkan jawaban klaim pemohon dengan catatan rahasia penemu.
  7. Petugas menyetujui klaim. Sistem mengunci status dan menerbitkan kode serah terima 6 digit kepada pemilik.
  8. Buka panel serah terima barang, masukkan kode 6 digit dari pemilik dan verifikasi nomor identitas untuk menyelesaikan penyerahan fisik.
- **Catatan Pembicara (Indonesia)**:
  "Petugas keamanan di pos perpustakaan tidak perlu mencari manual di tumpukan buku catatan. Beranda petugas menampilkan Antrean Prioritas yang dihitung secara cerdas oleh sistem berdasarkan urgensi waktu retensi dan skor kecocokan. Petugas dapat membuka Papan Tinjauan untuk membandingkan foto dan detail rahasia secara berdampingan. Ketika memeriksa klaim, asisten AI memberikan saran checklist verifikasi. Penyerahan barang diakhiri secara aman menggunakan kode unik 6 digit berbasis nonce untuk mencegah salah serah."

---

### Adegan 4: Administrator Kampus / Admin SimpanUGM (08:30 - 11:00)

- **Tujuan**: Memperlihatkan pemantauan kesehatan model AI, deteksi drift distribusi skor (PSI), klaster laporan HDBSCAN, peta rawan lonjakan kasus, dan rollback model instan tanpa henti layanan.
- **Masuk**: Masuk sebagai "Administrator Admin SimpanUGM".
- **Langkah Navigasi & Klik**:
  1. Buka menu "Monitoring AI" (`/admin/ai/monitoring`).
  2. Tunjukkan tabel Versi Model Aktif (fusion linear, embedding CLIP, E5 multilingual).
  3. Tunjukkan metrik drift skor (Population Stability Index / PSI) dengan status "Distribusi stabil".
  4. Tunjukkan tabel tingkat penolakan per tier dan tingkat penerimaan saran asisten.
  5. Klik tombol "Kelola & Rollback Model". Tunjukkan dialog pemilihan model kandidat (`fusion-learned`), simulasikan aktivasi dan rollback instan tanpa perlu build ulang kontainer.
  6. Buka menu "Klaster Terkait" (`/admin/ai/klaster`). Tunjukkan pengelompokan laporan otomatis menggunakan algoritma HDBSCAN pada vektor embedding untuk mendeteksi kehilangan beruntun.
  7. Buka menu "Peta Rawan" (`/admin/peta-rawan`). Tunjukkan peta sebaran titik rawan kampus dengan filter 30 hari, banner deteksi lonjakan kasus (surge detection), dan tabel peringkat lokasi yang melindungi privasi pelapor (penyamaran angka di bawah 3 kasus).
  8. Buka menu "Tren Operasional" (`/admin/tren`) untuk visualisasi statistik mingguan dan matriks jam rawan.
- **Catatan Pembicara (Indonesia)**:
  "Bagi administrator kampus, sistem memberikan visibilitas penuh terhadap operasi AI. Di dashboard monitoring, admin dapat mengamati stabilitas model lewat metrik drift skor PSI dan akurasi asisten. Bila tim data science merilis bobot fusion baru, admin dapat mengaktifkan atau melakukan rollback seketika tanpa downtime. Selain itu, algoritma klastering HDBSCAN membantu mendeteksi sindikat atau pola kehilangan beruntun, dan Peta Rawan Kampus memberikan wawasan spasial bagi pimpinan UGM untuk meningkatkan titik patroli keamanan di lokasi rawan secara objektif."

---

### Bagian 5: Penutup dan Tanya Jawab (11:00 - 12:00)

- **Layar**: Ringkasan sistem atau kembali ke beranda.
- **Catatan Pembicara (Indonesia)**:
  "Demikian alur lengkap SimpanUGM: dari penemuan barang, pencocokan cerdas, verifikasi petugas pos, hingga pemantauan analitik kampus. Semua fitur ini didukung oleh model AI nyata yang berjalan efisien di laptop lokal. Kami persilakan Bapak/Ibu dosen penguji untuk mencoba langsung atau mengajukan pertanyaan. Terima kasih."

---

## 4. Paket Rekaman Fallback dan Otomatisasi

Untuk mengantisipasi kendala teknis saat demonstrasi langsung, disediakan rekaman otomatis video dan tangkapan layar beresolusi tinggi menggunakan Playwright:

1. Menjalankan skrip perekaman otomatis:
   ```bash
   npm run test:e2e tests/e2e/demo-record.spec.ts
   ```
2. Lokasi artefak rekaman:
   - Video lengkap: `web/test-results/demo-videos/`
   - Tangkapan layar langkah per langkah: `web/test-results/screenshots/`
