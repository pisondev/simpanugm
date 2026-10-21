import { expect, test } from "@playwright/test"

import { expectAccessible } from "./helpers"

test.describe("character sign-in", () => {
  test.skip(process.env.DEMO_E2E !== "1", "requires the prepared demo accounts")

  test("four character choices stay accessible and open their own dashboards", async ({ page }) => {
    test.setTimeout(90_000)
    await page.setViewportSize({ width: 1440, height: 1000 })
    await page.goto("/masuk")
    await expect(page.getByRole("heading", { name: "Masuk", exact: true })).toBeVisible()
    const demo = page.getByRole("region", { name: "Masuk Cepat Mode Demo", exact: true })
    await expect(demo.locator("img")).toHaveCount(4)
    expect(
      await demo
        .locator("img")
        .evaluateAll((images) => images.every((image) => (image as HTMLImageElement).complete && (image as HTMLImageElement).naturalWidth > 0)),
    ).toBe(true)
    await expectAccessible(page)
    await page.screenshot({ path: "test-results/screenshots/login-desktop.png", fullPage: true, animations: "disabled" })
    for (const width of [360, 390, 768]) {
      await page.setViewportSize({ width, height: 844 })
      await page.emulateMedia({ reducedMotion: "reduce" })
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await expectAccessible(page)
      if (width === 390) await page.screenshot({ path: "test-results/screenshots/login-phone.png", fullPage: true, animations: "disabled" })
    }
    for (const [name, path, feature] of [
      ["Penemu (Mahasiswa) Budi Santoso", /\/beranda\?mode=penemu$/, "Found Item Reporting"],
      ["Pemilik (Mahasiswa) Siti Rahma", /\/beranda$/, "Lost Item Reporting"],
      ["Petugas (Pos Perpus) Agus Prasetyo", /\/petugas$/, "AI-Assisted Priority Queue"],
      ["Administrator Admin SimpanUGM", /\/admin$/, "AI & Matching Monitoring"],
    ] as const) {
      await page.context().clearCookies()
      await page.goto("/masuk")
      await page.getByRole("button", { name, exact: true }).click()
      await expect(page).toHaveURL(path)
      await expect(page.getByRole("heading", { name: feature, exact: true })).toBeVisible()
    }
  })

  test("manual sign-in keeps required validation, password visibility, expired guidance and return path", async ({ page }) => {
    await page.goto("/masuk?expired=1&next=/laporan")
    await expect(page.getByText("Sesi berakhir. Masuk lagi untuk melanjutkan.", { exact: true })).toBeVisible()
    const requests: string[] = []
    page.on("request", (request) => {
      if (request.url().endsWith("/auth/login")) requests.push(request.url())
    })
    await page.getByRole("main").getByRole("button", { name: "Masuk", exact: true }).click()
    await expect(page.getByLabel("Email UGM")).toHaveAttribute("aria-invalid", "true")
    expect(requests).toEqual([])
    await page.getByLabel("Email UGM").fill("anggota.pemilik@demo.simpanugm.my.id")
    const password = page.getByLabel("Kata sandi", { exact: true })
    await password.fill("kata sandi panjang 2026")
    await page.getByRole("button", { name: "Tampilkan kata sandi", exact: true }).click()
    await expect(password).toHaveAttribute("type", "text")
    await page.getByRole("button", { name: "Sembunyikan kata sandi", exact: true }).click()
    await expect(password).toHaveAttribute("type", "password")
    await page.getByRole("main").getByRole("button", { name: "Masuk", exact: true }).click()
    await expect(page).toHaveURL(/\/laporan$/)
    expect(requests).toHaveLength(1)
  })
})
