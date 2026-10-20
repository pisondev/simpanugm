import { readFileSync } from "node:fs"
import { resolve } from "node:path"

import { expect, test } from "@playwright/test"

import { expectAccessible, signIn } from "./helpers"

test.describe.configure({ mode: "serial" })

test("D01: unseeded flash drive holdout uses the real lost report flow", async ({ page }) => {
  test.setTimeout(240_000)
  const externalRequests: string[] = []
  page.on("request", (request) => {
    const url = new URL(request.url())
    if (url.protocol.startsWith("http") && !["localhost", "127.0.0.1"].includes(url.hostname)) {
      externalRequests.push(request.url())
    }
  })

  await signIn(page, "anggota.penemu@demo.simpanugm.my.id", "kata sandi panjang 2026")
  await expect(page).toHaveURL(/\/beranda$/)
  await page.goto("/lapor/hilang")
  const assistResult = page.waitForResponse((response) => response.url().includes("/api/v1/assist/analyze-image"), { timeout: 45_000 })
  await page.locator('input[type="file"][multiple]').setInputFiles({
    name: "holdout-flashdisk.jpg",
    mimeType: "image/jpeg",
    buffer: readFileSync(resolve("../data/demo/images/bg-flashdisk-011.jpg")),
  })
  await expect(page.getByRole("button", { name: "Hapus foto 1" })).toBeVisible({ timeout: 20_000 })
  await expect(page.getByText(/Mengunggah|Menyiapkan/)).toHaveCount(0, { timeout: 30_000 })
  await page.getByRole("button", { name: "Lanjut" }).click()
  await expect(page.getByText(/Saran otomatis sedang tidak tersedia|Saran dari fotomu sudah siap/).first()).toBeVisible({ timeout: 45_000 })
  const response = await assistResult
  const suggestions = (await response.json().catch(() => ({}))) as {
    categories?: Array<{ category?: { name?: string; slug?: string }; score?: number }>
  }
  console.log(`D01 live holdout response: HTTP ${response.status()}, ${JSON.stringify(suggestions.categories?.slice(0, 3) ?? [])}`)

  await page.getByRole("combobox", { name: "Kategori" }).click()
  await page.getByPlaceholder("Cari kategori, misalnya dompet").fill("flashdisk")
  await page.getByRole("option", { name: "Flashdisk", exact: true }).click()
  await page.getByRole("button", { name: "hitam, terpilih", exact: true }).click()
  await page.getByRole("button", { name: "perak, terpilih", exact: true }).click()
  await page.getByRole("button", { name: "biru", exact: true }).click()
  await page.getByRole("button", { name: "putih", exact: true }).click()
  await page.getByLabel("Nama barang").fill("Flashdisk contoh holdout")
  await page.getByLabel("Deskripsi").fill("Flashdisk bertema pesawat dengan beberapa perangkat USB berbentuk pesawat pada foto.")
  await page.getByRole("button", { name: "Lanjut" }).click()
  await page.getByLabel("Tempat terakhir terlihat", { exact: true }).click()
  await page.getByPlaceholder("Cari gedung atau tempat, misalnya perpus").fill("perpus")
  await page
    .getByRole("option", { name: /Perpustakaan Pusat UGM/ })
    .first()
    .click()
  await page.getByRole("button", { name: "Lanjut" }).click()
  await page.getByRole("button", { name: "Lanjut" }).click()
  await page.getByRole("button", { name: "Kirim laporan" }).click()

  await expect(page).toHaveURL(/\/laporan\/hilang\/LST-\d{4}-\d{4}$/)
  await expect(page.getByRole("heading", { name: "Flashdisk contoh holdout" })).toBeVisible()
  await expect(page.getByText("Mencari kandidat...")).toBeVisible()
  await page.waitForTimeout(125_000)
  await page.reload()
  await expect(page.getByText("Mencari kandidat...")).toHaveCount(0, { timeout: 20_000 })
  console.log(`D01 live match count: ${await page.locator("#kandidat ol li").count()}`)
  await page.screenshot({ path: "test-results/screenshots/d01-holdout-desktop.png", fullPage: true })
  await expectAccessible(page)
  expect(externalRequests).toEqual([])
})
