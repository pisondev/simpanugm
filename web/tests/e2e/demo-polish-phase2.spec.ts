import { mkdir } from "node:fs/promises"

import { expect, test, type Page } from "@playwright/test"

import { expectAccessible, latestLink, pathOf, registerMember, uniqueEmail } from "./helpers"

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

async function captureAuth(page: Page, title: string, name: string, width: string, batch = "B02") {
  await expect(page.getByRole("heading", { level: 1, name: title })).toBeVisible()
  await page.evaluate(() => Promise.all(document.getAnimations().map((animation) => animation.finished.catch(() => undefined))))
  await expectAccessible(page)
  await expect(page.locator("html")).toHaveJSProperty("scrollWidth", await page.evaluate(() => window.innerWidth))
  await mkdir("test-results/screenshots", { recursive: true })
  await page.screenshot({
    path: `test-results/screenshots/phase2-${batch}-${name}-${width}.png`,
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

test.describe("B02 sign-in and verification views", () => {
  for (const viewport of [
    { width: 1366, height: 900, label: "1366x900" },
    { width: 360, height: 800, label: "360x800" },
  ]) {
    test(`auth entry and recovery remain usable at ${viewport.label}`, async ({ page }) => {
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

      await page.goto("/masuk")
      await captureAuth(page, "Masuk", "masuk", viewport.label)
      await expect(page.getByText("Masuk Cepat Mode Demo", { exact: true })).toBeVisible()
      await expect(page.getByRole("button", { name: "Penemu (Mahasiswa)" })).toBeVisible()
      if (viewport.width === 1366) {
        await page.getByRole("button", { name: "Penemu (Mahasiswa)" }).click()
        await expect(page).toHaveURL(/\/beranda$/)
        await expect(page.getByRole("heading", { name: /Halo, Budi/ })).toBeVisible()
        await page.context().clearCookies()
      }

      await page.goto("/daftar")
      await captureAuth(page, "Daftar", "daftar", viewport.label)
      await expect(page.getByRole("link", { name: "pemberitahuan privasi" })).toBeVisible()
      await page.getByRole("button", { name: "Tampilkan kata sandi" }).click()
      await expect(page.getByLabel("Kata sandi", { exact: true })).toHaveAttribute("type", "text")
      await page.getByRole("button", { name: "Sembunyikan kata sandi" }).click()
      await expect(page.getByLabel("Kata sandi", { exact: true })).toHaveAttribute("type", "password")
      await page.getByRole("button", { name: "Buat akun" }).click()
      await expect(page.getByLabel("Nama lengkap")).toHaveAttribute("aria-invalid", "true")
      await expectAccessible(page)

      await page.goto("/verifikasi")
      await expect(page.getByText("Tautan verifikasi tidak lengkap.")).toBeVisible()
      await captureAuth(page, "Verifikasi email", "verifikasi-missing", viewport.label)
      await page.goto("/verifikasi?token=expired-invalid")
      await expect(page.getByRole("heading", { name: "Tautan kedaluwarsa" })).toBeVisible()
      await expect(page.getByRole("main").getByRole("link", { name: "Masuk", exact: true })).toBeVisible()
      await captureAuth(page, "Verifikasi email", "verifikasi-expired", viewport.label)

      await page.goto("/konfirmasi-email")
      await expect(page.getByText("Tautan konfirmasi tidak lengkap.")).toBeVisible()
      await captureAuth(page, "Konfirmasi email baru", "konfirmasi-missing", viewport.label)
      await page.goto("/konfirmasi-email?token=expired-invalid")
      await expect(page.getByText(/Tautan tidak berlaku/)).toBeVisible()
      await expect(page.getByRole("link", { name: "Ke Profil" })).toBeVisible()
      await captureAuth(page, "Konfirmasi email baru", "konfirmasi-expired", viewport.label)

      expect(external).toEqual([])
      expect(errors).toEqual([])
    })
  }
})

test.describe("B03 account recovery views", () => {
  for (const viewport of [
    { width: 1366, height: 900, label: "1366x900" },
    { width: 360, height: 800, label: "360x800" },
  ]) {
    test(`recovery forms and expired-link guidance at ${viewport.label}`, async ({ page }) => {
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

      await page.goto("/lupa-sandi")
      await captureAuth(page, "Lupa kata sandi", "lupa-sandi", viewport.label, "B03")
      await page.getByRole("button", { name: "Kirim tautan" }).click()
      await expect(page.getByLabel("Email UGM")).toHaveAttribute("aria-invalid", "true")
      await expectAccessible(page)

      await page.goto("/atur-sandi")
      await expect(page.getByText(/Tautan atur ulang tidak lengkap/)).toBeVisible()
      await expect(page.getByRole("link", { name: "Minta tautan baru" })).toBeVisible()
      await captureAuth(page, "Atur kata sandi baru", "atur-sandi-missing", viewport.label, "B03")

      await page.goto("/atur-sandi?token=expired-invalid")
      await captureAuth(page, "Atur kata sandi baru", "atur-sandi-expired", viewport.label, "B03")
      await page.getByRole("button", { name: "Tampilkan kata sandi" }).first().click()
      await expect(page.getByLabel("Kata sandi baru", { exact: true })).toHaveAttribute("type", "text")
      await page.getByRole("button", { name: "Sembunyikan kata sandi" }).first().click()
      await expect(page.getByLabel("Kata sandi baru", { exact: true })).toHaveAttribute("type", "password")
      await page.getByLabel("Kata sandi baru", { exact: true }).fill("sandi baru aman phase2")
      await page.getByLabel("Ulangi kata sandi baru", { exact: true }).fill("berbeda")
      await page.getByRole("button", { name: "Simpan kata sandi" }).click()
      await expect(page.getByText("Kata sandi tidak sama")).toBeVisible()
      await page.getByLabel("Ulangi kata sandi baru", { exact: true }).fill("sandi baru aman phase2")
      await page.getByRole("button", { name: "Simpan kata sandi" }).click()
      await expect(page.getByRole("link", { name: "Minta tautan baru" })).toBeVisible()
      await expectAccessible(page)

      await page.goto("/undangan")
      await expect(page.getByText(/Tautan undangan tidak lengkap/)).toBeVisible()
      await captureAuth(page, "Terima undangan", "undangan-missing", viewport.label, "B03")
      await page.goto("/undangan?token=invitation-preview-token")
      await captureAuth(page, "Terima undangan", "undangan-form", viewport.label, "B03")
      await expect(page.getByLabel("Kata sandi baru", { exact: true })).toBeVisible()
      await expect(page.getByLabel("Ulangi kata sandi baru", { exact: true })).toBeVisible()

      expect(external).toEqual([])
      expect(errors).toEqual([])
    })
  }

  test("a real reset email changes the password only after the API accepts its token", async ({ page }) => {
    test.setTimeout(120_000)
    const email = uniqueEmail("phase2-reset")
    const oldPassword = "kata sandi lama phase2 2026"
    const newPassword = "kata sandi baru phase2 2026"
    await registerMember(page, "Pemulihan Uji", email, oldPassword)
    await page.goto("/lupa-sandi")
    await page.getByLabel("Email UGM").fill(email)
    await page.getByRole("button", { name: "Kirim tautan" }).click()
    await expect(page.getByText(/Jika email terdaftar, tautan sudah dikirim/)).toBeVisible()
    const resetLink = await latestLink(email, "/atur-sandi")
    await page.goto(pathOf(resetLink))
    await page.getByLabel("Kata sandi baru", { exact: true }).fill(newPassword)
    await page.getByLabel("Ulangi kata sandi baru", { exact: true }).fill(newPassword)
    await page.getByRole("button", { name: "Simpan kata sandi" }).click()
    await expect(page.getByText("Kata sandi sudah diganti. Masuk dengan kata sandi baru.")).toBeVisible()
    await expect(page.getByRole("main").getByRole("link", { name: "Masuk", exact: true })).toBeVisible()
    await expectAccessible(page)
  })
})
