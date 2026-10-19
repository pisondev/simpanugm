import { expect, test, type Browser, type Page } from "@playwright/test"

import { bootstrapAdmin, expectAccessible, latestLink, photo, registerMember, setPassword, signIn, uniqueEmail } from "./helpers"

test.describe.configure({ mode: "serial" })

const password = "kata sandi panjang 2026"
const adminEmail = uniqueEmail("claim-admin", "admin.example.id")
const officerEmail = uniqueEmail("claim-petugas", "petugas.example.id")
const memberEmail = uniqueEmail("claim-anggota")
let foundCode = ""
let lostCode = ""
let claimCode = ""
let handoverCode = ""
const itemName = `Kunci motor Honda ${Date.now() % 100000}`

async function fresh(browser: Browser): Promise<Page> {
  const context = await browser.newContext()
  return context.newPage()
}

async function pickCategory(page: Page, search: string, option: string) {
  await page.getByRole("combobox", { name: "Kategori" }).click()
  await page.getByPlaceholder("Cari kategori, misalnya dompet").fill(search)
  await page.getByRole("option", { name: option, exact: true }).click()
}

async function pickPlace(page: Page, label: string, search: string, option: RegExp) {
  await page.getByLabel(label, { exact: true }).click()
  await page.getByPlaceholder("Cari gedung atau tempat, misalnya perpus").fill(search)
  await page.getByRole("option", { name: option }).first().click()
}

test("claims setup: administrator invites an officer to a post", async ({ browser }) => {
  const admin = await fresh(browser)
  await setPassword(admin, bootstrapAdmin(adminEmail, "Admin Klaim"), password, "Aktifkan akun")
  await expect(admin).toHaveURL(/\/admin$/)

  await admin.goto("/admin/pengguna")
  await admin.getByRole("button", { name: "Undang pengguna" }).click()
  const dialog = admin.getByRole("dialog")
  await dialog.getByLabel("Email").fill(officerEmail)
  await dialog.getByLabel("Nama lengkap").fill("Petugas Klaim")
  await dialog.getByText("Pos Lost and Found Perpustakaan Pusat").click()
  await dialog.getByRole("button", { name: "Kirim undangan" }).click()
  await expect(admin.getByText(`Undangan dikirim ke ${officerEmail}.`)).toBeVisible()

  const officer = await fresh(browser)
  await setPassword(officer, await latestLink(officerEmail, "/undangan"), password, "Aktifkan akun")
  await expect(officer).toHaveURL(/\/petugas$/)
})

test("officer records found item in custody and member files lost report", async ({ browser }) => {
  const officer = await fresh(browser)
  await signIn(officer, officerEmail, password)
  await expect(officer).toHaveURL(/\/petugas$/)
  await officer.goto("/petugas/lapor-temuan")
  await officer.locator('input[type="file"][multiple]').setInputFiles(await photo(officer))
  await expect(officer.getByRole("button", { name: "Hapus foto 1" })).toBeVisible()
  await expect(officer.getByText(/Mengunggah|Menyiapkan/)).toHaveCount(0, { timeout: 20_000 })
  await officer.getByRole("button", { name: "Lanjut" }).click()
  await pickCategory(officer, "kunci", "Kunci Motor")
  await officer.getByLabel("Nama barang").fill(itemName)
  await officer.getByLabel("Deskripsi").fill("Kunci motor Honda dengan gantungan hitam ditemukan di meja baca.")
  await officer.getByRole("button", { name: "Lanjut" }).click()
  await pickPlace(officer, "Tempat ditemukan", "perpus", /Perpustakaan Pusat UGM/)
  await officer.getByRole("button", { name: "Lanjut" }).click()
  await officer.getByText("Pos Lost and Found Perpustakaan Pusat").click()
  await officer.getByLabel("Rak atau slot penyimpanan").fill("Rak K1")
  await officer.getByRole("button", { name: "Lanjut" }).click()
  await officer.getByRole("button", { name: "Kirim laporan" }).click()
  await expect(officer).toHaveURL(/\/petugas\/laporan\/temuan\/FND-\d{4}-\d{4}$/)
  foundCode = officer.url().split("/").pop() ?? ""

  const member = await fresh(browser)
  await registerMember(member, "Budi Santoso", memberEmail, password)
  await signIn(member, memberEmail, password)
  await expect(member).toHaveURL(/\/beranda$/)
  await member.goto("/lapor/hilang")
  await member.getByRole("button", { name: "Lewati, saya tidak punya foto" }).click()
  await pickCategory(member, "kunci", "Kunci Motor")
  await member.getByLabel("Nama barang").fill(itemName)
  await member.getByLabel("Deskripsi").fill("Kunci motor Honda Vario warna hitam dengan gantungan pita.")
  await member.getByRole("button", { name: "Lanjut" }).click()
  await pickPlace(member, "Tempat terakhir terlihat", "perpus", /Perpustakaan Pusat UGM/)
  await member.getByRole("button", { name: "Lanjut" }).click()
  await member.getByRole("button", { name: "Lanjut" }).click()
  await member.getByRole("button", { name: "Kirim laporan" }).click()
  await expect(member).toHaveURL(/\/laporan\/hilang\/LST-\d{4}-\d{4}$/)
  lostCode = member.url().split("/").pop() ?? ""
})

test("member submits claim for the found item", async ({ browser }) => {
  const page = await fresh(browser)
  await signIn(page, memberEmail, password)
  await expect(page).toHaveURL(/\/beranda$/)
  await page.goto("/barang-temuan")
  await expectAccessible(page)

  const card = page.locator("li").filter({ hasText: itemName })
  await expect(card).toBeVisible()
  await card.getByRole("link", { name: "Ini milik saya" }).click()

  await expect(page).toHaveURL(/\/klaim\/baru/)
  await expectAccessible(page)

  await page
    .getByLabel(new RegExp(`Mengapa barang ${itemName} ini milikmu\\?`))
    .fill("Ini adalah kunci motor saya yang tertinggal di perpustakaan lantai dua kemarin.")
  await page.getByLabel("Sebutkan merek motor dan bentuk atau isi gantungan kunci.").fill("Motor Honda Vario 125, gantungan pita hitam.")
  await page.getByLabel("Rencana waktu pengambilan").fill("Besok siang jam 13:00")
  await page.getByRole("button", { name: "Kirim Pengajuan Klaim" }).click()

  await expect(page).toHaveURL(/\/klaim\/CLM-\d{4}-\d{4}$/)
  claimCode = page.url().split("/").pop() ?? ""
  await expect(page.getByText(foundCode)).toBeVisible()
  await expect(page.getByText("Diajukan", { exact: true })).toBeVisible()
  await expectAccessible(page)
})

test("officer reviews checklist and approves claim", async ({ browser }) => {
  const page = await fresh(browser)
  await signIn(page, officerEmail, password)
  await expect(page).toHaveURL(/\/petugas$/)
  await page.goto("/petugas/klaim")
  await expectAccessible(page)

  const card = page.locator("li").filter({ hasText: claimCode })
  await expect(card).toBeVisible()
  await card.click()

  await expect(page).toHaveURL(`/petugas/klaim/${claimCode}`)
  await expect(page.getByText(lostCode)).toBeVisible()
  await expectAccessible(page)

  const pickUpBtn = page.getByRole("button", { name: "Ambil Alih Peninjauan" })
  if (await pickUpBtn.isVisible()) {
    await pickUpBtn.click()
    await expect(page.getByText("Klaim diambil alih untuk ditinjau.")).toBeVisible()
  }

  const checklistSection = page.locator("section", { hasText: "Daftar Periksa Verifikasi" })
  const selects = checklistSection.locator("select")
  const count = await selects.count()
  for (let i = 0; i < count; i++) {
    await selects.nth(i).selectOption("pass")
  }
  if (count > 0) {
    await checklistSection.getByRole("button", { name: "Simpan Hasil Periksa" }).click()
    await expect(page.getByText("Hasil periksa disimpan.")).toBeVisible()
  }

  await page.getByRole("button", { name: "Setujui Klaim" }).click()
  await expect(page.getByText("Klaim berhasil disetujui. Kode serah terima diterbitkan.")).toBeVisible()
})

test("claimant sees code and officer completes handover", async ({ browser }) => {
  const member = await fresh(browser)
  await signIn(member, memberEmail, password)
  await expect(member).toHaveURL(/\/beranda$/)
  await member.goto(`/klaim/${claimCode}`)
  await expect(member.getByText("Disetujui", { exact: true })).toBeVisible()

  const codeLocator = member.locator("p.tracking-\\[0\\.3em\\]")
  await expect(codeLocator).toBeVisible()
  handoverCode = (await codeLocator.textContent())?.trim() ?? ""
  expect(handoverCode).toHaveLength(6)
  await expectAccessible(member)

  const officer = await fresh(browser)
  await signIn(officer, officerEmail, password)
  await expect(officer).toHaveURL(/\/petugas$/)
  await officer.goto(`/petugas/klaim/${claimCode}`)

  const handoverSection = officer.locator("section", { hasText: "Formulir Serah Terima Barang" })
  await handoverSection.locator("#h-code").fill(handoverCode)
  await handoverSection.locator("#h-last4").fill("5678")
  await handoverSection.getByRole("checkbox").click()
  await handoverSection.getByRole("button", { name: "Konfirmasi Serah Terima" }).click()

  await expect(officer.getByText("Barang berhasil diserahterimakan.")).toBeVisible()
  await expect(officer.getByText("Diserahkan", { exact: true }).first()).toBeVisible()
})
