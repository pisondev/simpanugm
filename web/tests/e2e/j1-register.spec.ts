import { expect, test } from "@playwright/test"

import { expectAccessible, registerMember, signIn, uniqueEmail } from "./helpers"

test("a visitor can browse the public found listing", async ({ page }) => {
  for (const viewport of [
    { width: 1280, height: 720 },
    { width: 360, height: 800 },
  ]) {
    await page.setViewportSize(viewport)
    await page.goto("/barang-temuan")
    await expect(page).toHaveURL(/\/barang-temuan$/)
    await expect(page.getByRole("heading", { name: "Barang Temuan" })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(viewport.width)
    await expectAccessible(page)
  }
})

// J1: register, verify through Mailpit, sign in (US-01, US-02)
test("a student registers, verifies the email and signs in", async ({ page }) => {
  const email = uniqueEmail("j1")
  const password = "tumbler biru dongker 2026"

  await page.goto("/daftar")
  await expectAccessible(page)
  await registerMember(page, "Rani Prasetya", email, password)
  await expectAccessible(page)

  await signIn(page, email, password)
  await expect(page).toHaveURL(/\/beranda$/)
  await expect(page.getByRole("heading", { name: /Halo, Rani/ })).toBeVisible()
  await expectAccessible(page)
})

test("a non campus email is refused before an account exists", async ({ page }) => {
  await page.goto("/daftar")
  await page.getByLabel("Nama lengkap").fill("Bukan Mahasiswa")
  await page.getByLabel("Email UGM").fill(uniqueEmail("luar", "gmail.com"))
  await page.getByLabel("Kata sandi", { exact: true }).fill("kata sandi yang panjang")
  await page.getByRole("checkbox").check()
  await page.getByRole("button", { name: "Buat akun" }).click()
  await expect(page.locator("#email-error")).toHaveText("Gunakan email UGM (@mail.ugm.ac.id atau @ugm.ac.id)")
})

test("a wrong password gets one message for every cause", async ({ page }) => {
  await signIn(page, uniqueEmail("tidak-ada"), "kata sandi yang salah")
  await expect(page.getByText("Email atau kata sandi salah")).toBeVisible()
})

test("a protected page sends a visitor to sign in and back", async ({ page }) => {
  await page.goto("/laporan")
  await expect(page).toHaveURL(/\/masuk\?next=%2Flaporan$/)
})
