import { expect, test } from "@playwright/test"

import { expectAccessible, signIn } from "./helpers"

const password = "kata sandi panjang 2026"
const titles = [
  "Found Item Reporting",
  "AI-Assisted Item Description",
  "Instant Match Suggestions",
  "Duplicate / Related Report Detection",
  "Handover to Officer",
  "Case History & Notifications",
]

test.describe("finder dashboard", () => {
  test.skip(process.env.DEMO_E2E !== "1", "requires the prepared demo accounts")

  test("finder login, real private-safe hints and feature destinations work on desktop and phone", async ({ page }) => {
    test.setTimeout(90_000)
    await page.setViewportSize({ width: 1440, height: 1000 })
    const errors: string[] = []
    page.on("pageerror", (error) => errors.push(error.message))
    await page.goto("/masuk")
    await page.getByRole("button", { name: "Penemu (Mahasiswa)" }).click()
    await expect(page).toHaveURL(/\/beranda\?mode=penemu$/)
    await expect(page).toHaveTitle(/Dashboard Penemu Barang/)
    const switcher = page.getByRole("navigation", { name: "Tampilan dashboard" })
    await expect(switcher.getByRole("link", { name: "Penemu barang", exact: true })).toHaveAttribute("aria-current", "page")
    for (const title of titles) await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible()
    await expect(page.locator(".finder-companion")).toBeVisible()

    const codes = await page.locator("#finder-report option").evaluateAll((options) => options.map((option) => (option as HTMLOptionElement).value))
    let matchedCode = ""
    for (const code of codes) {
      const response = await page.request.get(`/api/v1/found-reports/${code}/match-hint`)
      expect(response.ok()).toBe(true)
      const items = (await response.json()).items as { item_name: string; category_name: string }[]
      if (!items.length) continue
      matchedCode = code
      await page.getByLabel("Temuan yang ditinjau").selectOption(code)
      for (const item of items.slice(0, 2)) {
        expect(Object.keys(item).sort()).toEqual(["category_name", "item_name"])
        await expect(page.locator("#finder-suggestions")).toContainText(item.item_name)
      }
      break
    }
    expect(matchedCode).not.toBe("")
    await expect(page.locator("#finder-suggestions")).not.toContainText("%")
    await expect(page.getByRole("link", { name: "Lihat petunjuk penyerahan", exact: true })).toHaveAttribute(
      "href",
      `/laporan/temuan/${matchedCode}#penyerahan`,
    )
    await expectAccessible(page)
    await page.screenshot({ path: "test-results/screenshots/finder-dashboard-desktop.png", fullPage: true, animations: "disabled" })
    await page.setViewportSize({ width: 390, height: 844 })
    await page.emulateMedia({ reducedMotion: "reduce" })
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    await expectAccessible(page)
    await page.screenshot({ path: "test-results/screenshots/finder-dashboard-phone.png", fullPage: true, animations: "disabled" })
    await page.setViewportSize({ width: 1440, height: 1000 })
    await page.getByRole("link", { name: "Lihat petunjuk penyerahan", exact: true }).click()
    await expect(page.locator("#penyerahan")).toContainText("Handover to Officer")
    await expect(page.locator("#saran-pemilik")).toContainText("Instant Match Suggestions")
    await page.goto("/beranda?mode=penemu")
    await page.getByRole("link", { name: "Mulai dari foto", exact: true }).click()
    await expect(page.locator("#ai-assisted-description")).toContainText("AI-Assisted Item Description")
    await page.goto("/beranda?mode=penemu")
    await page.getByRole("link", { name: "Mulai pemeriksaan duplikat", exact: true }).click()
    await expect(page.getByText("Aktif pada tahap Tinjau sebelum laporan baru dikirim. Lengkapi foto dan detail barang terlebih dahulu.")).toBeVisible()
    await page.goto("/beranda?mode=penemu")
    await switcher.getByRole("link", { name: "Pemilik barang", exact: true }).click()
    await expect(page.getByRole("heading", { name: "Lost Item Reporting", exact: true })).toBeVisible()
    await switcher.getByRole("link", { name: "Penemu barang", exact: true }).click()
    await expect(page.getByRole("heading", { name: "Found Item Reporting", exact: true })).toBeVisible()
    expect(errors).toEqual([])
  })

  test("empty finder mode and failed hints keep reporting and custody usable", async ({ page }) => {
    test.setTimeout(60_000)
    await signIn(page, "anggota.pemilik@demo.simpanugm.my.id", password)
    await expect(page).toHaveURL(/\/beranda$/)
    await page.goto("/beranda?mode=penemu")
    await expect(page.getByText("Belum ada laporan temuan aktif.", { exact: true })).toBeVisible()
    await expect(page.getByRole("button", { name: "Lihat saran kecocokan", exact: true })).toBeDisabled()
    await expect(page.getByRole("link", { name: "Laporkan barang temuan", exact: true })).toBeVisible()
    await expectAccessible(page)
    await page.context().clearCookies()
    await page.route("**/api/v1/found-reports/*/match-hint", (route) =>
      route.fulfill({ status: 503, contentType: "application/problem+json", body: JSON.stringify({ title: "Unavailable", code: "unavailable", status: 503 }) }),
    )
    await signIn(page, "anggota.penemu@demo.simpanugm.my.id", password)
    await expect(page).toHaveURL(/\/beranda$/)
    await page.goto("/beranda?mode=penemu")
    await expect(page.getByText("Saran belum dapat dimuat. Laporan dan petunjuk penyerahan tetap tersedia.")).toBeVisible()
    await expect(page.getByRole("link", { name: "Lihat petunjuk penyerahan", exact: true })).toBeVisible()
    await expect(page.locator("#finder-handover")).toContainText(/FND-/)
    await page.unroute("**/api/v1/found-reports/*/match-hint")
    await page.getByRole("button", { name: "Coba lagi", exact: true }).click()
    await expect(page.getByText("Saran belum dapat dimuat. Laporan dan petunjuk penyerahan tetap tersedia.")).toHaveCount(0)
    await expectAccessible(page)
  })
})
