import { expect, test } from "@playwright/test"

import { allFoundReports, expectAccessible, signIn } from "./helpers"

test("demo reset exposes handover, intake and information-response actions", async ({ browser }) => {
  test.setTimeout(120_000)
  const ownerContext = await browser.newContext()
  const officerContext = await browser.newContext()
  try {
    const owner = await ownerContext.newPage()
    await signIn(owner, "anggota.pemilik@demo.simpanugm.my.id", "kata sandi panjang 2026")
    await expect(owner).toHaveURL(/\/beranda$/)
    const approvedResponse = await owner.request.get("/api/v1/claims?status=approved&limit=100")
    expect(approvedResponse.ok()).toBe(true)
    const approved = ((await approvedResponse.json()).items as { code: string }[])[0]
    if (!approved) throw new Error("demo reset must provide an approved owner claim")
    await owner.goto(`/klaim/${approved.code}`)
    await expect(owner.getByRole("heading", { name: "Kode Serah Terima" })).toBeVisible()
    await expect(owner.getByLabel(/^Kode serah terima (\d ){5}\d$/)).toBeVisible()
    await expectAccessible(owner)

    const infoResponse = await owner.request.get("/api/v1/claims?status=info_requested&limit=100")
    expect(infoResponse.ok()).toBe(true)
    const requested = ((await infoResponse.json()).items as { code: string }[])[0]
    if (!requested) throw new Error("demo reset must provide an owner information request")
    await owner.goto(`/klaim/${requested.code}`)
    await expect(owner.getByLabel("Tanggapi Permintaan Informasi")).toBeVisible()
    await expectAccessible(owner)

    const officer = await officerContext.newPage()
    await signIn(officer, "petugas.pos1@demo.simpanugm.my.id", "kata sandi panjang 2026")
    await expect(officer).toHaveURL(/\/petugas$/)
    await officer.goto(`/petugas/klaim/${approved.code}`)
    await expect(officer.getByRole("heading", { name: "Formulir Serah Terima Barang", exact: true })).toBeVisible()
    await expectAccessible(officer)
    const intake = (await allFoundReports(officer)).find((item) => item.custody_status === "with_finder")
    if (!intake) throw new Error("demo reset must provide an item still with its finder")
    await officer.goto(`/petugas/terima?code=${encodeURIComponent(intake.code)}`)
    await expect(officer.getByLabel("Rak atau slot")).toBeVisible()
    await expect(officer.getByRole("button", { name: "Terima barang" })).toBeVisible()
    await expectAccessible(officer)
  } finally {
    await ownerContext.close()
    await officerContext.close()
  }
})
