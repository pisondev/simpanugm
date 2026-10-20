import { mkdir } from "node:fs/promises"

import { expect, test, type Browser, type Page } from "@playwright/test"

import { bootstrapAdmin, expectAccessible, attachPhoto, latestLink, registerMember, setPassword, signIn, uniqueEmail } from "./helpers"

test.describe.configure({ mode: "serial" })

const password = "kata sandi panjang 2026"
const adminEmail = uniqueEmail("claim-admin", "admin.example.id")
const officerEmail = uniqueEmail("claim-petugas", "petugas.example.id")
const memberEmail = uniqueEmail("claim-anggota")
let foundCode = ""
let lostCode = ""
let claimCode = ""
let handoverCode = ""
let j6FoundCode = ""
const itemName = `Kunci motor Honda ${Date.now() % 100000}`
const j6ItemName = `Payung lipat abu ${Date.now() % 100000}`

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
  await expect(officer.getByRole("heading", { name: "Lapor temuan atas nama" })).toBeVisible()
  await attachPhoto(officer)
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
  await expect(page.getByText(foundCode, { exact: true })).toBeVisible()
  await expect(page.getByText("Diajukan", { exact: true })).toBeVisible()
  await expectAccessible(page)
})

test("J4 officer requests info, owner replies, officer approves claim", async ({ browser }) => {
  const officer = await fresh(browser)
  await signIn(officer, officerEmail, password)
  await expect(officer).toHaveURL(/\/petugas$/)
  await officer.goto("/petugas/klaim")
  await expectAccessible(officer)

  const card = officer.locator("li").filter({ hasText: claimCode })
  await expect(card).toBeVisible()
  await card.click()

  await expect(officer).toHaveURL(`/petugas/klaim/${claimCode}`)
  await expect(officer.getByText(lostCode, { exact: true })).toBeVisible()
  await expectAccessible(officer)

  const pickUpBtn = officer.getByRole("button", { name: "Ambil Alih Peninjauan" })
  if (await pickUpBtn.isVisible()) {
    await pickUpBtn.click()
    await expect(officer.getByText("Klaim diambil alih untuk ditinjau.")).toBeVisible()
  }

  // Officer requests info (US-27)
  await officer.getByRole("button", { name: "Minta Informasi" }).click()
  const reqDialog = officer.getByRole("alertdialog")
  await reqDialog.getByLabel("Pesan untuk pemohon").fill("Mohon sebutkan gantungan kunci secara rinci.")
  await reqDialog.getByRole("button", { name: "Kirim Permintaan" }).click()
  await expect(officer.getByText("Permintaan informasi dikirim ke pemohon.")).toBeVisible()

  // Claimant responds to info request (US-27)
  const member = await fresh(browser)
  await signIn(member, memberEmail, password)
  await expect(member).toHaveURL(/\/beranda$/)
  await member.goto(`/klaim/${claimCode}`)
  await expect(member.getByText("Perlu Informasi", { exact: true })).toBeVisible()
  await expectAccessible(member)

  await member.locator("#claim-respond").fill("Gantungan kunci memiliki tali pita hitam dan gantungan akrilik.")
  await member.getByRole("button", { name: "Kirim Tanggapan" }).click()
  await expect(member.getByText("Tanggapan berhasil dikirim.")).toBeVisible()

  // Officer reviews checklist and approves (US-26)
  await officer.goto(`/petugas/klaim/${claimCode}`)
  await expect(officer.getByText("Sedang Ditinjau", { exact: true })).toBeVisible()

  const checklistSection = officer.locator("section", { hasText: "Daftar Periksa Verifikasi" })
  await expect(checklistSection).toBeVisible()
  const selects = checklistSection.locator("select")
  await expect(selects.first()).toBeVisible()
  const count = await selects.count()
  for (let i = 0; i < count; i++) {
    await selects.nth(i).selectOption("pass")
  }
  if (count > 0) {
    await checklistSection.getByRole("button", { name: "Simpan Hasil Periksa" }).click()
    await expect(officer.getByText("Hasil periksa disimpan.")).toBeVisible()
    await expect(officer.getByText("Semua butir wajib sudah diperiksa.")).toBeVisible()
  }

  await officer.getByRole("button", { name: "Setujui Klaim" }).click()
  await expect(officer.getByText("Klaim berhasil disetujui. Kode serah terima diterbitkan.")).toBeVisible()
  await expect(officer.getByText("Disetujui", { exact: true })).toBeVisible()
  await expect(officer.getByRole("heading", { name: "Formulir Serah Terima Barang" })).toBeVisible()
  await mkdir("test-results/screenshots", { recursive: true })
  for (const viewport of [
    { width: 1366, height: 900, label: "1366x900" },
    { width: 360, height: 800, label: "360x800" },
  ]) {
    await officer.setViewportSize({ width: viewport.width, height: viewport.height })
    await expectAccessible(officer)
    await officer.screenshot({
      path: `test-results/screenshots/phase2-B10-handover-${viewport.label}.png`,
      fullPage: true,
      animations: "disabled",
    })
  }
})

test("J5 claimant sees code, officer tries wrong code then correct one, owner confirms receipt", async ({ browser }) => {
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

  // Five incorrect codes lock handover until an officer verifies the claimant and regenerates it.
  const wrongCode = handoverCode === "000000" ? "000001" : "000000"
  await handoverSection.locator("#h-last4").fill("5678")
  await handoverSection.getByRole("checkbox").click()
  for (let attempt = 0; attempt < 5; attempt++) {
    await handoverSection.locator("#h-code").fill(wrongCode)
    await handoverSection.getByRole("button", { name: "Konfirmasi Serah Terima" }).click()
    if (attempt < 4) {
      await expect(handoverSection.getByText("Kode serah terima tidak cocok.")).toBeVisible()
    } else {
      await expect(handoverSection.getByText(/Terlalu banyak kode salah/)).toBeVisible()
    }
  }
  await expect(officer.getByText("Kode serah terima terkunci karena percobaan gagal berulang.")).toBeVisible()
  await expect(officer.getByRole("button", { name: "Buat Ulang Kode" })).toBeVisible()
  await expectAccessible(officer)
  await mkdir("test-results/screenshots", { recursive: true })
  await officer.screenshot({ path: "test-results/screenshots/phase2-B10-handover-locked-1366x900.png", fullPage: true, animations: "disabled" })

  await officer.getByRole("button", { name: "Buat Ulang Kode" }).click()
  await expect(officer.getByText("Kode serah terima baru telah diterbitkan.")).toBeVisible()
  await member.reload()
  const renewedCode = (await member.locator("p.tracking-\\[0\\.3em\\]").textContent())?.trim() ?? ""
  expect(renewedCode).toHaveLength(6)
  expect(renewedCode).not.toBe(handoverCode)

  // Officer enters the regenerated code after rechecking the claimant.
  await officer.reload()
  const renewedHandover = officer.locator("section", { hasText: "Formulir Serah Terima Barang" })
  await renewedHandover.locator("#h-code").fill(renewedCode)
  await renewedHandover.locator("#h-last4").fill("5678")
  await renewedHandover.getByRole("checkbox").check()
  await renewedHandover.getByRole("button", { name: "Konfirmasi Serah Terima" }).click()

  await expect(officer.getByText("Barang berhasil diserahterimakan.")).toBeVisible()
  await expect(officer.getByText("Diserahkan", { exact: true }).first()).toBeVisible()

  // Owner confirms receipt on the lost report page (US-11, US-31)
  await member.goto(`/laporan/hilang/${lostCode}`)
  await expect(member.getByText("Sudah kembali", { exact: true })).toBeVisible()
  await member.getByRole("button", { name: "Konfirmasi Penerimaan" }).click()
  const confirmDialog = member.getByRole("alertdialog")
  await confirmDialog.getByRole("button", { name: "Ya, Saya Sudah Menerima" }).click()
  await expect(member.getByText("Penerimaan barang berhasil dikonfirmasi.")).toBeVisible()
  await expect(member.getByText("Ditutup", { exact: true })).toBeVisible()
  await expectAccessible(member)
})

test("member without prior lost report submits claim directly from public listing (inline lost report, US-26)", async ({ browser }) => {
  const directItemName = `Kunci motor Beat ${Date.now() % 100000}`
  const directMemberEmail = uniqueEmail("claim-direct")

  // Officer records an item in custody at the library post
  const officer = await fresh(browser)
  await signIn(officer, officerEmail, password)
  await expect(officer).toHaveURL(/\/petugas$/)
  await officer.goto("/petugas/lapor-temuan")
  await expect(officer.getByRole("heading", { name: "Lapor temuan atas nama" })).toBeVisible()
  await attachPhoto(officer)
  await expect(officer.getByText(/Mengunggah|Menyiapkan/)).toHaveCount(0, { timeout: 20_000 })
  await officer.getByRole("button", { name: "Lanjut" }).click()
  await pickCategory(officer, "kunci", "Kunci Motor")
  await officer.getByLabel("Nama barang").fill(directItemName)
  await officer.getByLabel("Deskripsi").fill("Kunci motor Honda Beat tertinggal di area loker.")
  await officer.getByRole("button", { name: "Lanjut" }).click()
  await pickPlace(officer, "Tempat ditemukan", "perpus", /Perpustakaan Pusat UGM/)
  await officer.getByRole("button", { name: "Lanjut" }).click()
  await officer.getByText("Pos Lost and Found Perpustakaan Pusat").click()
  await officer.getByLabel("Rak atau slot penyimpanan").fill("Rak K2")
  await officer.getByRole("button", { name: "Lanjut" }).click()
  await officer.getByRole("button", { name: "Kirim laporan" }).click()
  await expect(officer).toHaveURL(/\/petugas\/laporan\/temuan\/FND-\d{4}-\d{4}$/)
  const directFoundCode = officer.url().split("/").pop() ?? ""

  // Member without prior lost reports browses public listing and files direct claim
  const member = await fresh(browser)
  await registerMember(member, "Siti Rahma", directMemberEmail, password)
  await signIn(member, directMemberEmail, password)
  await expect(member).toHaveURL(/\/beranda$/)

  await member.goto("/barang-temuan")
  await expectAccessible(member)

  const card = member.locator("li").filter({ hasText: directItemName })
  await expect(card).toBeVisible()
  await card.getByRole("link", { name: "Ini milik saya" }).click()

  await expect(member).toHaveURL(/\/klaim\/baru/)
  await expectAccessible(member)

  // Verify inline lost report notice and section are displayed
  await expect(member.getByText("Buat Laporan Kehilangan Sekaligus")).toBeVisible()
  await expect(member.getByText("Detail Laporan Kehilangan Baru")).toBeVisible()

  // Fill in inline lost report fields
  await member.getByLabel("Nama barang yang hilang").fill(`Kunci motor Beat ${directItemName}`)
  await member.getByLabel("Deskripsi barang").fill("Kunci motor Honda Beat warna hitam dengan gantungan karet merah hilang di dekat loker.")

  // Fill claim statement and answers
  await member
    .getByLabel(new RegExp(`Mengapa barang ${directItemName} ini milikmu\\?`))
    .fill("Kunci ini milik saya yang terjatuh saat membuka loker perpustakaan kemarin sore.")
  await member.getByLabel("Sebutkan merek motor dan bentuk atau isi gantungan kunci.").fill("Honda Beat gantungan karet merah")
  await member.getByRole("button", { name: "Kirim Pengajuan Klaim" }).click()

  await expect(member).toHaveURL(/\/klaim\/CLM-\d{4}-\d{4}$/)
  await expect(member.getByText(directFoundCode, { exact: true })).toBeVisible()
  await expect(member.getByText("Diajukan", { exact: true })).toBeVisible()
  await expectAccessible(member)
})

test("J6 retention passes on fake clock, officer disposes unclaimed item", async ({ browser }) => {
  const officer = await fresh(browser)
  await signIn(officer, officerEmail, password)
  await expect(officer).toHaveURL(/\/petugas$/)

  // Officer records an item in custody at the post
  await officer.goto("/petugas/lapor-temuan")
  await expect(officer.getByRole("heading", { name: "Lapor temuan atas nama" })).toBeVisible()
  await attachPhoto(officer)
  await expect(officer.getByText(/Mengunggah|Menyiapkan/)).toHaveCount(0, { timeout: 20_000 })
  await officer.getByRole("button", { name: "Lanjut" }).click()
  await pickCategory(officer, "payung", "Payung")
  await officer.getByLabel("Nama barang").fill(j6ItemName)
  await officer.getByLabel("Deskripsi").fill("Payung lipat abu-abu tertinggal di dekat rak buku perpustakaan.")
  await officer.getByRole("button", { name: "Lanjut" }).click()
  await pickPlace(officer, "Tempat ditemukan", "perpus", /Perpustakaan Pusat UGM/)
  await officer.getByRole("button", { name: "Lanjut" }).click()
  await officer.getByText("Pos Lost and Found Perpustakaan Pusat").click()
  await officer.getByLabel("Rak atau slot penyimpanan").fill("Rak P3")
  await officer.getByRole("button", { name: "Lanjut" }).click()
  await officer.getByRole("button", { name: "Kirim laporan" }).click()
  await expect(officer).toHaveURL(/\/petugas\/laporan\/temuan\/FND-\d{4}-\d{4}$/)
  j6FoundCode = officer.url().split("/").pop() ?? ""

  // Advance fake clock past retention due date (95 days) and trigger retention sweep
  const advanceRes = await officer.request.post("http://localhost:8000/api/test/clock/advance", {
    data: { days: 95 },
  })
  expect(advanceRes.ok()).toBeTruthy()

  const sweepRes = await officer.request.post("http://localhost:8000/api/test/custody/sweep")
  expect(sweepRes.ok()).toBeTruthy()

  // Fresh officer session after clock advance past previous session expiry
  const officer2 = await fresh(browser)
  await signIn(officer2, officerEmail, password)
  await expect(officer2).toHaveURL(/\/petugas$/)

  // Officer views inventory: item is now unclaimed
  await officer2.goto("/petugas/inventaris")
  await expectAccessible(officer2)
  const invRow = officer2.locator("tr").filter({ hasText: j6FoundCode })
  await expect(invRow).toBeVisible()
  await expect(invRow.getByText("Tidak diklaim")).toBeVisible()

  // Officer opens item detail and records disposition (US-33)
  await invRow.getByRole("link", { name: new RegExp(j6ItemName) }).click()
  await expect(officer2).toHaveURL(`/petugas/laporan/temuan/${j6FoundCode}`)
  await expect(officer2.getByText("Tidak diklaim", { exact: true })).toBeVisible()
  await expectAccessible(officer2)

  await officer2.getByRole("button", { name: "Tindak Lanjut" }).click()
  const dispDialog = officer2.getByRole("alertdialog")
  await dispDialog.locator("#disp-kind").selectOption("donated")
  await dispDialog.locator("#disp-recipient").fill("Panti Asuhan Mitra UGM")
  await dispDialog.locator("#disp-note").fill("Masa simpan telah habis, disalurkan ke panti asuhan mitra.")
  await dispDialog.getByRole("button", { name: "Simpan Tindak Lanjut" }).click()

  await expect(officer2.getByText("Tindak lanjut barang berhasil disimpan.")).toBeVisible()
  await expect(officer2.getByText("Sudah ditindaklanjuti", { exact: true })).toBeVisible()
  await expectAccessible(officer2)

  // Reset fake clock back to current system time
  await officer2.request.post("http://localhost:8000/api/test/clock/reset")
})
