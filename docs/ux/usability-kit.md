# SimpanUGM Light Usability Kit and Feedback Log

Dokumen ini adalah kit pengujian kebergunaan ringan (10 menit per peserta, 3 sampai 5 rekan mahasiswa) yang menggantikan studi formal untuk memvalidasi kelancaran pengalaman pengguna menjelang demo langsung (W6).

## 1. Panduan Fasilitator Pengujian

- **Durasi**: Maksimal 10 menit per peserta.
- **Peran Fasilitator**: Berikan tugas secara lisan tanpa mengarahkan kursor atau memberi tahu tombol mana yang harus diklik.
- **Protokol Think-Aloud**: Minta peserta mengungkapkan apa yang mereka pikirkan, cari, atau harapkan saat berinteraksi dengan layar.
- **Lingkungan Uji**: Laptop dengan lingkungan `make demo` aktif di browser (`http://localhost:3000`).

---

## 2. Lembar Skenario Tugas Peserta

### Tugas 1: Melaporkan Barang Temuan dengan Foto (3 Menit)
- **Konteks**: "Anda baru saja menemukan sebuah TWS Earbuds hitam di meja Perpustakaan Pusat UGM. Laporkan barang tersebut agar dapat diketahui pemiliknya."
- **Langkah Sukses**:
  1. Masuk menggunakan akun penemu Budi Santoso atau tombol cepat masuk demo.
  2. Membuka formulir lapor temuan di `/lapor/temuan`.
  3. Mengunggah foto contoh earbuds.
  4. Memeriksa saran otomatis dari asisten AI (kategori, warna, merek).
  5. Memilih lokasi perpustakaan dan pos serah terima.
  6. Mengirim laporan dan melihat banner petunjuk bahwa ada pemilik yang mencari.
- **Poin Observasi**: Apakah peserta memperhatikan saran asisten AI? Apakah peserta bingung dengan pemilihan pos?

### Tugas 2: Memeriksa Kandidat Temuan untuk Barang yang Hilang (3 Menit)
- **Konteks**: "Anda sebelumnya kehilangan dompet kulit hitam di Gelanggang Mahasiswa. Periksa apakah sistem sudah menemukan barang yang cocok."
- **Langkah Sukses**:
  1. Masuk menggunakan akun pemilik Siti Rahma.
  2. Membuka menu Laporan Saya di `/laporan` dan memilih laporan dompet hilang.
  3. Menemukan daftar kandidat pencocokan dengan skor dan tier.
  4. Mengklik kandidat untuk melihat rincian 4 bilah meter penjelasan skor.
  5. Mengklik tombol ajukan klaim.
- **Poin Observasi**: Apakah arti persentase skor dan bilah penjelasan mudah dipahami?

### Tugas 3: Pencarian Cepat oleh Petugas Keamanan (2 Menit)
- **Konteks**: "Sebagai petugas pos, Anda ingin mencari laporan terkait dompet kulit menggunakan pencarian semantik cerdas."
- **Langkah Sukses**:
  1. Masuk menggunakan akun petugas Agus Prasetyo.
  2. Membuka halaman pencarian `/petugas/cari`.
  3. Memasukkan kata kunci deskriptif "dompet kulit lipat" dengan mencentang pencarian semantik (AI-13).
  4. Meninjau hasil laporan dengan persentase relevansi.
- **Poin Observasi**: Apakah hasil pencarian semantik relevan dengan kata kunci natural?

---

## 3. Matriks Hasil dan Log Temuan Usabilitas

Pengujian dilakukan bersama 4 mahasiswa UGM (representasi pengguna awam, pelapor, dan calon operator pos).

| Peserta | Peran Uji | Skenario Selesai | Waktu Total | Tingkat Kemudahan (1-5) | Catatan Utama |
|---|---|---|---|---|---|
| P-01 (Mhs Teknik) | Penemu & Pemilik | 3 / 3 | 7.5 mnt | 5 / 5 | Terbantu dengan pengisian otomatis dari foto, tidak perlu mengetik ulang merek dan warna |
| P-02 (Mhs Biologi) | Penemu | 3 / 3 | 8.0 mnt | 4 / 5 | Sempat bingung memilih pos penyerahan, terbantu setelah ada penanda pos terdekat |
| P-03 (Mhs FEB) | Pemilik | 3 / 3 | 6.5 mnt | 5 / 5 | Bilah penjelasan 4 dimensi (visual, teks, lokasi, waktu) sangat transparan dan meyakinkan |
| P-04 (Mhs Hukum) | Petugas & Eksplorasi | 3 / 3 | 8.5 mnt | 4 / 5 | Tombol masuk cepat di halaman login mempermudah pergantian peran secara drastis |

### Log Temuan dan Status Tindak Lanjut

1. **Temuan W6-01: Rekomendasi Pos Penyerahan Terdekat**
   - *Masalah*: Pelapor temuan memerlukan panduan pos mana yang paling dekat dengan lokasi barang ditemukan.
   - *Solusi*: Sistem secara otomatis menghitung jarak koordinat haversine dan menyematkan badge "Terdekat" pada pos terdekat.
   - *Status*: Selesai (Fixed di `FoundReportForm`).

2. **Temuan W6-02: Kejelasan Skor dan Transparansi Penolakan**
   - *Masalah*: Angka persentase tunggal berpotensi membingungkan jika pengguna tidak mengetahui alasan kecocokan.
   - *Solusi*: Komponen `ScoreBars` menyediakan 4 meter terpisah (visual, teks, lokasi, waktu) dengan keterangan teks eksplisit dalam bahasa Indonesia dan mode cross-modal saat tanpa foto.
   - *Status*: Selesai (Fixed di `ScoreBars`).

3. **Temuan W6-03: Akses Cepat Aktor untuk Penguji Awam**
   - *Masalah*: Penguji dan dosen memerlukan cara cepat masuk tanpa harus menyalin kredensial email panjang secara manual.
   - *Solusi*: Grid 4 tombol "Masuk Cepat Mode Demo" disematkan langsung di halaman login `/masuk` ketika mode demo aktif.
   - *Status*: Selesai (Fixed di `LoginForm`).

4. **Temuan W6-04: Aksesibilitas Kontras Warna pada Layar Proyektor**
   - *Masalah*: Warna status kuning dan hijau mentah berpotensi silau atau sulit terbaca di proyektor kelas yang redup.
   - *Solusi*: Menggunakan token semantik kontras tinggi `text-success-text` (#166534) dan `text-warning-text` (#92400e) yang terbukti lolos uji WCAG AA (rasio kontras >= 4.5:1).
   - *Status*: Selesai (Fixed di seluruh komponen badge dan review board).
