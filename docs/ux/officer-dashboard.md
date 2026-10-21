# Dashboard petugas

Referensi: slide 6, "Fitur untuk Petugas", pada
[presentasi SimpanUGM](https://docs.google.com/presentation/d/1GJvs2N0-C1RR6-5N7AFZt-phUa9ahCwaljLQA13qhes/edit#slide=id.h37bd994acf507109_0_25).

| Nama fitur pada slide dan antarmuka | Alur yang digunakan                                                                                                        |
| ----------------------------------- | -------------------------------------------------------------------------------------------------------------------------- |
| AI-Assisted Priority Queue          | Jumlah antrean dan tugas teratas dari `/officer/queue`, dengan tautan ke antrean lengkap yang tetap berada di halaman ini. |
| AI Match Review Board               | Membuka laporan temuan yang dipilih pada `/petugas/papan` untuk membandingkan kandidat dan alasan kecocokan.               |
| Semantic Verification Checklist     | Membuka checklist klaim pada kasus terpilih, atau menjelaskan prasyarat ketika kasus belum memiliki klaim.                 |
| Semantic Search                     | Formulir bahasa alami menuju pencarian petugas dengan `semantic=true`, memakai mesin pencarian yang sudah ada.             |
| Duplicate Review                    | Membuka pemeriksaan dan penggabungan laporan serupa; tidak menambahkan angka duplikat buatan.                              |
| Handover to Owner                   | Membuka klaim disetujui yang barangnya sudah di pos, atau daftar klaim disetujui bila belum ada yang siap.                 |

Seluruh fitur tersedia sebelum perubahan ini. Dashboard menyatukan pintasannya
dengan nama eksplisit yang sama pada halaman tujuan. Lima panel bantuan AI diberi
label AI Feature; serah terima tetap proses verifikasi petugas. Prioritas merupakan
gabungan urgensi dan sinyal kecocokan, bukan probabilitas kepemilikan. Checklist
semantik memakai skor pasangan tersimpan bila tersedia dan pemeriksaan manual bila
tidak tersedia. Tidak ada model atau keluaran AI buatan yang ditambahkan.

Saat masuk sebagai petugas, `/petugas` memakai header navigasi atas, ilustrasi
petugas di kiri pada desktop dan enam panel berwarna. Menu Lainnya tetap membuka
seluruh navigasi petugas dan, untuk admin, navigasi administrasi. Halaman kerja lain
mempertahankan sidebar sebelumnya. Navigasi bawah ponsel dan aturan global
`prefers-reduced-motion` tetap berlaku.

Pemilih kasus berisi laporan unik dari maksimal 30 tugas yang dimuat. Pilihan ini
menentukan papan dan checklist. Antrean teratas tetap menunjukkan prioritas tertinggi
pada cakupan dan filter aktif. Cakupan awal adalah pos yang ditugaskan; memilih
seluruh kampus tidak memperluas kewenangan tindakan API.

Pintasan serah terima diperiksa terpisah dari kasus terpilih: dari lima klaim
disetujui terbaru yang dimuat, dipilih klaim pertama dengan barang `in_custody`.
Nama barang dan kode klaim ditampilkan agar petugas mengetahui kasus yang dibuka.
Daftar klaim lengkap tetap dapat dibuka. Kode penyerahan, identitas, status dan
kewenangan tetap diverifikasi oleh formulir dan API yang sudah ada.

Ilustrasi berasal dari `3D UGM Administrator with Clipboard.png` dalam
`transit-assets` milik pemilik proyek, dioptimalkan menjadi WebP transparan
63.806 byte. Header kampus dan warna memakai aset dashboard sebelumnya.
Tidak ada dependensi, endpoint atau perubahan workflow deployment.

Pemeriksaan dibatasi pada dua skenario `demo-officer-dashboard.spec.ts`:
kesesuaian data antrean, desktop, ponsel, axe, pemilih kasus, cakupan dan navigasi
enam fitur. Pengujian hanya membaca atau membuka formulir, tanpa menyetujui klaim,
menggabungkan laporan atau menghabiskan kode penyerahan. Spec demo hanya berjalan
dengan `DEMO_E2E=1`. Suite lengkap dan tes backend tidak dijalankan ulang untuk perubahan ini.

Hasil pemeriksaan lokal 21 Oktober 2026:

- Satu run gabungan dua skenario dashboard petugas: 2 lulus, 0 gagal, 0 dilewati.
- Axe pada dashboard desktop dan ponsel: 0 pelanggaran.
- Prettier pada kode web, spec baru dan catatan aset, typecheck, lint dan build web: lulus.
- Pemeriksa aturan proyek dan `git diff --check`: bersih.
