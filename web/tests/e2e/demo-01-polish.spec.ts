import { readFile } from "node:fs/promises"
import { resolve } from "node:path"

import { expect, test, type Page } from "@playwright/test"

import { expectAccessible, signIn } from "./helpers"

const password = "kata sandi panjang 2026"

async function capture(page: Page, name: string) {
  await expect(page.locator("main")).toBeVisible()
  await page.evaluate(async () => {
    await document.fonts.ready
    await Promise.all(Array.from(document.images, (image) => image.decode().catch(() => undefined)))
  })
  await expectAccessible(page)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await page.screenshot({ path: `test-results/screenshots/phase1-${name}.png`, fullPage: true, animations: "disabled" })
}

async function uploadDemoPhoto(page: Page) {
  const buffer = await readFile(resolve("../data/demo/images/hero-earbuds-found.jpg"))
  const uploaded = page.waitForResponse((response) => response.url().endsWith("/api/v1/media") && response.request().method() === "POST", { timeout: 20_000 })
  await expect(async () => {
    await page.locator('input[type="file"][multiple]').setInputFiles({ name: "earbuds.jpg", mimeType: "image/jpeg", buffer })
    await expect(page.getByRole("button", { name: "Hapus foto 1" })).toBeVisible({ timeout: 3000 })
  }).toPass({ timeout: 20_000 })
  expect((await uploaded).status()).toBe(201)
  await expect(page.getByText(/Mengunggah|Menyiapkan foto/)).toHaveCount(0, { timeout: 20_000 })
  await expect(page.getByText("Gagal mengunggah", { exact: true })).toHaveCount(0)
}

test.describe("phase 1 reference screens", () => {
  test.skip(!process.env.DEMO_E2E, "requires the real seeded demo stack")

  for (const viewport of [
    { label: "desktop", width: 1366, height: 900 },
    { label: "phone", width: 360, height: 800 },
  ]) {
    test(`landing, owner candidates and photo suggestions at ${viewport.label} width`, async ({ page }) => {
      test.setTimeout(120_000)
      await page.setViewportSize(viewport)
      const external: string[] = []
      const broken: string[] = []
      page.on("request", (request) => {
        const url = new URL(request.url())
        if (/^https?:$/.test(url.protocol) && !url.hostname.match(/^(localhost|127\.0\.0\.1|\[::1\])$/)) external.push(request.url())
      })
      page.on("pageerror", (error) => broken.push(error.message))
      await page.goto("/")
      await expect(page.getByRole("heading", { level: 1 })).toContainText("Yang hilang,")
      await capture(page, `landing-${viewport.label}`)

      await signIn(page, "anggota.pemilik@demo.simpanugm.my.id", password)
      await expect(page).toHaveURL(/\/beranda$/)
      await page.goto("/laporan")
      const reportsResponse = await page.request.get(new URL("/api/v1/lost-reports?mine=true&limit=20", page.url()).toString())
      expect(reportsResponse.ok()).toBe(true)
      const reports = (await reportsResponse.json()).items as { code: string }[]
      let candidateCode: string | undefined
      for (const report of reports) {
        const matchesResponse = await page.request.get(new URL(`/api/v1/lost-reports/${report.code}/matches`, page.url()).toString())
        expect(matchesResponse.ok()).toBe(true)
        if (((await matchesResponse.json()).items as unknown[]).length > 0) {
          candidateCode = report.code
          break
        }
      }
      if (!candidateCode) throw new Error("the demo owner needs a report with at least one current candidate")
      await page.goto("/laporan")
      await page.locator(`main a[href="/laporan/hilang/${candidateCode}"]`).click()
      const candidates = page.locator('#kandidat a[href*="/kandidat/"]')
      await expect(candidates).not.toHaveCount(0)
      await capture(page, `candidate-list-${viewport.label}`)
      await candidates.first().click()
      await expect(page.getByRole("meter")).toHaveCount(4)
      await expect(page.getByRole("heading", { name: "Kenapa kandidat ini muncul" })).toBeVisible()
      await expect(page.getByRole("link", { name: "Ini milik saya", exact: true })).toBeVisible()
      await expect(page.getByRole("button", { name: "Sepertinya milik saya", exact: true })).toBeVisible()
      await capture(page, `candidate-${viewport.label}`)

      await page.context().clearCookies()
      await signIn(page, "anggota.penemu@demo.simpanugm.my.id", password)
      await expect(page).toHaveURL(/\/beranda$/)
      await page.goto("/lapor/temuan")
      await capture(page, `found-photo-${viewport.label}`)
      await uploadDemoPhoto(page)
      await page.getByRole("button", { name: "Lanjut", exact: true }).click()
      await expect(page.getByRole("status").filter({ hasText: /Saran dari fotomu sudah siap|Saran otomatis sedang tidak tersedia/ })).toBeVisible({
        timeout: 12_000,
      })
      if (await page.getByText("Saran otomatis sedang tidak tersedia", { exact: true }).isVisible()) {
        await page.getByRole("button", { name: "Coba saran lagi", exact: true }).click()
      }
      await expect(page.getByText("Saran dari fotomu sudah siap", { exact: true })).toBeVisible({ timeout: 15_000 })
      await expect(page.getByText("Saran otomatis", { exact: true }).first()).toBeVisible()
      await expect(page.getByLabel("Deskripsi", { exact: true })).not.toBeEmpty({ timeout: 12_000 })
      await page.getByLabel("Nama barang", { exact: true }).fill("Earbuds yang ditemukan")
      await capture(page, `found-assist-${viewport.label}`)
      expect(external).toEqual([])
      expect(broken).toEqual([])
    })
  }

  test("reduced motion disables transitions and slow or failed analysis leaves manual entry usable", async ({ page }) => {
    await page.emulateMedia({ reducedMotion: "reduce" })
    await signIn(page, "anggota.penemu@demo.simpanugm.my.id", password)
    await expect(page).toHaveURL(/\/beranda$/)
    await page.route("**/api/v1/assist/analyze-image", async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 1500))
      await route.abort("failed")
    })
    await page.goto("/lapor/temuan")
    await uploadDemoPhoto(page)
    await page.getByRole("button", { name: "Lanjut", exact: true }).click()
    await expect(page.getByText("Menyiapkan saran dari foto...", { exact: true })).toBeVisible()
    await page.getByLabel("Nama barang", { exact: true }).fill("Barang tanpa bantuan model")
    await expect(page.getByText("Saran otomatis sedang tidak tersedia", { exact: true })).toBeVisible({ timeout: 15_000 })
    await expect(page.getByLabel("Nama barang", { exact: true })).toHaveValue("Barang tanpa bantuan model")
    await capture(page, "found-unavailable-reduced")
    expect(
      await page
        .locator("[data-slot=button]")
        .first()
        .evaluate((button) => getComputedStyle(button).transitionDuration),
    ).toBe("1e-05s")
  })
})
