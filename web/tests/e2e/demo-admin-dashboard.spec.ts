import { expect, test } from "@playwright/test"

import type { AiMonitoringResponse, AnalyticsSummary } from "../../src/lib/api/types"

import { expectAccessible, signIn } from "./helpers"

const titles = [
  "AI & Matching Monitoring",
  "Drift Detection",
  "Fusion Control & Rollback",
  "HDBSCAN Case Clustering",
  "Lost & Found Hotspot Analytics",
  "Trend Analytics",
]

test.describe("administrator dashboard", () => {
  test.skip(process.env.DEMO_E2E !== "1", "requires the prepared demo accounts")

  test.beforeEach(async ({ page }) => {
    await page.goto("/masuk")
    await page.getByRole("button", { name: "Administrator Admin SimpanUGM", exact: true }).click()
    await expect(page).toHaveURL(/\/admin$/)
  })

  test("six slide features show real metrics and remain accessible on desktop and phone", async ({ page }) => {
    test.setTimeout(90_000)
    await page.setViewportSize({ width: 1440, height: 1000 })
    const errors: string[] = []
    page.on("pageerror", (error) => errors.push(error.message))
    await expect(page).toHaveTitle(/Dashboard Administrator/)
    for (const title of titles) await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible()
    await expect(page.locator(".admin-companion")).toBeVisible()
    await expect(page.locator(".admin-workspace").getByText("AI Feature", { exact: true })).toHaveCount(5)
    const monResponse = await page.request.get("/api/v1/admin/ai/monitoring")
    expect(monResponse.ok()).toBe(true)
    const mon = (await monResponse.json()) as AiMonitoringResponse
    await expect(page.locator("#admin-monitoring")).toContainText(`${mon.confirmed_ranks.confirmed_total} klaim terkonfirmasi`)
    if (mon.confirmed_ranks.confirmed_total === 0) {
      await expect(page.locator("#admin-monitoring")).toContainText("Belum ada kasus terkonfirmasi untuk membaca Recall@1.")
      await expect(page.locator("#admin-monitoring")).not.toContainText("0.0%")
    }
    await expect(page.locator("#admin-drift")).toContainText(`PSI distribusi skor: ${mon.score_psi.toFixed(3)}`)
    const fusion = mon.active_versions.find((version) => version.component === "fusion")
    if (fusion) await expect(page.locator("#admin-fusion")).toContainText(fusion.name)
    const summaryResponse = await page.request.get("/api/v1/analytics/summary")
    expect(summaryResponse.ok()).toBe(true)
    const summary = (await summaryResponse.json()) as AnalyticsSummary
    await expect(page.locator("#admin-trends")).toContainText(`${summary.total_lost} kehilangan | ${summary.total_found} temuan`)
    await page.getByRole("button", { name: "Lainnya", exact: true }).filter({ visible: true }).click()
    await expect(page.getByRole("dialog").getByRole("link", { name: "Audit Log", exact: true })).toBeVisible()
    await expect(page.getByRole("dialog").getByRole("link", { name: "Beranda", exact: true })).toBeVisible()
    await page.keyboard.press("Escape")
    await expectAccessible(page)
    await page.screenshot({ path: "test-results/screenshots/admin-dashboard-desktop.png", fullPage: true, animations: "disabled" })
    await page.setViewportSize({ width: 390, height: 844 })
    await page.emulateMedia({ reducedMotion: "reduce" })
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    for (const title of titles) await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible()
    await expectAccessible(page)
    await page.screenshot({ path: "test-results/screenshots/admin-dashboard-phone.png", fullPage: true, animations: "disabled" })
    expect(errors).toEqual([])
  })

  test("feature shortcuts and administrative menus work while officer access remains denied", async ({ page }) => {
    test.setTimeout(120_000)
    await page.setViewportSize({ width: 1440, height: 1000 })
    await page.locator("#admin-monitoring a").click()
    await expect(page.getByText("AI & Matching Monitoring", { exact: true })).toBeVisible()
    await page.goto("/admin")
    await page.locator("#admin-drift a").click()
    await expect(page).toHaveURL(/#drift-detection$/)
    await expect(page.locator("#drift-detection")).toContainText("Drift Detection")
    await page.goto("/admin")
    await page.locator("#admin-fusion a").click()
    await expect(page).toHaveURL(/#fusion-control$/)
    await expect(page.locator("#fusion-control")).toContainText("Fusion Control & Rollback")
    await page.locator("#fusion-control").getByRole("button", { name: "Kelola & Rollback Model", exact: true }).click()
    await expect(page.getByRole("dialog")).toContainText("Ganti atau Rollback Versi Model")
    await expect(page.getByRole("button", { name: "Aktifkan Versi Ini", exact: true })).toBeDisabled()
    await page.getByRole("button", { name: "Batal", exact: true }).click()
    await page.goto("/admin")
    await page.locator("#admin-clusters a").click()
    await expect(page.getByText("HDBSCAN Case Clustering", { exact: true })).toBeVisible()
    await page.goto("/admin")
    await page.locator("#admin-hotspots a").click()
    await expect(page.getByText("Lost & Found Hotspot Analytics", { exact: true })).toBeVisible()
    await expect(page.getByRole("region", { name: "Peta interaktif kampus UGM", exact: true })).toBeVisible()
    await page.goto("/admin")
    await page.locator("#admin-trends a").click()
    await expect(page.getByText("Trend Analytics", { exact: true })).toBeVisible()
    await page.goto("/admin")
    const navigation = page.getByRole("navigation", { name: "Admin", exact: true })
    await navigation.getByRole("link", { name: "Pengguna", exact: true }).click()
    await expect(page.getByRole("heading", { name: "Pengguna", exact: true })).toBeVisible()
    await page.context().clearCookies()
    await signIn(page, "petugas.pos1@demo.simpanugm.my.id", "kata sandi panjang 2026")
    await expect(page).toHaveURL(/\/petugas$/)
    await page.goto("/admin")
    await expect(page).toHaveURL(/\/petugas$/)
    await expect(page.getByRole("heading", { name: "AI & Matching Monitoring", exact: true })).toHaveCount(0)
    expect((await page.request.get("/api/v1/admin/ai/monitoring")).status()).toBe(403)
  })
})
