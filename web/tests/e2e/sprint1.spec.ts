import { expect, test, type Browser, type Page } from "@playwright/test"

import { bootstrapAdmin, expectAccessible, attachPhoto, latestLink, registerMember, setPassword, signIn, uniqueEmail } from "./helpers"

// release v0.1.0 exit of project-plan Sprint 1: staff are invited, a member files both report types with a photo,
// an officer receives the item into custody and finds reports by code and by text. J2 continues with the same accounts
// because the suite shares one address and BR-SEC-01 allows it five registrations an hour
test.describe.configure({ mode: "serial" })

const password = "kata sandi panjang 2026"
const adminEmail = uniqueEmail("admin", "admin.example.id")
const officerEmail = uniqueEmail("petugas", "petugas.example.id")
const memberEmail = uniqueEmail("anggota")
let lostCode = ""
let foundCode = ""
// a stack that already ran the suite keeps older items, a name of this run picks ours among them
const handedIn = `Tumbler biru ${Date.now() % 100000}`

async function fresh(browser: Browser): Promise<Page> {
  const context = await browser.newContext()
  return context.newPage()
}

async function pickCategory(page: Page, search: string, option: string) {
  await page.getByRole("combobox", { name: "Kategori" }).click()
  await page.getByPlaceholder("Cari kategori, misalnya dompet").fill(search)
  await page.getByRole("option", { name: option, exact: true }).click()
}

async function pickPlace(page: Page, label: string, search: string, option: RegExp) {
  await page.getByLabel(label, { exact: true }).click()
  await page.getByPlaceholder("Cari gedung atau tempat, misalnya perpus").fill(search)
  await page.getByRole("option", { name: option }).first().click()
}

test("an administrator from the command line invites an officer to a post", async ({ browser }) => {
  const admin = await fresh(browser)
  await setPassword(admin, bootstrapAdmin(adminEmail, "Admin Uji"), password, "Aktifkan akun")
  await expect(admin).toHaveURL(/\/admin$/)
  await expectAccessible(admin)

  await admin.goto("/admin/pengguna")
  await admin.getByRole("button", { name: "Undang pengguna" }).click()
  const dialog = admin.getByRole("dialog")
  await dialog.getByLabel("Email").fill(officerEmail)
  await dialog.getByLabel("Nama lengkap").fill("Petugas Uji")
  await dialog.getByText("Pos Lost and Found Perpustakaan Pusat").click()
  await dialog.getByRole("button", { name: "Kirim undangan" }).click()
  await expect(admin.getByText(`Undangan dikirim ke ${officerEmail}.`)).toBeVisible()

  const officer = await fresh(browser)
  await setPassword(officer, await latestLink(officerEmail, "/undangan"), password, "Aktifkan akun")
  await expect(officer).toHaveURL(/\/petugas$/)
  await expectAccessible(officer)
})

test("a member files a lost report without a photo", async ({ browser }) => {
  const page = await fresh(browser)
  await registerMember(page, "Rani Prasetya", memberEmail, password)
  await signIn(page, memberEmail, password)
  await expect(page).toHaveURL(/\/beranda$/)

  await page.goto("/lapor/hilang")
  await page.getByRole("button", { name: "Lewati, saya tidak punya foto" }).click()
  await pickCategory(page, "tumbler", "Tumbler")
  await page.getByLabel("Nama barang").fill("Tumbler biru dongker")
  await page.getByLabel("Deskripsi").fill("Tumbler 500 ml warna biru dongker, tutup hitam, ada stiker kucing di sisi.")
  await page.getByRole("button", { name: "Lanjut" }).click()
  await pickPlace(page, "Tempat terakhir terlihat", "perpus", /Perpustakaan Pusat UGM/)
  await page.getByRole("button", { name: "Lanjut" }).click()
  await page.getByLabel("Ciri khusus").fill("Goresan kecil di bawah tutup")
  await page.getByRole("button", { name: "Lanjut" }).click()
  await expectAccessible(page)
  await page.getByRole("button", { name: "Kirim laporan" }).click()

  await expect(page).toHaveURL(/\/laporan\/hilang\/LST-\d{4}-\d{4}$/)
  lostCode = page.url().split("/").pop() ?? ""
  await expect(page.getByRole("heading", { name: "Tumbler biru dongker" })).toBeVisible()
  await expectAccessible(page)
})

test("the member files a found report with a photo and is told where to hand it in", async ({ browser }) => {
  const page = await fresh(browser)
  await signIn(page, memberEmail, password)
  await expect(page).toHaveURL(/\/beranda$/)

  await page.goto("/lapor/temuan")
  await attachPhoto(page)
  await expect(page.getByText(/Mengunggah|Menyiapkan/)).toHaveCount(0, { timeout: 20_000 })
  await page.getByRole("button", { name: "Lanjut" }).click()
  // J3: the assistant looks at the photo and either fills the form or says it cannot, the form works both ways
  await expect(page.getByText("Saran otomatis", { exact: false }).first()).toBeVisible({ timeout: 30_000 })
  await pickCategory(page, "payung", "Payung")
  await page.getByLabel("Nama barang").fill("Payung lipat hitam")
  await page.getByLabel("Deskripsi").fill("Payung lipat hitam dengan gagang kayu")
  await page.getByRole("button", { name: "Lanjut" }).click()
  await pickPlace(page, "Tempat ditemukan", "perpus", /Perpustakaan Pusat UGM/)
  await page.getByRole("button", { name: "Lanjut" }).click()
  await expect(page.getByText("Terdekat")).toBeVisible()
  await page.getByRole("button", { name: "Lanjut" }).click()
  await page.getByRole("button", { name: "Kirim laporan" }).click()

  await expect(page).toHaveURL(/\/laporan\/temuan\/FND-\d{4}-\d{4}$/)
  foundCode = page.url().split("/").pop() ?? ""
  await expect(page.getByRole("heading", { name: "Serahkan barang ke pos" })).toBeVisible()
  await expectAccessible(page)
})

test("the officer receives the item and finds both reports", async ({ browser }) => {
  const page = await fresh(browser)
  await signIn(page, officerEmail, password)
  await expect(page).toHaveURL(/\/petugas$/)

  await page.goto(`/petugas/terima?code=${foundCode}`)
  await expect(page.getByRole("heading", { name: "Payung lipat hitam" })).toBeVisible()
  await page.getByLabel("Rak atau slot").fill("Rak A1")
  await page.getByRole("button", { name: "Terima barang" }).click()
  await page.getByRole("alertdialog").getByRole("button", { name: "Ya, terima barang" }).click()
  await expect(page.getByRole("status").filter({ hasText: "Barang diterima di" })).toBeVisible()
  await expectAccessible(page)

  await page.goto("/petugas/inventaris")
  await expect(page.getByRole("cell", { name: new RegExp(`Payung lipat hitam ${foundCode}`) })).toBeVisible()

  await page.goto("/petugas/cari")
  await page.getByRole("searchbox").fill(lostCode)
  await page.getByRole("button", { name: "Cari" }).click()
  await expect(page).toHaveURL(new RegExp(`/petugas/laporan/hilang/${lostCode}$`))
  await expect(page.getByText("Goresan kecil di bawah tutup")).toBeVisible()

  await page.goto("/petugas/cari?q=tumblr+biru")
  await expect(page.getByRole("link", { name: new RegExp(lostCode) })).toBeVisible()
  await expectAccessible(page)
})

// J2: the lost tumbler without a photo meets a matching item handed in at the desk (US-19, US-20, US-21)
test("J2 an officer records a matching item handed in at the desk and sees the candidate", async ({ browser }) => {
  const page = await fresh(browser)
  await signIn(page, officerEmail, password)
  await expect(page).toHaveURL(/\/petugas$/)
  await page.goto("/petugas/lapor-temuan")
  await attachPhoto(page)
  await expect(page.getByText(/Mengunggah|Menyiapkan/)).toHaveCount(0, { timeout: 20_000 })
  await page.getByRole("button", { name: "Lanjut" }).click()
  await pickCategory(page, "tumbler", "Tumbler")
  await page.getByLabel("Nama barang").fill(handedIn)
  await page.getByLabel("Deskripsi").fill("Tumbler biru tua dengan tutup hitam, diserahkan pengunjung di meja pos.")
  await page.getByRole("button", { name: "Lanjut" }).click()
  await pickPlace(page, "Tempat ditemukan", "perpus", /Perpustakaan Pusat UGM/)
  await page.getByRole("button", { name: "Lanjut" }).click()
  await page.getByText("Pos Lost and Found Perpustakaan Pusat").click()
  await page.getByLabel("Rak atau slot penyimpanan").fill("Rak B2")
  await page.getByRole("button", { name: "Lanjut" }).click()
  await page.getByRole("button", { name: "Kirim laporan" }).click()
  await expect(page).toHaveURL(/\/petugas\/laporan\/temuan\/FND-\d{4}-\d{4}$/)

  // scoring runs in the worker, the table fills once it has
  await expect(async () => {
    await page.reload()
    await expect(page.getByRole("link", { name: new RegExp(lostCode) })).toBeVisible({ timeout: 1_000 })
  }).toPass({ timeout: 90_000, intervals: [2_000] })
  await expect(page.getByRole("heading", { name: "Kandidat pencocokan" })).toBeVisible()
  await expectAccessible(page)
})

test("J2 the owner sees the candidate, reads why, and hides it", async ({ browser }) => {
  const page = await fresh(browser)
  await signIn(page, memberEmail, password)
  await expect(page).toHaveURL(/\/beranda$/)
  await page.goto(`/laporan/hilang/${lostCode}`)
  const candidate = page.getByRole("link", { name: new RegExp(handedIn) })
  await expect(candidate).toBeVisible()
  await expect(page.getByText(/\d+ kandidat ditemukan/)).toBeVisible()
  await expectAccessible(page)

  await candidate.click()
  await expect(page.getByRole("heading", { name: handedIn, exact: true })).toBeVisible()
  await expect(page.getByRole("heading", { name: "Kenapa kandidat ini muncul" })).toBeVisible()
  await expect(page.getByRole("meter")).toHaveCount(4)
  await expect(page.getByText("Ditemukan di lokasi yang sama.")).toBeVisible()
  await expect(page.getByText("Kategori sama.")).toBeVisible()
  await expectAccessible(page)

  await page.getByRole("button", { name: "Bukan milik saya" }).click()
  const dialog = page.getByRole("alertdialog")
  await dialog.getByLabel("Alasan (opsional)").fill("Punya saya ada stiker kucing")
  await dialog.getByRole("button", { name: "Ya, sembunyikan" }).click()
  await expect(page).toHaveURL(new RegExp(`/laporan/hilang/${lostCode}$`))
  await expect(page.getByText("Kandidat disembunyikan.")).toBeVisible()
  await expect(page.getByRole("link", { name: new RegExp(handedIn) })).toHaveCount(0)
})

// US-07: the master data pages are reachable from the admin navigation and pass the accessibility check
test("the administrator reviews master data", async ({ browser }) => {
  const page = await fresh(browser)
  await signIn(page, adminEmail, password)
  await expect(page).toHaveURL(/\/admin$/)
  await page.getByRole("link", { name: "Data Master" }).first().click()
  await expect(page).toHaveURL(/\/admin\/master\/kategori$/)
  await expect(page.getByRole("heading", { name: "Kompatibilitas kategori" })).toBeVisible()
  await expectAccessible(page)
  for (const [tab, path] of [
    ["Lokasi", "lokasi"],
    ["Zona", "zona"],
    ["Pos", "pos"],
  ] as const) {
    await page.getByRole("link", { name: tab, exact: true }).click()
    await expect(page).toHaveURL(new RegExp(`/admin/master/${path}$`))
    await expectAccessible(page)
  }
  await page.getByRole("button", { name: /^Ubah Pos Lost and Found Perpustakaan Pusat/ }).click()
  await expect(page.getByRole("dialog").getByLabel("Senin")).toHaveValue(/\d{2}:\d{2}-\d{2}:\d{2}/)
  await expectAccessible(page)
  await page.getByRole("dialog").getByRole("button", { name: "Batal" }).click()
})
