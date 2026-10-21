import { expect, test } from "@playwright/test"
import { resolve } from "node:path"

import { expectAccessible, registerMember, signIn, uniqueEmail } from "./helpers"

const password = "kata sandi panjang 2026"
const titles = ["Lost Item Reporting", "Smart Report Assistant", "Match Candidates", "Explainable Match", "Claim Submission", "Case History & Notifications"]

test.describe("owner dashboard presentation features", () => {
  test.skip(process.env.DEMO_E2E !== "1", "requires the prepared demo accounts")

  test("owner login exposes all six features and links them to the selected real report", async ({ page }) => {
    test.setTimeout(120_000)
    await page.setViewportSize({ width: 1440, height: 1000 })
    const errors: string[] = []
    page.on("pageerror", (error) => errors.push(error.message))
    await signIn(page, "anggota.pemilik@demo.simpanugm.my.id", password)
    await expect(page).toHaveURL(/\/beranda$/)
    for (const title of titles) await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible()
    await expect(page.locator(".owner-companion")).toBeVisible()
    await expect(page.getByRole("heading", { name: /Halo, Siti/ })).toBeVisible()
    await expect(page.getByRole("link", { name: "Lihat alasan kecocokan", exact: true })).toBeVisible()
    const code = await page.getByLabel("Laporan yang ditinjau").inputValue()
    const response = await page.request.get(`/api/v1/lost-reports/${code}/matches`)
    expect(response.ok()).toBe(true)
    const matches = (await response.json()).items as { id: string; overall: number; found_report: { item_name: string } }[]
    expect(matches.length).toBeGreaterThan(0)
    const first = matches[0]!
    await expect(page.locator("#fitur-candidates")).toContainText(first.found_report.item_name)
    await expect(page.locator("#fitur-candidates")).toContainText(`${Math.round(first.overall * 100)}%`)
    await expect(page.getByRole("link", { name: "Lihat kandidat", exact: true })).toHaveAttribute("href", `/laporan/hilang/${code}#kandidat`)
    await expect(page.getByRole("link", { name: "Lihat alasan kecocokan", exact: true })).toHaveAttribute(
      "href",
      `/laporan/hilang/${code}/kandidat/${first.id}#alasan`,
    )
    await expect(page.getByRole("link", { name: "Ajukan klaim", exact: true })).toHaveAttribute("href", `/klaim/baru?pasangan=${first.id}`)
    await expectAccessible(page)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    await page.screenshot({ path: "test-results/screenshots/owner-dashboard-desktop.png", fullPage: true, animations: "disabled" })

    await page.getByRole("link", { name: "Mulai dari foto", exact: true }).click()
    await expect(page.getByText("Smart Report Assistant", { exact: true })).toBeVisible()
    await expect(page.locator('input[type="file"][multiple]')).toHaveCount(1)
    await page.goto("/beranda")
    await page.getByRole("link", { name: "Lihat kandidat", exact: true }).click()
    await expect(page.locator("#kandidat")).toContainText("Match Candidates")
    await page.goto("/beranda")
    await page.getByRole("link", { name: "Lihat alasan kecocokan", exact: true }).click()
    await expect(page.locator("#alasan")).toContainText("Explainable Match")
    await expect(page.getByRole("meter")).toHaveCount(4)
    await page.goto("/beranda")
    await page.getByRole("link", { name: "Ajukan klaim", exact: true }).click()
    await expect(page.getByText("Claim Submission", { exact: true })).toBeVisible()
    await page.goto("/beranda")
    await page.getByRole("link", { name: /belum dibaca/ }).click()
    await expect(page.getByText("Case History & Notifications", { exact: true })).toBeVisible()
    expect(errors).toEqual([])
  })

  test("selection, unavailable candidates and narrow reduced-motion layout preserve useful actions", async ({ page }) => {
    test.setTimeout(90_000)
    await page.emulateMedia({ reducedMotion: "reduce" })
    await signIn(page, "anggota.pemilik@demo.simpanugm.my.id", password)
    await expect(page.getByRole("link", { name: "Lihat alasan kecocokan", exact: true })).toBeVisible()
    const options = await page.locator("#owner-report option").evaluateAll((nodes) => nodes.map((node) => (node as HTMLOptionElement).value))
    expect(options.length).toBeGreaterThan(1)
    const selected = await page.getByLabel("Laporan yang ditinjau").inputValue()
    const other = options.find((code) => code !== selected)!
    await page.getByLabel("Laporan yang ditinjau").selectOption(other)
    await expect(page.getByRole("link", { name: "Lihat kandidat", exact: true })).toHaveAttribute("href", `/laporan/hilang/${other}#kandidat`)
    await page.route("**/api/v1/lost-reports/*/matches", (route) =>
      route.fulfill({ status: 503, contentType: "application/problem+json", body: JSON.stringify({ title: "Unavailable", code: "unavailable", status: 503 }) }),
    )
    await page.reload()
    await expect(page.getByText("Kandidat belum dapat dimuat. Laporanmu tetap tersimpan.")).toBeVisible({ timeout: 20_000 })
    await expect(page.getByRole("button", { name: "Lihat alasan kecocokan", exact: true })).toBeDisabled()
    await expect(page.getByRole("link", { name: "Laporkan kehilangan", exact: true })).toBeVisible()
    const failedRetry = page.waitForResponse((response) => response.url().endsWith("/matches") && response.status() === 503)
    await page.getByRole("button", { name: "Coba lagi", exact: true }).click()
    await failedRetry
    await expect(page.getByRole("button", { name: "Coba lagi", exact: true })).toBeEnabled()
    await page.unroute("**/api/v1/lost-reports/*/matches")
    await page.getByRole("button", { name: "Coba lagi", exact: true }).click()
    await expect(page.getByRole("link", { name: "Lihat alasan kecocokan", exact: true })).toBeVisible()
    for (const width of [360, 768, 1280]) {
      await page.setViewportSize({ width, height: 900 })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await expectAccessible(page)
      await page.screenshot({ path: `test-results/screenshots/owner-dashboard-${width}.png`, fullPage: true, animations: "disabled" })
    }
  })

  test("a new member has visible feature names and honest prerequisites without fake candidates", async ({ page }) => {
    test.setTimeout(90_000)
    const email = uniqueEmail("owner-dashboard")
    await registerMember(page, "Rani Dashboard", email, password)
    await signIn(page, email, password)
    await expect(page).toHaveURL(/\/beranda$/)
    for (const title of titles) await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible()
    await expect(page.getByText("Belum ada laporan kehilangan aktif.", { exact: true }).first()).toBeVisible()
    await expect(page.getByRole("button", { name: "Lihat kandidat", exact: true })).toBeDisabled()
    await expect(page.getByRole("button", { name: "Ajukan klaim", exact: true })).toBeDisabled()
    await expect(page.locator("#fitur-candidates")).not.toContainText("%")
    await expect(page.getByRole("link", { name: "Mulai dari foto", exact: true })).toBeVisible()
    await expectAccessible(page)
  })

  test("the owner photo assistant fills an editable description from the real existing template endpoint", async ({ page }) => {
    test.setTimeout(90_000)
    await signIn(page, "anggota.pemilik@demo.simpanugm.my.id", password)
    await page.getByRole("link", { name: "Mulai dari foto", exact: true }).click()
    await page.locator('input[type="file"][multiple]').setInputFiles(resolve("../data/demo/images/hero-wallet-found.jpg"))
    await expect(page.getByRole("button", { name: "Hapus foto 1", exact: true })).toBeVisible()
    await expect(page.getByText(/Mengunggah|Menyiapkan foto/)).toHaveCount(0, { timeout: 20_000 })
    const described = page.waitForResponse((response) => response.url().endsWith("/api/v1/assist/describe") && response.status() === 200, { timeout: 30_000 })
    await page.getByRole("button", { name: "Lanjut", exact: true }).click()
    await expect(page.getByRole("status").filter({ hasText: /Saran dari fotomu sudah siap|Saran otomatis sedang tidak tersedia/ })).toBeVisible({
      timeout: 12_000,
    })
    if (await page.getByText("Saran otomatis sedang tidak tersedia", { exact: true }).isVisible()) {
      await page.getByRole("button", { name: "Coba saran lagi", exact: true }).click()
    }
    const actual = (await (await described).json()) as { description: string }
    await expect(page.getByLabel("Deskripsi", { exact: true })).toHaveValue(actual.description)
    await expect(page.getByRole("button", { name: "Buat ulang dari foto", exact: true })).toBeVisible()
    const correction = "Dompet milik saya, ada bekas jahitan pada sisi kiri."
    await page.getByLabel("Deskripsi", { exact: true }).fill(correction)
    await expect(page.getByLabel("Deskripsi", { exact: true })).toHaveValue(correction)
    await expectAccessible(page)
  })
})
