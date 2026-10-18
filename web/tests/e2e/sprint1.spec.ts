import { expect, test, type Browser, type Page } from "@playwright/test"

import { bootstrapAdmin, expectAccessible, latestLink, photo, registerMember, setPassword, signIn, uniqueEmail } from "./helpers"

// release v0.1.0 exit of project-plan Sprint 1: staff are invited, a member files both report types with a photo,
// an officer receives the item into custody and finds reports by code and by text
test.describe.configure({ mode: "serial" })

const password = "kata sandi panjang 2026"
const officerEmail = uniqueEmail("petugas", "petugas.example.id")
const memberEmail = uniqueEmail("anggota")
let lostCode = ""
let foundCode = ""

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
  await setPassword(admin, bootstrapAdmin(uniqueEmail("admin", "admin.example.id"), "Admin Uji"), password, "Aktifkan akun")
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
  await page.locator('input[type="file"][multiple]').setInputFiles(await photo(page))
  await expect(page.getByRole("button", { name: "Hapus foto 1" })).toBeVisible()
  await expect(page.getByText(/Mengunggah|Menyiapkan/)).toHaveCount(0, { timeout: 20_000 })
  await page.getByRole("button", { name: "Lanjut" }).click()
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
  await expect(page.getByRole("cell", { name: /Payung lipat hitam/ })).toBeVisible()

  await page.goto("/petugas/cari")
  await page.getByRole("searchbox").fill(lostCode)
  await page.getByRole("button", { name: "Cari" }).click()
  await expect(page).toHaveURL(new RegExp(`/petugas/laporan/hilang/${lostCode}$`))
  await expect(page.getByText("Goresan kecil di bawah tutup")).toBeVisible()

  await page.goto("/petugas/cari?q=tumblr+biru")
  await expect(page.getByRole("link", { name: new RegExp(lostCode) })).toBeVisible()
  await expectAccessible(page)
})
