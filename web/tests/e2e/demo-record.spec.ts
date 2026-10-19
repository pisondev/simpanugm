import { expect, test, type Page } from "@playwright/test"

import { attachPhoto, expectAccessible, signIn } from "./helpers"

test.describe.configure({ mode: "serial" })

test.use({
  video: "on",
  viewport: { width: 1280, height: 720 },
})

const password = "kata sandi panjang 2026"
const finderEmail = "anggota.penemu@demo.simpanugm.my.id"
const ownerEmail = "anggota.pemilik@demo.simpanugm.my.id"
const officerEmail = "petugas.pos1@demo.simpanugm.my.id"
const adminEmail = "admin@demo.simpanugm.my.id"

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

test.describe("Demo Fallback Video and Screenshot Recording (W5)", () => {
  test("Record Scene 1: Finder reports earbuds with AI assistant and finder hint", async ({ page }) => {
    await signIn(page, finderEmail, password)
    await expect(page).toHaveURL(/\/beranda$/)

    await page.goto("/lapor/temuan")
    await attachPhoto(page)
    await expect(page.getByText(/Mengunggah|Menyiapkan/)).toHaveCount(0, { timeout: 20_000 })
    await page.getByRole("button", { name: "Lanjut" }).click()

    await expect(page.getByText(/Saran otomatis|tidak tersedia/).first()).toBeVisible({ timeout: 30_000 })
    await pickCategory(page, "earphone", "Earphone / Headphone")
    await page.getByLabel("Nama barang").fill("TWS Earbuds Hitam Demo")
    await page.getByLabel("Deskripsi").fill("Earbuds warna hitam di meja belajar")
    await page.getByRole("button", { name: "Lanjut" }).click()

    await pickPlace(page, "Tempat ditemukan", "perpus", /Perpustakaan Pusat UGM/)
    await page.getByRole("button", { name: "Lanjut" }).click()

    await expect(page.getByText("Terdekat")).toBeVisible()
    await page.getByRole("button", { name: "Lanjut" }).click()
    await page.getByRole("button", { name: "Kirim laporan" }).click()

    await expect(page).toHaveURL(/\/laporan\/temuan\/FND-\d{4}-\d{4}$/)
    await page.screenshot({ path: "test-results/demo-record/scene1-finder-hint.png", fullPage: true })
    await expectAccessible(page)
  })

  test("Record Scene 2: Owner reviews candidates and submits category claim", async ({ page }) => {
    await signIn(page, ownerEmail, password)
    await expect(page).toHaveURL(/\/beranda$/)

    await page.goto("/laporan")
    await expect(page.getByRole("heading", { name: "Laporan Saya" })).toBeVisible()

    const reportLink = page.getByRole("link", { name: /Dompet|Earbuds|Kunci/i }).first()
    if (await reportLink.isVisible()) {
      await reportLink.click()
      await expect(page).toHaveURL(/\/laporan\/hilang\/LST-/)

      const candidatesSection = page.getByRole("heading", { name: "Kandidat barang temuan" })
      if (await candidatesSection.isVisible()) {
        const firstCandidate = page.locator("#kandidat ol li a").first()
        if (await firstCandidate.isVisible()) {
          await firstCandidate.click()
          await expect(page).toHaveURL(/\/laporan\/hilang\/.*\/kandidat\//)

          await expect(page.getByRole("meter")).toHaveCount(4)
          await expect(page.getByRole("heading", { name: "Kenapa kandidat ini muncul" })).toBeVisible()
          await page.screenshot({ path: "test-results/demo-record/scene2-owner-score-bars.png", fullPage: true })
          await expectAccessible(page)
        }
      }
    }
  })

  test("Record Scene 3: Officer manages priority queue, review board, and duplicates", async ({ page }) => {
    await signIn(page, officerEmail, password)
    await expect(page).toHaveURL(/\/petugas$/)

    await expect(page.getByRole("heading", { name: "Antrean Prioritas Tindakan" })).toBeVisible()
    await page.screenshot({ path: "test-results/demo-record/scene3-officer-queue.png", fullPage: true })
    await expectAccessible(page)

    await page.goto("/petugas/papan")
    await expect(page.getByRole("heading", { name: "Papan Tinjauan Pasangan" })).toBeVisible()
    await page.screenshot({ path: "test-results/demo-record/scene3-officer-board.png", fullPage: true })
    await expectAccessible(page)

    await page.goto("/petugas/duplikat")
    await expect(page.getByRole("heading", { name: "Pendeteksian Duplikat" })).toBeVisible()
    await page.screenshot({ path: "test-results/demo-record/scene3-officer-duplicates.png", fullPage: true })
    await expectAccessible(page)
  })

  test("Record Scene 4: Admin reviews monitoring drift, clusters, and hotspot map", async ({ page }) => {
    await signIn(page, adminEmail, password)
    await expect(page).toHaveURL(/\/admin$/)

    await page.goto("/admin/ai/monitoring")
    await expect(page.getByRole("heading", { name: "Monitoring AI & Mesin Pencocokan" })).toBeVisible()
    await page.screenshot({ path: "test-results/demo-record/scene4-admin-monitoring.png", fullPage: true })
    await expectAccessible(page)

    await page.goto("/admin/ai/klaster")
    await expect(page.getByRole("heading", { name: /Klaster Laporan Terkait/ })).toBeVisible()
    await page.screenshot({ path: "test-results/demo-record/scene4-admin-clusters.png", fullPage: true })
    await expectAccessible(page)

    await page.goto("/admin/peta-rawan")
    await expect(page.getByRole("heading", { name: /Peta Rawan/ })).toBeVisible()
    await page.screenshot({ path: "test-results/demo-record/scene4-admin-hotspots.png", fullPage: true })
    await expectAccessible(page)
  })
})
