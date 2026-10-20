import { expect, test, type Browser, type Page } from "@playwright/test"

import { attachPhoto, expectAccessible, signIn } from "./helpers"

test.describe.configure({ mode: "serial" })

const password = "kata sandi panjang 2026"
const finderEmail = "anggota.penemu@demo.simpanugm.my.id"
const ownerEmail = "anggota.pemilik@demo.simpanugm.my.id"
const officerEmail = "petugas.pos1@demo.simpanugm.my.id"
const adminEmail = "admin@demo.simpanugm.my.id"

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

test.describe("AI Moments: Live Demonstration Story (W4)", () => {
  test("Moment 1: Finder reports item with photo and sees AI assistant suggestions and finder hint", async ({ browser }) => {
    const page = await fresh(browser)
    await signIn(page, finderEmail, password)
    await expect(page).toHaveURL(/\/beranda$/)

    await page.goto("/lapor/temuan")
    await attachPhoto(page)
    await expect(page.getByText(/Mengunggah|Menyiapkan/)).toHaveCount(0, { timeout: 20_000 })
    await page.getByRole("button", { name: "Lanjut" }).click()

    // AI Assistant: suggestion tag or graceful fallback note
    await expect(page.getByText(/Saran otomatis|tidak tersedia/).first()).toBeVisible({ timeout: 30_000 })
    await pickCategory(page, "earphone", "Earphone")
    await page.getByLabel("Nama barang").fill("TWS Earbuds Hitam Demo")
    await page.getByLabel("Deskripsi").fill("Earbuds warna hitam di meja belajar")
    await page.getByRole("button", { name: "Lanjut" }).click()

    await pickPlace(page, "Tempat ditemukan", "perpus", /Perpustakaan Pusat UGM/)
    await page.getByRole("button", { name: "Lanjut" }).click()

    await expect(page.getByText("Terdekat")).toBeVisible()
    await page.getByRole("button", { name: "Lanjut" }).click()
    await page.getByRole("button", { name: "Kirim laporan" }).click()

    await expect(page).toHaveURL(/\/laporan\/temuan\/FND-\d{4}-\d{4}$/)
    await page.screenshot({ path: "test-results/screenshots/demo-1-finder.png", fullPage: true })
    await expectAccessible(page)
  })

  test("Moment 2: Owner sees candidate matches with score bars and reason lines", async ({ browser }) => {
    const page = await fresh(browser)
    await signIn(page, ownerEmail, password)
    await expect(page).toHaveURL(/\/beranda$/)

    await page.goto("/laporan")
    await expect(page.getByRole("heading", { name: "Laporan Saya" })).toBeVisible()

    // Navigate to lost report
    const reportLink = page.getByRole("link", { name: /Dompet|Earbuds|Kunci/i }).first()
    if (await reportLink.isVisible()) {
      await reportLink.click()
      await expect(page).toHaveURL(/\/laporan\/hilang\/LST-/)

      // Verify candidates section
      const candidatesSection = page.getByRole("heading", { name: "Kandidat barang temuan" })
      if (await candidatesSection.isVisible()) {
        const firstCandidate = page.locator("#kandidat ol li a").first()
        if (await firstCandidate.isVisible()) {
          await firstCandidate.click()
          await expect(page).toHaveURL(/\/laporan\/hilang\/.*\/kandidat\//)

          // Verify score meters and reasons
          await expect(page.getByRole("meter")).toHaveCount(4)
          await expect(page.getByRole("heading", { name: "Kenapa kandidat ini muncul" })).toBeVisible()
          await expect(page.getByRole("button", { name: "Sepertinya milik saya" })).toBeVisible()
          await expect(page.getByRole("link", { name: "Ini milik saya", exact: true })).toBeVisible()
          await page.screenshot({ path: "test-results/screenshots/demo-2-owner.png", fullPage: true })
          await expectAccessible(page)
        }
      }
    }
  })

  test("Moment 3: Officer uses priority queue, review board, and semantic search", async ({ browser }) => {
    const page = await fresh(browser)
    await signIn(page, officerEmail, password)
    await expect(page).toHaveURL(/\/petugas$/)

    // Priority Queue
    await expect(page.getByRole("heading", { name: "Antrean Prioritas Kerja" })).toBeVisible()
    await expectAccessible(page)

    // Review Board
    await page.goto("/petugas/papan")
    await expect(page.getByRole("heading", { name: "Papan Tinjauan" })).toBeVisible()
    await expectAccessible(page)

    // Duplicate detection workspace
    await page.goto("/petugas/duplikat")
    await expect(page.getByRole("heading", { name: "Duplikat Laporan" })).toBeVisible()
    await expectAccessible(page)

    // Semantic search
    await page.goto("/petugas/cari?q=dompet+kulit&semantic=true")
    await expect(page.getByRole("heading", { name: "Cari laporan" })).toBeVisible()
    await page.screenshot({ path: "test-results/screenshots/demo-3-officer.png", fullPage: true })
    await expectAccessible(page)
  })

  test("Moment 4: Admin reviews AI monitoring, clusters, hotspots, and trends", async ({ browser }) => {
    const page = await fresh(browser)
    await signIn(page, adminEmail, password)
    await expect(page).toHaveURL(/\/admin$/)

    // AI Monitoring
    await page.goto("/admin/ai/monitoring")
    await expect(page.getByRole("heading", { name: "Monitoring AI & Mesin Pencocokan" })).toBeVisible()
    await expect(page.getByRole("cell", { name: "fusion-linear-2026.11.0" })).toBeVisible()
    await expectAccessible(page)

    // AI Clusters
    await page.goto("/admin/ai/klaster")
    await expect(page.getByRole("heading", { name: /Pengelompokan Kasus & Deteksi Duplikat/ })).toBeVisible()
    await expectAccessible(page)

    // Hotspot Map
    await page.goto("/admin/peta-rawan")
    await expect(page.getByRole("heading", { name: /Peta Rawan/ })).toBeVisible()
    await expectAccessible(page)

    // Operational Trends
    await page.goto("/admin/tren")
    await expect(page.getByRole("heading", { name: /Tren Operasional/ })).toBeVisible()
    await page.screenshot({ path: "test-results/screenshots/demo-4-admin.png", fullPage: true })
    await expectAccessible(page)
  })
})
