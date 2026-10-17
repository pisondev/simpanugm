# UI Guidelines | SimpanUGM

| | |
|---|---|
| Owner | Pison Golda |
| Implementation | Tailwind CSS 4 tokens in `web/src/styles/tokens.css`, components from shadcn/ui on Radix primitives, icons from Lucide |

## 1. Principles

1. **One next action per screen.** The primary button names exactly what happens next.
2. **Calm under stress.** Owners arrive anxious. Plain words, no alarming colours for normal
   states, progress always visible.
3. **Status in words.** Every status is a label and an icon, colour is a third cue only
   (NFR-USE-05).
4. **Officer speed.** Officer screens favour density, keyboard use and scanning over
   decoration.
5. **No emoji, no stock illustrations.** Icons from Lucide only, inline SVG.

## 2. Brand

The product name is written **SimpanUGM** as a text wordmark. The university logo and
official colours are not used (NFR-LEGAL-02). Favicon and app icon: the Lucide
`package-check` glyph in the primary colour on white.

## 3. Tokens

### 3.1 Colour

Light theme in v1.0. All text pairs meet WCAG AA contrast of 4.5 to 1 on their intended
background.

| Token | Hex | Use |
|---|---|---|
| `--primary` | `#1E40AF` | primary buttons, links, focus ring |
| `--primary-hover` | `#1E3A8A` | |
| `--primary-soft` | `#DBEAFE` | selected chips, info backgrounds |
| `--accent` | `#B45309` | highlights such as the high tier badge outline, used sparingly |
| `--bg` | `#F8FAFC` | page background |
| `--surface` | `#FFFFFF` | cards, sheets |
| `--border` | `#E2E8F0` | |
| `--text` | `#0F172A` | body text |
| `--text-muted` | `#475569` | secondary text |
| `--success` | `#15803D` | returned, approved, handed over |
| `--warning` | `#B45309` | under verification, info requested, retention due |
| `--danger` | `#B91C1C` | rejected, destructive actions, errors |
| `--info` | `#1D4ED8` | potential match |
| `--neutral` | `#64748B` | closed, withdrawn, expired |

### 3.2 Typography

Inter through `next/font`, with system UI as fallback.

| Token | Size and line height | Use |
|---|---|---|
| `text-xs` | 12 / 16 | meta, timestamps |
| `text-sm` | 14 / 20 | officer tables, helper text |
| `text-base` | 16 / 24 | body, inputs. Never below 16 px in inputs, so iOS does not zoom |
| `text-lg` | 18 / 28 | card titles |
| `text-xl` | 20 / 28 | page titles on mobile |
| `text-2xl` | 24 / 32 | page titles on desktop |

### 3.3 Space, shape, elevation

Tailwind's 4 px scale. Page padding 16 px on mobile, 24 px from `md`. Radius 8 px on
inputs and buttons, 12 px on cards and sheets. Two shadows only: `shadow-sm` for cards,
`shadow-lg` for dialogs and sheets.

### 3.4 Breakpoints

`sm` 640, `md` 768, `lg` 1024, `xl` 1280. Member layouts are designed at 360 px first.
Officer layouts are designed at 1366 by 768 and must remain usable at 360 px.

## 4. Status vocabulary

The single source for labels, icons and colours, implemented as one map in
`web/src/lib/status.ts` and covered by a test that every enum value has an entry.

### 4.1 Reports

| Status | Label | Lucide icon | Colour |
|---|---|---|---|
| `reported` | Dilaporkan | `file-text` | neutral |
| `potential_match` | Ada kandidat, Ada kandidat pemilik | `sparkles` | info |
| `under_verification` | Dalam verifikasi | `shield-question` | warning |
| `claimed` | Klaim disetujui | `shield-check` | primary |
| `returned` | Sudah kembali | `circle-check` | success |
| `unclaimed` | Tidak diklaim | `archive` | warning |
| `closed` | Ditutup | `circle-slash` | neutral |

### 4.2 Custody

| Status | Label | Icon |
|---|---|---|
| `with_finder` | Masih dibawa penemu | `hand` |
| `in_custody` | Disimpan di pos | `warehouse` |
| `released` | Sudah diserahkan | `handshake` |
| `disposed` | Sudah ditindaklanjuti | `archive-x` |
| `cancelled` | Dibatalkan | `ban` |

### 4.3 Claims

| Status | Label | Icon | Colour |
|---|---|---|---|
| `submitted` | Menunggu petugas | `inbox` | neutral |
| `under_review` | Sedang diperiksa | `search-check` | warning |
| `info_requested` | Perlu info tambahan | `message-circle-question` | warning |
| `on_hold` | Ditunda | `circle-pause` | neutral |
| `approved` | Disetujui | `badge-check` | success |
| `handed_over` | Sudah diserahkan | `handshake` | success |
| `rejected` | Ditolak | `circle-x` | danger |
| `withdrawn` | Dibatalkan | `undo-2` | neutral |
| `expired` | Kedaluwarsa | `clock-alert` | neutral |

### 4.4 Match tiers

| Tier | Label | Icon | Style |
|---|---|---|---|
| `high` | Sangat mirip | `sparkles` | solid success badge |
| `medium` | Mirip | `sparkle` | outline info badge |
| `low` | Kurang mirip | `circle-dashed` | outline neutral badge, officers only |

## 5. Components

| Need | Component | Notes |
|---|---|---|
| Multi step form | custom `Stepper` on top of shadcn `Form` with React Hook Form and Zod | each step validates on Next, the review step shows all errors with links back |
| Category picker | shadcn `Command` in a `Popover` | suggestions from AI-03 appear as chips above the list |
| Location picker | `Command` with server search, recent picks first | shows zone under each name |
| Photo picker | custom, `input type=file accept=image/* capture=environment` plus gallery | client side resize and HEIC conversion before upload, per image progress |
| Score breakdown | custom `ScoreBars`, four horizontal bars with labels and reason lines | bars carry `aria-valuenow` and the reason as text |
| Side by side review | two column grid from `lg`, stacked tabs below | photo viewer with zoom |
| Tables | TanStack Table on shadcn `Table` | sticky header, keyboard navigation, column for status badges |
| Feedback | `sonner` toasts for success, inline alerts for errors | never a toast for an error the user must act on |
| Confirmations | `AlertDialog` for destructive or irreversible actions: close report, withdraw, reject, approve, handover, dispose | the confirm button repeats the verb, for example "Ya, setujui klaim" |
| Map | `react-leaflet` with OpenStreetMap tiles and `leaflet.heat` | attribution visible, campus bounds fixed |
| Charts | Recharts | palette from the tokens, every chart has a table alternative |
| Scanner | `@zxing/browser` for QR on the receive screen | falls back to typing the code |

## 6. Copy

### 6.1 Voice

| Audience | Voice | Example |
|---|---|---|
| Members | friendly, "kamu", short sentences | "Laporanmu sudah terkirim. Kami sedang mencari barang yang cocok." |
| Officers and admins | neutral, no pronoun where possible, imperative for actions | "Periksa kecocokan ciri sebelum menyetujui." |

### 6.2 Vocabulary

Use these words consistently. The glossary has the English equivalents.

| Use | Avoid |
|---|---|
| laporan kehilangan | laporan barang hilang, lost report |
| laporan temuan | laporan penemuan, found report |
| kandidat | match, rekomendasi |
| klaim | pengajuan kepemilikan |
| serah terima | pengambilan, handover |
| pos | posko, Lost and Found desk |
| ciri khusus | detail rahasia |
| kode serah terima | PIN, OTP |

### 6.3 Formats

| Item | Format | Implementation |
|---|---|---|
| Date and time | "Senin, 2 Nov 2026, 14.30 WIB" | `Intl.DateTimeFormat('id-ID', { timeZone: 'Asia/Jakarta' })`, WIB suffix added |
| Relative time | "5 menit lalu", "kemarin" | `Intl.RelativeTimeFormat('id-ID')` under 7 days, absolute beyond |
| Score | "81%" | rounded to whole percent |
| Distance | "sekitar 150 m", "1,2 km" | rounded to 50 m, Indonesian decimal comma |
| Codes | `LST-2611-0007` in a monospace font | copy button next to it |

### 6.4 Messages

| Situation | Message |
|---|---|
| Required field | "Wajib diisi" |
| Too short | "Minimal 20 karakter, saat ini 12" |
| Upload too large | "Ukuran foto maksimal 10 MB" |
| Network error | "Gagal terhubung. Coba lagi." with a retry button |
| Server error | "Terjadi kesalahan di sisi kami. Kode: {request_id}" |
| Rate limited | "Terlalu banyak percobaan. Coba lagi dalam {n} menit." |
| Assistant down | "Saran otomatis sedang tidak tersedia" |

## 7. Accessibility

- WCAG 2.1 AA as the baseline (NFR-USE-03), checked by axe in every end to end test.
- Every interactive element reachable and operable by keyboard, with a visible 2 px focus
  ring in `--primary`.
- Touch targets at least 44 by 44 px.
- Images of items have alternative text from the item name and category, for example
  "Foto barang temuan: botol minum biru".
- Form errors are announced with `aria-live="polite"` and linked with `aria-describedby`.
- Motion limited to 200 ms transitions and disabled under `prefers-reduced-motion`.
- No information by colour alone: every badge has a label, every chart a table.
