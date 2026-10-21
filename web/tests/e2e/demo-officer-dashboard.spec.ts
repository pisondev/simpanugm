import { expect, test } from "@playwright/test"

import type { OfficerQueueList } from "../../src/lib/api/types"

import { expectAccessible } from "./helpers"

const titles = [
  "AI-Assisted Priority Queue",
  "AI Match Review Board",
  "Semantic Verification Checklist",
  "Semantic Search",
  "Duplicate Review",
  "Handover to Owner",
]

test.describe("officer dashboard", () => {
  test.skip(process.env.DEMO_E2E !== "1", "requires the prepared demo accounts")

  test.beforeEach(async ({ page }) => {
    await page.goto("/masuk")
    await page.getByRole("button", { name: "Petugas (Pos Perpus) Agus Prasetyo", exact: true }).click()
    await expect(page).toHaveURL(/\/petugas$/)
  })

  test("slide features use the live queue and remain accessible on desktop and phone", async ({ page }) => {
    test.setTimeout(90_000)
    await page.setViewportSize({ width: 1440, height: 1000 })
    await expect(page).toHaveTitle(/Dashboard Petugas/)
    for (const title of titles) await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible()
    await expect(page.locator(".officer-companion")).toBeVisible()
    const response = await page.request.get("/api/v1/officer/queue?scope=posts&limit=30")
    expect(response.ok()).toBe(true)
    const queue = (await response.json()) as OfficerQueueList
    await expect(page.locator("#officer-priority")).toContainText(`${queue.total} kasus dalam antrean`)
    const selected = queue.items.find((item) => item.claim_code) ?? queue.items[0]
    if (!selected) throw new Error("demo officer needs a queued case")
    await expect(page.getByLabel("Kasus yang ditinjau")).toHaveValue(selected.found_code)
    await expect(page.locator("#officer-board a")).toHaveAttribute("href", `/petugas/papan?report_type=found&report_id=${selected.found_id}`)
    if (selected.claim_code) {
      await expect(page.locator("#officer-checklist a")).toHaveAttribute("href", `/petugas/klaim/${selected.claim_code}#verification-checklist`)
    }
    await page.getByRole("button", { name: "Lainnya", exact: true }).filter({ visible: true }).click()
    await expect(page.getByRole("dialog").getByRole("link", { name: "Papan Tinjauan", exact: true })).toBeVisible()
    await page.keyboard.press("Escape")
    await expectAccessible(page)
    await page.screenshot({ path: "test-results/screenshots/officer-dashboard-desktop.png", fullPage: true, animations: "disabled" })
    await page.setViewportSize({ width: 390, height: 844 })
    await page.emulateMedia({ reducedMotion: "reduce" })
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    for (const title of titles) await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible()
    await expectAccessible(page)
    await page.screenshot({ path: "test-results/screenshots/officer-dashboard-phone.png", fullPage: true, animations: "disabled" })
  })

  test("case selection, campus scope and feature destinations preserve real officer workflows", async ({ page }) => {
    test.setTimeout(120_000)
    await page.setViewportSize({ width: 1440, height: 1000 })
    const options = await page.locator("#officer-case option").evaluateAll((items) => items.map((item) => (item as HTMLOptionElement).value))
    expect(options.length).toBeGreaterThan(0)
    const code = options[options.length - 1]
    if (!code) throw new Error("demo officer needs a selectable case")
    await page.getByLabel("Kasus yang ditinjau").selectOption(code)
    await page.getByRole("button", { name: "Tinjau kasus", exact: true }).click()
    await expect(page).toHaveURL(new RegExp(`kasus=${code}`))
    await expect(page.locator("#officer-board")).toContainText(code)
    await page.locator("#officer-board a").click()
    await expect(page.getByText("AI Match Review Board", { exact: true })).toBeVisible()
    await page.goto("/petugas")
    const checklist = page.locator("#officer-checklist a[href*='#verification-checklist']")
    await expect(checklist).toHaveCount(1)
    await checklist.click()
    await expect(page.locator("#verification-checklist")).toContainText("Semantic Verification Checklist")
    await page.goto("/petugas")
    const handover = page.locator("#officer-handover a")
    await expect(handover).toHaveAttribute("href", /#handover-to-owner$/)
    await handover.click()
    await expect(page.locator("#handover-to-owner")).toContainText("Handover to Owner")
    await page.goto("/petugas")
    await page.locator("#officer-duplicates a").click()
    await expect(page.getByText("Duplicate Review", { exact: true })).toBeVisible()
    await page.goto("/petugas")
    await page.getByLabel("Cari berdasarkan makna", { exact: true }).fill("tas hitam")
    await page.locator("#officer-search button").click()
    await expect(page).toHaveURL(/\/petugas\/cari\?semantic=true&q=tas\+hitam/)
    await expect(page.getByText("Semantic Search", { exact: true })).toBeVisible()
    await page.goto("/petugas")
    const scope = page.getByRole("navigation", { name: "Cakupan ruang petugas" })
    await scope.getByRole("link", { name: "Seluruh kampus", exact: true }).click()
    await expect(scope.getByRole("link", { name: "Seluruh kampus", exact: true })).toHaveAttribute("aria-current", "page")
    await expect(page.getByText("Ringkasan seluruh kampus. Tindakan tetap mengikuti kewenangan pos.")).toBeVisible()
    await scope.getByRole("link", { name: "Pos saya", exact: true }).click()
    await expect(scope.getByRole("link", { name: "Pos saya", exact: true })).toHaveAttribute("aria-current", "page")
    await expect(page.getByRole("heading", { name: "Antrean Prioritas Kerja", exact: true })).toBeVisible()
  })
})
