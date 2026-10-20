import { mkdir } from "node:fs/promises"

import { expect, test, type Locator, type Page } from "@playwright/test"

import { attachPhoto, expectAccessible, latestLink, pathOf, registerMember, signIn, uniqueEmail } from "./helpers"

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

async function captureB04(page: Page, route: string, width: string) {
  await expect(page.getByRole("main")).toBeVisible()
  await page.evaluate(async () => {
    await document.fonts.ready
    const finiteAnimations = document.getAnimations().filter((animation) => animation.effect?.getComputedTiming().iterations !== Infinity)
    await Promise.all(finiteAnimations.map((animation) => animation.finished.catch(() => undefined)))
  })
  await expectAccessible(page)
  const overflow = await page.evaluate(() => ({ viewport: window.innerWidth, document: document.documentElement.scrollWidth }))
  await mkdir("test-results/screenshots", { recursive: true })
  await page.screenshot({
    path: `test-results/screenshots/phase2-B04-${route}-${width}.png`,
    fullPage: true,
    animations: "disabled",
  })
  expect(overflow.document).toBe(overflow.viewport)
}

test.describe("B04 member discovery and lost reporting", () => {
  test("home, reports, public finder filters and a photo-free lost report work across screen sizes", async ({ page }) => {
    test.setTimeout(180_000)
    const external: string[] = []
    const errors: string[] = []
    page.on("request", (request) => {
      const url = new URL(request.url())
      if (/^https?:$/.test(url.protocol) && !/^(localhost|127\.0\.0\.1|\[::1\])$/.test(url.hostname)) {
        external.push(request.url())
      }
    })
    page.on("pageerror", (error) => errors.push(error.message))

    await page.setViewportSize({ width: 1366, height: 900 })
    const email = uniqueEmail("phase2-b04")
    const password = "kata sandi panjang fase 2"
    await registerMember(page, "Anggota Uji B04", email, password)
    await signIn(page, email, password)
    await expect(page).toHaveURL(/\/beranda$/)
    await expect(page.getByRole("link", { name: "Lapor kehilangan" })).toBeVisible()
    await expect(page.getByRole("heading", { name: "Laporan kehilangan aktif" })).toBeVisible()
    await captureB04(page, "beranda", "1366x900")

    await page.setViewportSize({ width: 360, height: 800 })
    await captureB04(page, "beranda", "360x800")
    await page.goto("/laporan")
    await expect(page.getByRole("link", { name: "Kehilangan", exact: true })).toHaveAttribute("aria-current", "page")
    await captureB04(page, "laporan-hilang", "360x800")
    await page.setViewportSize({ width: 1366, height: 900 })
    await captureB04(page, "laporan-hilang", "1366x900")
    await page.goto("/laporan?tab=temuan")
    await expect(page.getByRole("link", { name: "Temuan", exact: true })).toHaveAttribute("aria-current", "page")
    await captureB04(page, "laporan-temuan", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB04(page, "laporan-temuan", "360x800")

    await page.goto("/barang-temuan")
    await expect(page.getByRole("heading", { name: "Barang Temuan" })).toBeVisible()
    await captureB04(page, "barang-temuan", "360x800")
    await page.setViewportSize({ width: 1366, height: 900 })
    await captureB04(page, "barang-temuan", "1366x900")
    const publicResponse = await page.request.get(new URL("/api/v1/found-reports/public?limit=30", page.url()).toString())
    expect(publicResponse.ok()).toBe(true)
    const publicItems = (await publicResponse.json()).items as Record<string, unknown>[]
    expect(publicItems.length).toBeGreaterThan(0)
    for (const item of publicItems) {
      expect(item).not.toHaveProperty("description")
      expect(item).not.toHaveProperty("private_details")
      expect(item).not.toHaveProperty("photos")
      expect(item).not.toHaveProperty("finder")
    }
    const publicList = page.getByRole("main").locator("ul").last()
    await expect(publicList.locator("img")).toHaveCount(0)
    await page.keyboard.press("Tab")
    await expect(page.locator(":focus")).toBeVisible()
    await page.getByRole("link", { name: "Halaman berikutnya" }).click()
    await expect(page).toHaveURL(/cursor=/)
    await expect(page.getByRole("link", { name: "Kembali ke halaman awal" })).toBeVisible()
    await page.getByRole("link", { name: "Kembali ke halaman awal" }).click()
    await expect(page).not.toHaveURL(/cursor=/)
    const selectedCategory = String(publicItems[0]?.category && (publicItems[0].category as Record<string, unknown>).id)
    await page.locator("#f-category").selectOption(selectedCategory)
    await page.getByRole("button", { name: "Terapkan filter" }).click()
    await expect(page).toHaveURL(new RegExp(`category_id=${selectedCategory}`))
    await expect(page.locator("#f-category")).toHaveValue(selectedCategory)

    await page.setViewportSize({ width: 360, height: 800 })
    await page.goto("/lapor/hilang")
    await expect(page.getByRole("heading", { name: "Lapor kehilangan" })).toBeVisible()
    await captureB04(page, "lapor-hilang", "360x800")
    await page.getByRole("button", { name: "Lewati, saya tidak punya foto" }).click()
    await page.getByRole("combobox").click()
    await page.getByPlaceholder("Cari kategori, misalnya dompet").fill("tumbler")
    await page.getByRole("option", { name: "Tumbler" }).click()
    const reportName = `Tumbler biru draf ${Date.now()}`
    await page.getByLabel("Nama barang").fill(reportName)
    await page.getByLabel("Deskripsi").fill("Tumbler biru 500 ml dengan tutup hitam dan stiker kecil di sisi.")
    await page.reload()
    await expect(page.getByText("Draf sebelumnya dipulihkan.")).toBeVisible()
    await page.getByRole("button", { name: "Lewati, saya tidak punya foto" }).click()
    await expect(page.getByLabel("Nama barang")).toHaveValue(reportName)
    await expect(page.getByLabel("Deskripsi")).toHaveValue("Tumbler biru 500 ml dengan tutup hitam dan stiker kecil di sisi.")
    await captureB04(page, "lapor-hilang", "360x800")
    await page.setViewportSize({ width: 1366, height: 900 })
    await captureB04(page, "lapor-hilang", "1366x900")
    await page.getByRole("button", { name: "Lanjut" }).click()
    await page.getByRole("checkbox", { name: "Saya tidak tahu lokasinya" }).check()
    await page.getByRole("button", { name: "Lanjut" }).click()
    await page.getByRole("button", { name: "Lanjut" }).click()
    await expect(page.getByRole("heading", { name: "Tinjau" })).toBeVisible()
    await expect(page.getByText("Tidak ada foto", { exact: true })).toBeVisible()
    await captureB04(page, "lapor-hilang-tinjau", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB04(page, "lapor-hilang-tinjau", "360x800")
    await page.setViewportSize({ width: 1366, height: 900 })
    await page.getByRole("button", { name: "Kirim laporan" }).click()
    await expect(page).toHaveURL(/\/laporan\/hilang\/LST-\d{4}-\d{4}$/)
    await expect(page.getByRole("heading", { name: reportName })).toBeVisible()

    expect(external).toEqual([])
    expect(errors).toEqual([])
  })
})

async function captureB05(page: Page, route: string, width: string) {
  await expect(page.getByRole("main")).toBeVisible()
  await page.evaluate(async () => {
    await document.fonts.ready
    const finiteAnimations = document.getAnimations().filter((animation) => animation.effect?.getComputedTiming().iterations !== Infinity)
    await Promise.all(finiteAnimations.map((animation) => animation.finished.catch(() => undefined)))
  })
  await expectAccessible(page)
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(await page.evaluate(() => window.innerWidth))
  await mkdir("test-results/screenshots", { recursive: true })
  await page.screenshot({
    path: `test-results/screenshots/phase2-B05-${route}-${width}.png`,
    fullPage: true,
    animations: "disabled",
  })
}

async function signInDemo(page: Page, role: "Pemilik (Mahasiswa)" | "Penemu (Mahasiswa)") {
  await page.goto("/masuk")
  await page.getByRole("button", { name: role === "Pemilik (Mahasiswa)" ? /Pemilik \(Mahasiswa\)/ : /Penemu \(Mahasiswa\)/ }).click()
  await expect(page).toHaveURL(/\/beranda$/)
}

test.describe("B05 member report details and editing", () => {
  test("owner and finder details retain private evidence and protected photo zoom", async ({ page }) => {
    test.setTimeout(150_000)
    await page.setViewportSize({ width: 1366, height: 900 })
    const external: string[] = []
    const errors: string[] = []
    page.on("request", (request) => {
      const url = new URL(request.url())
      if (/^https?:$/.test(url.protocol) && !/^(localhost|127\.0\.0\.1|\[::1\])$/.test(url.hostname)) {
        external.push(request.url())
      }
    })
    page.on("pageerror", (error) => errors.push(error.message))

    await signInDemo(page, "Pemilik (Mahasiswa)")
    await page.goto("/laporan")
    const lostLink = page.getByRole("link", { name: /TWS Earbuds Jabra Elite 4 Active/ })
    await expect(lostLink).toBeVisible()
    await lostLink.click()
    const lostCode = await page.locator("code").first().innerText()
    expect(lostCode).toMatch(/^LST-\d{4}-\d{4}$/)
    expect(lostCode).toBeTruthy()
    await expect(page.getByRole("heading", { name: "TWS Earbuds Jabra Elite 4 Active" })).toBeVisible()
    await expect(page.locator('section[aria-labelledby="ciri"]')).toContainText("SN-JBR4-9821")
    await expect(page.getByRole("heading", { name: "Kandidat barang temuan" })).toBeVisible()
    await captureB05(page, "detail-hilang", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB05(page, "detail-hilang", "360x800")
    const lostPhoto = page.locator('button:has(img[alt^="Foto barang hilang"])').first()
    await expect(lostPhoto).toBeVisible()
    await lostPhoto.click()
    await expect(page.getByRole("dialog")).toBeVisible()
    await expect(page.getByRole("dialog").locator("img")).toBeVisible()
    await captureB05(page, "zoom-hilang", "360x800")
    await page.keyboard.press("Escape")
    await expect(page.getByRole("dialog")).toHaveCount(0)
    await page.setViewportSize({ width: 1366, height: 900 })
    await lostPhoto.click()
    await expect(page.getByRole("dialog").locator("img")).toBeVisible()
    await captureB05(page, "zoom-hilang", "1366x900")
    await page.keyboard.press("Escape")

    await page.context().clearCookies()
    await signInDemo(page, "Penemu (Mahasiswa)")
    await page.goto("/laporan?tab=temuan")
    const foundLink = page.getByRole("link", { name: /Payung Lipat Hitam Otomatis/ })
    await expect(foundLink).toBeVisible()
    await foundLink.click()
    await expect(page.getByRole("heading", { name: "Payung Lipat Hitam Otomatis" })).toBeVisible()
    await expect(page.locator('section[aria-labelledby="tersembunyi"]')).toContainText("jahitan manual benang biru")
    const privateResponse = await page.request.get(new URL(`/api/v1/lost-reports/${lostCode}`, page.url()).toString())
    expect([401, 403, 404]).toContain(privateResponse.status())
    expect(await privateResponse.text()).not.toContain("SN-JBR4-9821")
    await captureB05(page, "detail-temuan", "360x800")
    await page.setViewportSize({ width: 1366, height: 900 })
    await captureB05(page, "detail-temuan", "1366x900")
    const foundPhoto = page.locator('button:has(img[alt^="Foto barang temuan"])').first()
    await expect(foundPhoto).toBeVisible()
    await foundPhoto.click()
    await expect(page.getByRole("dialog").locator("img")).toBeVisible()
    await captureB05(page, "zoom-temuan", "1366x900")
    await page.keyboard.press("Escape")
    await expect(page.getByRole("dialog")).toHaveCount(0)
    await page.setViewportSize({ width: 360, height: 800 })
    await foundPhoto.click()
    await expect(page.getByRole("dialog").locator("img")).toBeVisible()
    await captureB05(page, "zoom-temuan", "360x800")
    await page.keyboard.press("Escape")

    expect(external).toEqual([])
    expect(errors).toEqual([])
  })

  test("a concurrent edit preserves the owner's values and close still requires a reason", async ({ page, browser }) => {
    test.setTimeout(180_000)
    page.setDefaultTimeout(10_000)
    await page.setViewportSize({ width: 360, height: 800 })
    const external: string[] = []
    const errors: string[] = []
    page.on("request", (request) => {
      const url = new URL(request.url())
      if (/^https?:$/.test(url.protocol) && !/^(localhost|127\.0\.0\.1|\[::1\])$/.test(url.hostname)) {
        external.push(request.url())
      }
    })
    page.on("pageerror", (error) => errors.push(error.message))
    const email = uniqueEmail("phase2-b05")
    const password = "kata sandi panjang fase 2"
    await registerMember(page, "Anggota Uji B05", email, password)
    await signIn(page, email, password)
    await expect(page).toHaveURL(/\/beranda$/)
    await page.goto("/lapor/hilang")
    const skipPhoto = page.getByRole("button", { name: "Lewati, saya tidak punya foto" })
    await expect(skipPhoto).toBeVisible()
    await skipPhoto.click()
    await page.getByRole("combobox").click()
    await page.getByPlaceholder("Cari kategori, misalnya dompet").fill("tumbler")
    await page.getByRole("option", { name: "Tumbler" }).click()
    const reportName = `Tumbler B05 ${Date.now()}`
    await page.getByLabel("Nama barang").fill(reportName)
    await page.getByLabel("Deskripsi").fill("Tumbler biru 500 ml dengan tutup hitam dan stiker kecil.")
    await page.getByRole("button", { name: "Lanjut" }).click()
    await page.getByRole("checkbox", { name: "Saya tidak tahu lokasinya" }).check()
    await page.getByRole("button", { name: "Lanjut" }).click()
    await page.getByRole("textbox", { name: /^Ciri khusus/ }).fill("Tanda pemilik B05: tutup bagian dalam berwarna hijau.")
    await page.getByRole("button", { name: "Lanjut" }).click()
    await page.getByRole("button", { name: "Kirim laporan" }).click()
    await expect(page).toHaveURL(/\/laporan\/hilang\/LST-\d{4}-\d{4}$/)
    const detailUrl = page.url()
    const code = /LST-\d{4}-\d{4}$/.exec(detailUrl)?.[0]
    expect(code).toBeTruthy()
    await expect(page.locator('section[aria-labelledby="ciri"]')).toContainText("Tanda pemilik B05")
    await captureB05(page, "detail-baru", "360x800")
    await page.setViewportSize({ width: 1366, height: 900 })
    await captureB05(page, "detail-baru", "1366x900")

    const editUrl = `${detailUrl}/ubah`
    await page.goto(editUrl)
    await expect(page.getByRole("heading", { name: "Ubah laporan kehilangan" })).toBeVisible()
    await captureB05(page, "ubah-hilang", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB05(page, "ubah-hilang", "360x800")
    await page.setViewportSize({ width: 1366, height: 900 })

    async function prepareEdit(target: Page, description: string) {
      await target.getByRole("button", { name: "Lanjut" }).click()
      await target.getByLabel("Deskripsi").fill(description)
      await target.getByRole("button", { name: "Lanjut" }).click()
      await target.getByRole("button", { name: "Lanjut" }).click()
      await target.getByRole("textbox", { name: /^Ciri khusus/ }).fill("Tanda pemilik B05: tutup bagian dalam berwarna hijau.")
      await target.getByRole("button", { name: "Lanjut" }).click()
      await expect(target.getByRole("heading", { name: "Tinjau" })).toBeVisible()
    }

    const ownerVersion = await page.context().newPage()
    await ownerVersion.goto(editUrl)
    await prepareEdit(page, "Versi lama yang tetap terlihat setelah konflik penyuntingan.")
    await prepareEdit(ownerVersion, "Versi baru yang disimpan oleh tab lain sebelum konflik.")
    await ownerVersion.getByRole("button", { name: "Simpan perubahan" }).click()
    await expect(ownerVersion).toHaveURL(new RegExp(`/laporan/hilang/${code}$`))
    await page.getByRole("button", { name: "Simpan perubahan" }).click()
    await expect(page.getByText("Data sudah diubah oleh orang lain.", { exact: false })).toBeVisible()
    await expect(
      page.getByRole("region", { name: "Tinjau" }).getByText("Versi lama yang tetap terlihat setelah konflik penyuntingan.", { exact: true }),
    ).toBeVisible()
    await expectAccessible(page)

    await ownerVersion.close()
    await page.goto(detailUrl)
    await page.getByRole("button", { name: "Tutup laporan", exact: true }).click()
    const closeDialog = page.getByRole("alertdialog")
    await expect(closeDialog).toBeVisible()
    await expectAccessible(page)
    await expect(closeDialog.getByRole("radio", { name: "Sudah ditemukan sendiri" })).toBeChecked()
    await closeDialog.getByRole("button", { name: "Ya, tutup laporan", exact: true }).click()
    await expect(page.getByText("Ditutup", { exact: true })).toBeVisible()

    const otherContext = await browser.newContext()
    const otherPage = await otherContext.newPage()
    await otherPage.goto("http://localhost:8000/masuk")
    await otherPage.getByRole("button", { name: "Penemu (Mahasiswa)" }).click()
    await expect(otherPage).toHaveURL(/\/beranda$/)
    const denied = await otherPage.request.get(new URL(`/api/v1/lost-reports/${code}`, otherPage.url()).toString())
    expect([401, 403, 404]).toContain(denied.status())
    expect(await denied.text()).not.toContain("Tanda pemilik B05")
    await otherContext.close()

    expect(external).toEqual([])
    expect(errors).toEqual([])
  })
})

async function captureB06(page: Page, route: string, width: string, target?: Locator) {
  await page.evaluate(() => window.scrollTo(0, 0))
  if (target) {
    await target.evaluate((element) => element.scrollIntoView({ block: "center" }))
    await page.evaluate(() => window.scrollBy(0, -25))
    const nav = page.locator("nav.fixed").last()
    const [targetBox, navBox] = await Promise.all([
      target.evaluate((element) => {
        const { bottom, top } = element.getBoundingClientRect()
        return { bottom, top }
      }),
      nav.evaluate((element) => {
        const { bottom, top } = element.getBoundingClientRect()
        return { bottom, top }
      }),
    ])
    expect(targetBox.bottom).toBeLessThanOrEqual(navBox.top)
  }
  await expect(page.getByRole("main")).toBeVisible()
  await page.evaluate(async () => {
    await document.fonts.ready
    const finiteAnimations = document.getAnimations().filter((animation) => animation.effect?.getComputedTiming().iterations !== Infinity)
    await Promise.all(finiteAnimations.map((animation) => animation.finished.catch(() => undefined)))
  })
  await expectAccessible(page)
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(await page.evaluate(() => window.innerWidth))
  await mkdir("test-results/screenshots", { recursive: true })
  await page.screenshot({ path: `test-results/screenshots/phase2-B06-${route}-${width}.png`, animations: "disabled" })
}

test.describe("B06 member claims and notifications", () => {
  test("claims retain secure state and notification links mark items read", async ({ page }) => {
    test.setTimeout(180_000)
    page.setDefaultTimeout(10_000)
    await page.setViewportSize({ width: 1366, height: 900 })
    const external: string[] = []
    const errors: string[] = []
    page.on("request", (request) => {
      const url = new URL(request.url())
      if (/^https?:$/.test(url.protocol) && !/^(localhost|127\.0\.0\.1|\[::1\])$/.test(url.hostname)) external.push(request.url())
    })
    page.on("pageerror", (error) => errors.push(error.message))

    await signInDemo(page, "Pemilik (Mahasiswa)")
    const claimsResponse = await page.request.get(new URL("/api/v1/claims?limit=20", page.url()).toString())
    expect(claimsResponse.ok()).toBe(true)
    const claimSummaries = (await claimsResponse.json()).items as { code: string; status: string }[]
    expect(claimSummaries.length).toBeGreaterThan(0)
    const selected = claimSummaries.find((claim) => claim.status === "approved") ?? claimSummaries[0]
    if (!selected) throw new Error("the demo owner needs a seeded claim")
    const claimResponse = await page.request.get(new URL(`/api/v1/claims/${selected.code}`, page.url()).toString())
    expect(claimResponse.ok()).toBe(true)
    const claim = (await claimResponse.json()) as {
      code: string
      status: string
      found_report: { id: string; item_name: string; category: { id: string } }
      lost_report: { code: string }
      handover_code?: string | null
    }

    await page.goto("/klaim")
    await expect(page.getByRole("heading", { name: "Klaim Saya" })).toBeVisible()
    await expect(page.getByRole("link", { name: new RegExp(claim.code) })).toBeVisible()
    await captureB06(page, "klaim", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB06(page, "klaim", "360x800")
    await page.keyboard.press("Tab")
    await expect(page.locator(":focus")).toBeVisible()
    await page.getByRole("link", { name: new RegExp(claim.code) }).click()
    await expect(page).toHaveURL(new RegExp(`/klaim/${claim.code}$`))
    await expect(page.getByRole("heading", { name: claim.code })).toBeVisible()
    if (claim.status === "approved") {
      await expect(page.locator("#kode-serah")).toBeVisible()
      expect(claim.handover_code).toMatch(/^\d{6}$/)
    } else {
      await expect(page.locator("#kode-serah")).toHaveCount(0)
    }
    await expect(page.getByText("Ciri khusus pemilik", { exact: true })).toHaveCount(0)
    await captureB06(page, "detail-klaim", "360x800", page.locator("#claim-thread-body"))
    await page.setViewportSize({ width: 1366, height: 900 })
    await captureB06(page, "detail-klaim", "1366x900")

    const params = new URLSearchParams({
      temuan: claim.found_report.id,
      nama: claim.found_report.item_name,
      kategori: claim.found_report.category.id,
      hilang: claim.lost_report.code,
    })
    await page.goto(`/klaim/baru?${params.toString()}`)
    await expect(page.getByRole("heading", { name: "Ajukan Klaim" })).toBeVisible()
    await expect(page.getByRole("button", { name: "Kirim Pengajuan Klaim" })).toBeVisible()
    await captureB06(page, "form-klaim", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB06(page, "form-klaim", "360x800", page.getByRole("button", { name: "Tambah foto" }))

    const unreadResponse = await page.request.get(new URL("/api/v1/notifications?unread=true&limit=20", page.url()).toString())
    expect(unreadResponse.ok()).toBe(true)
    const unread = (await unreadResponse.json()).items as { id: string; link: string; read_at: string | null }[]
    expect(unread.length).toBeGreaterThan(0)
    const firstUnread = unread[0]
    if (!firstUnread) throw new Error("the demo owner needs an unread notification")
    await page.goto("/notifikasi")
    await expect(page.getByRole("heading", { name: "Notifikasi", level: 1 })).toBeVisible()
    await captureB06(page, "notifikasi", "360x800")
    await page.setViewportSize({ width: 1366, height: 900 })
    await captureB06(page, "notifikasi", "1366x900")
    const unreadFilter = page.getByRole("group", { name: "Notifikasi" }).getByRole("button", { name: "Belum dibaca" })
    await unreadFilter.click()
    const firstNotification = page.getByRole("main").getByRole("link").first()
    await expect(firstNotification).toBeVisible()
    const target = await firstNotification.getAttribute("href")
    expect(target).toMatch(/^\//)
    const markedRead = page.waitForResponse(
      (response) => response.request().method() === "POST" && response.url().endsWith(`/notifications/${firstUnread.id}/read`),
    )
    await firstNotification.click()
    expect((await markedRead).ok()).toBe(true)
    await expect(page).toHaveURL(new RegExp(`${target?.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}$`))
    const afterReadResponse = await page.request.get(new URL("/api/v1/notifications?limit=20", page.url()).toString())
    expect(afterReadResponse.ok()).toBe(true)
    const afterRead = (await afterReadResponse.json()).items as { id: string; read_at: string | null }[]
    expect(afterRead.find((item) => item.id === firstUnread.id)?.read_at).toBeTruthy()

    await page.goto("/notifikasi")
    const markedAllRead = page.waitForResponse((response) => response.request().method() === "POST" && response.url().endsWith("/notifications/read-all"))
    await page.getByRole("button", { name: "Tandai semua dibaca" }).click()
    expect((await markedAllRead).ok()).toBe(true)
    await page.getByRole("group", { name: "Notifikasi" }).getByRole("button", { name: "Belum dibaca" }).click()
    await expect(page.getByText("Semua notifikasi sudah dibaca.")).toBeVisible()
    expect(external).toEqual([])
    expect(errors).toEqual([])
  })
})

async function captureB07(page: Page, route: string, width: string, target?: Locator) {
  await page.evaluate(() => window.scrollTo(0, 0))
  if (target) {
    await target.evaluate((element) => element.scrollIntoView({ block: "center" }))
    await page.evaluate(() => window.scrollBy(0, -25))
    const nav = page.locator("nav.fixed").last()
    const [targetBox, navBox] = await Promise.all([
      target.evaluate((element) => {
        const { bottom } = element.getBoundingClientRect()
        return { bottom }
      }),
      nav.evaluate((element) => {
        const { top } = element.getBoundingClientRect()
        return { top }
      }),
    ])
    expect(targetBox.bottom).toBeLessThanOrEqual(navBox.top)
  }
  await expect(page.getByRole("main")).toBeVisible()
  await expectAccessible(page)
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(await page.evaluate(() => window.innerWidth))
  await mkdir("test-results/screenshots", { recursive: true })
  await page.screenshot({ path: `test-results/screenshots/phase2-B07-${route}-${width}.png`, animations: "disabled" })
}

test.describe("B07 member profile settings", () => {
  test("profile, security and notification preferences retain their real actions and confirmations", async ({ page, browser }) => {
    test.setTimeout(180_000)
    page.setDefaultTimeout(10_000)
    await page.setViewportSize({ width: 1366, height: 900 })
    const external: string[] = []
    const errors: string[] = []
    page.on("request", (request) => {
      const url = new URL(request.url())
      if (/^https?:$/.test(url.protocol) && !/^(localhost|127\.0\.0\.1|\[::1\])$/.test(url.hostname)) external.push(request.url())
    })
    page.on("pageerror", (error) => errors.push(error.message))

    await signInDemo(page, "Pemilik (Mahasiswa)")
    await page.goto("/profil")
    await expect(page.getByRole("heading", { name: "Profil", level: 1 })).toBeVisible()
    await captureB07(page, "profil", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB07(page, "profil", "360x800", page.getByRole("button", { name: "Simpan profil" }))
    await page.setViewportSize({ width: 1366, height: 900 })

    const profileSave = page.waitForResponse((response) => response.request().method() === "PATCH" && response.url().endsWith("/me"))
    await page.getByLabel("Nama").fill(`Siti Rahma B07 ${Date.now()}`)
    await page.getByLabel("Telepon").fill("081234567800")
    await page.getByLabel("NIM atau NIP").fill("24/543770/PA/23107")
    await page.getByRole("button", { name: "Simpan profil" }).click()
    expect((await profileSave).ok()).toBe(true)
    await expect(page.getByText("Profil disimpan.")).toBeVisible()

    await page.getByRole("button", { name: "Ganti email" }).click()
    const emailDialog = page.getByRole("dialog")
    await expect(emailDialog.getByRole("heading", { name: "Ganti email akun" })).toBeVisible()
    const sendEmail = emailDialog.getByRole("button", { name: "Kirim tautan konfirmasi" })
    await expect(sendEmail).toBeDisabled()
    await emailDialog.getByLabel("Email baru").fill("bukan-email")
    await emailDialog.getByLabel("Kata sandi sekarang").fill("sandi-uji-yang-salah")
    await sendEmail.click()
    await expect(emailDialog).toBeVisible()
    await expect(emailDialog.getByText("Jika alamat itu bisa dipakai, tautan konfirmasi sudah dikirim ke sana. Berlaku 24 jam.")).toHaveCount(0)
    await emailDialog.getByRole("button", { name: "Batal" }).click()

    await page.goto("/profil/keamanan")
    await expect(page.getByRole("heading", { name: "Keamanan", level: 1 })).toBeVisible()
    await expect(page.getByText("Perangkat ini", { exact: true })).toBeVisible()
    await captureB07(page, "keamanan", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB07(page, "keamanan", "360x800")
    await page.setViewportSize({ width: 1366, height: 900 })

    await page.getByLabel("Kata sandi sekarang").fill("sandi-uji-yang-salah")
    await page.getByLabel("Kata sandi baru", { exact: true }).fill("singkat")
    await page.getByLabel("Ulangi kata sandi baru", { exact: true }).fill("tidak-sama")
    await page.getByRole("button", { name: "Simpan kata sandi" }).click()
    await expect(page.getByText(/Minimal 10 karakter, saat ini/)).toBeVisible()
    await expect(page.getByText("Kata sandi tidak sama")).toBeVisible()

    const extraSession = await browser.newPage()
    await signInDemo(extraSession, "Pemilik (Mahasiswa)")
    await extraSession.goto("/profil/keamanan")
    await page.goto("/profil/keamanan")
    const revokeButtons = page.getByRole("button", { name: "Keluarkan" })
    await expect(revokeButtons.first()).toBeVisible()
    await revokeButtons.first().click()
    await expect(page.getByText("Perangkat dikeluarkan.")).toBeVisible()
    await extraSession.close()

    await page.getByRole("button", { name: "Keluar dari semua perangkat" }).click()
    const signOutDialog = page.getByRole("alertdialog")
    await expect(signOutDialog.getByRole("heading", { name: "Keluar dari semua perangkat?" })).toBeVisible()
    await signOutDialog.getByRole("button", { name: "Batal" }).click()
    await expect(page).toHaveURL(/\/profil\/keamanan$/)

    await page.getByRole("button", { name: "Hapus Akunku" }).click()
    const deleteDialog = page.getByRole("dialog")
    await expect(deleteDialog.getByRole("heading", { name: "Yakin ingin menghapus akun?" })).toBeVisible()
    await expect(deleteDialog.getByText("Tindakan ini permanen dan tidak dapat dibatalkan.")).toBeVisible()
    await deleteDialog.getByRole("button", { name: "Batal" }).click()
    await expect(page).toHaveURL(/\/profil\/keamanan$/)

    await page.goto("/profil/notifikasi")
    await expect(page.getByRole("heading", { name: "Pengaturan notifikasi", level: 1 })).toBeVisible()
    await captureB07(page, "notifikasi", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB07(page, "notifikasi", "360x800", page.getByRole("button", { name: "Simpan" }))
    await page.setViewportSize({ width: 1366, height: 900 })

    const beforeResponse = await page.request.get(new URL("/api/v1/me/notification-preferences", page.url()).toString())
    expect(beforeResponse.ok()).toBe(true)
    const beforeItems = (await beforeResponse.json()).items as { group: string; app: boolean; email: boolean }[]
    const firstPreference = beforeItems[0]
    if (!firstPreference) throw new Error("the demo account needs notification preferences")
    const firstCheckbox = page.getByRole("checkbox").first()
    const originalValue = await firstCheckbox.isChecked()
    await firstCheckbox.click()
    const firstSave = page.waitForResponse((response) => response.request().method() === "PUT" && response.url().endsWith("/me/notification-preferences"))
    await page.getByRole("button", { name: "Simpan" }).click()
    expect((await firstSave).ok()).toBe(true)
    const changedResponse = await page.request.get(new URL("/api/v1/me/notification-preferences", page.url()).toString())
    const changedItems = (await changedResponse.json()).items as { group: string; app: boolean; email: boolean }[]
    expect(changedItems.find((item) => item.group === firstPreference.group)?.app).toBe(!originalValue)
    await firstCheckbox.click()
    const restoreSave = page.waitForResponse((response) => response.request().method() === "PUT" && response.url().endsWith("/me/notification-preferences"))
    await page.getByRole("button", { name: "Simpan" }).click()
    expect((await restoreSave).ok()).toBe(true)
    const restoredResponse = await page.request.get(new URL("/api/v1/me/notification-preferences", page.url()).toString())
    const restoredItems = (await restoredResponse.json()).items as { group: string; app: boolean; email: boolean }[]
    expect(restoredItems.find((item) => item.group === firstPreference.group)?.app).toBe(originalValue)

    expect(external).toEqual([])
    expect(errors).toEqual([])
  })
})

async function captureB08(page: Page, route: string, width: string) {
  await page.evaluate(() => window.scrollTo(0, 0))
  await expect(page.getByRole("main")).toBeVisible()
  await expectAccessible(page)
  await mkdir("test-results/screenshots", { recursive: true })
  await page.screenshot({ path: `test-results/screenshots/phase2-B08-${route}-${width}.png`, fullPage: true, animations: "disabled" })
  const overflow = await page.evaluate(() =>
    Array.from(document.body.querySelectorAll("*"))
      .filter((element) => {
        const rect = element.getBoundingClientRect()
        return rect.width > 0 && (rect.right > window.innerWidth + 1 || element.scrollWidth > element.clientWidth + 1)
      })
      .slice(0, 12)
      .map((element) => ({
        tag: element.tagName,
        className: (element as HTMLElement).className,
        right: Math.round(element.getBoundingClientRect().right),
        width: element.clientWidth,
        scrollWidth: element.scrollWidth,
      })),
  )
  expect(await page.evaluate(() => document.documentElement.scrollWidth), JSON.stringify(overflow)).toBeLessThanOrEqual(
    await page.evaluate(() => window.innerWidth),
  )
}

test.describe("B08 officer review and search workspaces", () => {
  test("priority queue, report-code review, semantic search and duplicate controls work at both widths", async ({ page }) => {
    test.setTimeout(180_000)
    page.setDefaultTimeout(10_000)
    await page.setViewportSize({ width: 1366, height: 900 })
    const external: string[] = []
    const errors: string[] = []
    page.on("request", (request) => {
      const url = new URL(request.url())
      if (/^https?:$/.test(url.protocol) && !/^(localhost|127\.0\.0\.1|\[::1\])$/.test(url.hostname)) external.push(request.url())
    })
    page.on("pageerror", (error) => errors.push(error.message))

    await signIn(page, "petugas.pos1@demo.simpanugm.my.id", "kata sandi panjang 2026")
    await expect(page).toHaveURL(/\/petugas$/)
    await expect(page.getByRole("heading", { name: "Antrean Prioritas Kerja" })).toBeVisible()
    await captureB08(page, "antrean", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB08(page, "antrean", "360x800")

    const queueResponse = await page.request.get(new URL("/api/v1/officer/queue?scope=posts&limit=30", page.url()).toString())
    expect(queueResponse.ok()).toBe(true)
    const queue = (await queueResponse.json()).items as { type: string; found_id: string; found_code: string }[]
    const highMatch = queue.find((item) => item.type === "high_match_no_claim")
    if (!highMatch) throw new Error("the seeded officer queue needs a high-match found report")

    await page.emulateMedia({ reducedMotion: "reduce" })
    await page.setViewportSize({ width: 1366, height: 900 })
    await page.goto(`/petugas/papan?report_type=found&report_id=${encodeURIComponent(highMatch.found_code)}`)
    await expect(page.getByRole("heading", { name: "Papan Tinjauan" })).toBeVisible()
    await expect(page.getByRole("heading", { name: /\d+ kandidat pencocokan/ })).toBeVisible()
    await expect(page.getByRole("button", { name: "Cocokkan ulang" })).toBeVisible()
    const meters = page.getByRole("meter")
    expect(await meters.count()).toBeGreaterThan(0)
    expect((await meters.count()) % 4).toBe(0)
    await expect(page.getByRole("button", { name: /Undang klaim|Pemilik sudah diundang/ }).first()).toBeVisible()
    const pinButton = page.getByRole("button", { name: /Tandai kuat|Lepas tanda/ }).first()
    await expect(pinButton).toBeVisible()
    const pinLabel = await pinButton.innerText()
    await pinButton.click()
    const toggledLabel = pinLabel === "Tandai kuat" ? "Lepas tanda" : "Tandai kuat"
    await expect(page.getByRole("button", { name: toggledLabel }).first()).toBeVisible()
    await page.getByRole("button", { name: toggledLabel }).first().click()
    await expect(page.getByRole("button", { name: pinLabel }).first()).toBeVisible()
    await captureB08(page, "papan", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB08(page, "papan", "360x800")

    await page.setViewportSize({ width: 1366, height: 900 })
    await page.goto("/petugas/cari")
    const search = page.getByRole("searchbox")
    await search.fill("dompet kulit")
    const semantic = page.getByRole("checkbox", { name: "Cari berdasarkan makna" })
    await semantic.check()
    await search.press("Enter")
    await expect(page).toHaveURL(/semantic=true/)
    await expect(page.getByRole("status").filter({ hasText: /\d+ hasil/ })).toBeVisible()
    await captureB08(page, "cari", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB08(page, "cari", "360x800")

    await page.goto("/petugas/duplikat")
    await expect(page.getByRole("heading", { name: "Duplikat Laporan" })).toBeVisible()
    await expect(page.getByRole("link", { name: "Disarankan" })).toBeVisible()
    const manualMerge = page.getByRole("button", { name: "Gabung Laporan Manual" })
    await expect(manualMerge).toBeVisible()
    await expect(page.getByRole("button", { name: "Bukan Duplikat" })).toBeVisible()
    await expect(page.getByRole("button", { name: "Gabungkan" })).toBeVisible()
    await captureB08(page, "duplikat", "360x800")
    await page.setViewportSize({ width: 1366, height: 900 })
    await manualMerge.click()
    await expect(page.getByLabel("Kode Laporan Bertahan (UUID)")).toBeVisible()
    await expect(page.getByLabel("Kode Laporan Duplikat (UUID)")).toBeVisible()
    await captureB08(page, "duplikat", "1366x900")

    expect(external).toEqual([])
    expect(errors).toEqual([])
  })
})

async function captureB09(page: Page, route: string, width: string) {
  await page.evaluate(() => window.scrollTo(0, 0))
  await expect(page.getByRole("main")).toBeVisible()
  await expectAccessible(page)
  await mkdir("test-results/screenshots", { recursive: true })
  await page.screenshot({ path: `test-results/screenshots/phase2-B09-${route}-${width}.png`, fullPage: true, animations: "disabled" })
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(await page.evaluate(() => window.innerWidth))
}

test.describe("B09 officer custody workspaces", () => {
  test("manual custody lookup, inventory filters and guarded custody actions work at both widths", async ({ page }) => {
    test.setTimeout(120_000)
    page.setDefaultTimeout(10_000)
    const external: string[] = []
    const errors: string[] = []
    page.on("request", (request) => {
      const url = new URL(request.url())
      if (/^https?:$/.test(url.protocol) && !/^(localhost|127\.0\.0\.1|\[::1\])$/.test(url.hostname)) external.push(request.url())
    })
    page.on("pageerror", (error) => errors.push(error.message))

    await signIn(page, "petugas.pos1@demo.simpanugm.my.id", "kata sandi panjang 2026")
    await expect(page).toHaveURL(/\/petugas$/)
    await page.setViewportSize({ width: 1366, height: 900 })
    await page.goto("/petugas/terima")
    await expect(page.getByRole("heading", { name: "Terima Barang" })).toBeVisible()
    const code = page.getByLabel("Kode laporan temuan")
    await code.fill("FND-2610-9999")
    await page.getByRole("button", { name: "Periksa" }).click()
    await expect(page.getByText("Laporan temuan dengan kode ini tidak ditemukan.")).toBeVisible()
    await captureB09(page, "terima", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB09(page, "terima", "360x800")

    const foundResponse = await page.request.get(new URL("/api/v1/found-reports?limit=100", page.url()).toString())
    expect(foundResponse.ok()).toBe(true)
    const foundItems = (await foundResponse.json()).items as { code: string; item_name: string; custody_status: string }[]
    const awaitingReceipt = foundItems.find((item) => item.custody_status === "with_finder")
    if (awaitingReceipt) {
      await page.goto(`/petugas/terima?code=${encodeURIComponent(awaitingReceipt.code)}`)
      await expect(page.getByRole("heading", { name: awaitingReceipt.item_name })).toBeVisible()
      const slot = page.getByLabel("Rak atau slot")
      await page.getByRole("button", { name: "Terima barang" }).click()
      await expect(slot).toHaveAttribute("aria-invalid", "true")
      await slot.fill("Rak B09")
      await page.getByRole("button", { name: "Terima barang" }).click()
      await page.getByRole("alertdialog").getByRole("button", { name: "Ya, terima barang" }).click()
      await expect(page.getByRole("status").filter({ hasText: "Barang diterima di" })).toBeVisible()
    }

    await page.setViewportSize({ width: 1366, height: 900 })
    await page.goto("/petugas/inventaris")
    await expect(page.getByRole("heading", { name: "Inventaris" })).toBeVisible()
    await expect(page.getByRole("region", { name: "Daftar barang di inventaris pos" })).toBeVisible()
    if (awaitingReceipt) await expect(page.getByRole("link", { name: new RegExp(awaitingReceipt.code) })).toBeVisible()
    const sort = page.getByLabel("Urutkan")
    await sort.selectOption("-received_at")
    await page.getByRole("button", { name: "Terapkan" }).click()
    await expect(page).toHaveURL(/sort=-received_at/)
    await expect(page.getByRole("button", { name: "Perpanjang simpan" }).first()).toBeVisible()
    const extend = page.getByRole("button", { name: "Perpanjang simpan" }).first()
    await extend.click()
    await expect(page.getByRole("heading", { name: "Perpanjang masa simpan" })).toBeVisible()
    await expect(page.getByLabel("Tambahan hari (1 sampai 30)")).toHaveValue("7")
    await expect(page.getByRole("button", { name: "Perpanjang masa simpan" })).toBeDisabled()
    await page.getByRole("button", { name: "Batal" }).click()
    const transfer = page.getByRole("button", { name: "Pindah pos" }).first()
    await expect(transfer).toBeVisible()
    await expect(transfer).toBeEnabled()
    await transfer.click()
    await expect(page.getByRole("heading", { name: "Pindahkan barang ke pos lain" })).toBeVisible()
    const destination = page.getByLabel("Pos tujuan")
    await expect(destination.locator("option")).toHaveCount(6)
    await destination.selectOption({ index: 1 })
    await expect(page.getByRole("button", { name: /Pindahkan ke/ })).toBeEnabled()
    await page.getByRole("button", { name: "Batal" }).click()
    await captureB09(page, "inventaris", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB09(page, "inventaris", "360x800")

    expect(external).toEqual([])
    expect(errors).toEqual([])
  })
})

async function captureB10(page: Page, route: string, width: string) {
  await page.evaluate(() => window.scrollTo(0, 0))
  await expect(page.getByRole("main")).toBeVisible()
  await expectAccessible(page)
  await mkdir("test-results/screenshots", { recursive: true })
  await page.screenshot({ path: `test-results/screenshots/phase2-B10-${route}-${width}.png`, fullPage: true, animations: "disabled" })
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(await page.evaluate(() => window.innerWidth))
}

test.describe("B10 officer claim verification and handover", () => {
  test("under-review claim evidence, actions and thread stay clear at both widths", async ({ page }) => {
    test.setTimeout(120_000)
    page.setDefaultTimeout(10_000)
    const external: string[] = []
    const errors: string[] = []
    page.on("request", (request) => {
      const url = new URL(request.url())
      if (/^https?:$/.test(url.protocol) && !/^(localhost|127\.0\.0\.1|\[::1\])$/.test(url.hostname)) external.push(request.url())
    })
    page.on("pageerror", (error) => errors.push(error.message))

    await signIn(page, "petugas.pos1@demo.simpanugm.my.id", "kata sandi panjang 2026")
    await expect(page).toHaveURL(/\/petugas$/)
    await page.setViewportSize({ width: 1366, height: 900 })
    await page.goto("/petugas/klaim")
    await expect(page.getByRole("heading", { name: "Antrean Klaim" })).toBeVisible()
    const reviewResponse = await page.request.get(new URL("/api/v1/claims?status=under_review&limit=20", page.url()).toString())
    expect(reviewResponse.ok()).toBe(true)
    const reviewClaims = (await reviewResponse.json()).items as { code: string }[]
    const reviewClaim = reviewClaims[0]
    if (!reviewClaim) throw new Error("the demo needs a claim under officer review")
    await page.getByLabel("Status klaim").selectOption("under_review")
    await page.getByRole("button", { name: "Terapkan" }).click()
    await expect(page).toHaveURL(/status=under_review/)
    await expect(page.getByRole("link", { name: new RegExp(reviewClaim.code) })).toBeVisible()
    await captureB10(page, "antrean", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB10(page, "antrean", "360x800")

    await page.setViewportSize({ width: 1366, height: 900 })
    await page.goto(`/petugas/klaim/${reviewClaim.code}`)
    await expect(page.getByRole("heading", { name: reviewClaim.code })).toBeVisible()
    await expect(page.getByRole("heading", { name: "Daftar Periksa Verifikasi" })).toBeVisible()
    await expect(page.getByRole("heading", { name: "Tindakan Petugas" })).toBeVisible()
    await expect(page.getByRole("region", { name: "Catatan Internal Petugas" })).toBeVisible()
    await expect(page.getByRole("heading", { name: "Pesan dengan Petugas" })).toBeVisible()
    await captureB10(page, "detail", "1366x900")
    await page.setViewportSize({ width: 360, height: 800 })
    await captureB10(page, "detail", "360x800")

    expect(external).toEqual([])
    expect(errors).toEqual([])
  })
})

async function captureB11(page: Page, route: string, width: string) {
  await expect(page.getByRole("main")).toBeVisible()
  await expectAccessible(page)
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(await page.evaluate(() => window.innerWidth))
  await mkdir("test-results/screenshots", { recursive: true })
  await page.screenshot({ path: `test-results/screenshots/phase2-B11-${route}-${width}.png`, fullPage: true, animations: "disabled" })
}

test.describe("B11 officer proxy reports and records", () => {
  test("proxy report forms preserve identity, custody and staff-only evidence at both widths", async ({ page }) => {
    test.setTimeout(120_000)
    page.setDefaultTimeout(10_000)
    const external: string[] = []
    const errors: string[] = []
    page.on("request", (request) => {
      const url = new URL(request.url())
      if (/^https?:$/.test(url.protocol) && !/^(localhost|127\.0\.0\.1|\[::1\])$/.test(url.hostname)) external.push(request.url())
    })
    page.on("pageerror", (error) => errors.push(error.message))
    await page.goto("/masuk")
    await page.getByRole("button", { name: /Petugas \(Pos Perpus\)/ }).click()
    await expect(page).toHaveURL(/\/petugas$/)

    for (const viewport of [
      { width: 1366, height: 900, label: "1366x900" },
      { width: 360, height: 800, label: "360x800" },
    ]) {
      await page.setViewportSize(viewport)
      await page.goto("/petugas/lapor-hilang")
      await expect(page.getByRole("heading", { name: "Lapor kehilangan atas nama" })).toBeVisible()
      await expect(page.getByLabel("Nama pengunjung")).toBeVisible()
      await expect(page.getByRole("checkbox").first()).toBeVisible()
      await captureB11(page, "lapor-hilang", viewport.label)

      await page.goto("/petugas/lapor-temuan")
      await expect(page.getByRole("heading", { name: "Lapor temuan atas nama" })).toBeVisible()
      await expect(page.getByRole("button", { name: "Lanjut" })).toBeEnabled()
      await captureB11(page, "lapor-temuan", viewport.label)
    }
    expect(external).toEqual([])
    expect(errors).toEqual([])
  })
})
