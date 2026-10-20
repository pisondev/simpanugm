import { mkdir } from "node:fs/promises"

import { expect, test, type Page } from "@playwright/test"

import { expectAccessible } from "./helpers"

const routes = [
  { path: "/bantuan", slug: "bantuan", title: "Bantuan" },
  { path: "/privasi", slug: "privasi", title: "Pemberitahuan Privasi" },
  { path: "/ketentuan", slug: "ketentuan", title: "Ketentuan Penggunaan" },
]

async function capture(page: Page, route: (typeof routes)[number], width: string) {
  await expect(page.getByRole("main")).toBeVisible()
  await expect(page.getByRole("heading", { level: 1, name: route.title })).toBeVisible()
  await page.evaluate(() => Promise.all(document.getAnimations().map((animation) => animation.finished.catch(() => undefined))))
  await expectAccessible(page)
  await expect(page.locator("html")).toHaveJSProperty("scrollWidth", await page.evaluate(() => window.innerWidth))
  await mkdir("test-results/screenshots", { recursive: true })
  await page.screenshot({
    path: `test-results/screenshots/phase2-B01-${route.slug}-${width}.png`,
    fullPage: true,
    animations: "disabled",
  })
}

test.describe("B01 public help and legal views", () => {
  for (const viewport of [
    { width: 1366, height: 900, label: "1366x900" },
    { width: 360, height: 800, label: "360x800" },
  ]) {
    test(`all pages retain public content at ${viewport.label}`, async ({ page }) => {
      test.setTimeout(120_000)
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      const external: string[] = []
      const errors: string[] = []
      page.on("request", (request) => {
        const url = new URL(request.url())
        if (/^https?:$/.test(url.protocol) && !/^(localhost|127\.0\.0\.1|\[::1\])$/.test(url.hostname)) {
          external.push(request.url())
        }
      })
      page.on("pageerror", (error) => errors.push(error.message))

      await page.goto("/bantuan")
      await page.keyboard.press("Tab")
      const skipLink = page.getByRole("link", { name: "Lewati ke konten" })
      await expect(skipLink).toBeFocused()
      expect(await skipLink.evaluate((element) => element.getBoundingClientRect().height)).toBeGreaterThan(0)

      for (const route of routes) {
        await page.goto(route.path)
        await capture(page, route, viewport.label)
        await expect(page.getByRole("link", { name: "Masuk", exact: true })).toBeVisible()
        await expect(page.getByRole("link", { name: "Daftar", exact: true })).toBeVisible()
        await expect(page.getByRole("contentinfo").getByText("bukan layanan resmi universitas", { exact: false })).toBeVisible()

        if (route.path === "/bantuan") {
          await expect(page.getByRole("heading", { name: "Pos barang temuan" })).toBeVisible()
          await expect(page.getByRole("heading", { name: "Pertanyaan umum" })).toBeVisible()
          await expect(page.locator("table caption").first()).toContainText("Jam buka")
          const response = await page.request.get(new URL("/api/v1/posts", page.url()).toString())
          const posts = (await response.json()).items as { phone?: string | null }[]
          const contacts = posts.filter((post) => post.phone)
          const phoneLinks = page.locator('a[href^="tel:"]')
          await expect(phoneLinks).toHaveCount(contacts.length)
          if (contacts.length > 0) await expect(phoneLinks.first()).toBeVisible()
        }
        if (route.path === "/privasi") {
          await expect(page.getByRole("heading", { name: "Data yang kami kumpulkan" })).toBeVisible()
          await expect(page.getByRole("heading", { name: "Pemrosesan foto dan AI" })).toBeVisible()
        }
        if (route.path === "/ketentuan") {
          await expect(page.getByRole("heading", { name: "Laporan yang jujur" })).toBeVisible()
          await expect(page.getByRole("heading", { name: "Serah terima" })).toBeVisible()
        }
      }

      expect(external).toEqual([])
      expect(errors).toEqual([])
    })
  }
})
