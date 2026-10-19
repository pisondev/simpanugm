import { expect, test, type Browser, type Page } from "@playwright/test"

import { bootstrapAdmin, expectAccessible, setPassword, uniqueEmail } from "./helpers"

// Journey J7: Admin activates a fusion version and rolls back, monitoring reflects it (US-49, testing-strategy J7)
test.describe("Journey J7: Model Lifecycle and Fusion Rollback", () => {
  const adminEmail = uniqueEmail("admin-j7", "admin.example.id")
  const password = "kata sandi panjang 2026"

  async function fresh(browser: Browser): Promise<Page> {
    const context = await browser.newContext()
    return context.newPage()
  }

  test("admin activates candidate fusion version and rolls back with monitoring reflecting it", async ({ browser }) => {
    const admin = await fresh(browser)
    await setPassword(admin, bootstrapAdmin(adminEmail, "Admin J7"), password, "Aktifkan akun")
    await expect(admin).toHaveURL(/\/admin$/)
    await expectAccessible(admin)

    await admin.goto("/admin/ai/monitoring")
    await expect(admin.getByRole("heading", { name: "Monitoring AI & Mesin Pencocokan" })).toBeVisible()
    await expectAccessible(admin)

    // Verify initial active fusion version is visible
    const initialVersionCell = admin.getByRole("cell", { name: "fusion-linear-2026.11.0" })
    await expect(initialVersionCell).toBeVisible()

    // Open activation and rollback dialog
    await admin.getByRole("button", { name: "Kelola & Rollback Model" }).click()
    const dialog = admin.getByRole("dialog")
    await expect(dialog).toBeVisible()
    await expectAccessible(admin)

    // Select candidate fusion version
    const select = dialog.getByLabel("Pilih model versi")
    const candidateValue = await dialog.locator("option", { hasText: "fusion-learned-2026.11.0" }).getAttribute("value")
    if (candidateValue) {
      await select.selectOption(candidateValue)
    }
    await dialog.getByRole("button", { name: "Aktifkan Versi Ini" }).click()

    // Toast notification and table update
    await expect(admin.getByText(/Versi model fusion-learned-2026\.11\.0 kini aktif/)).toBeVisible()
    await expect(admin.getByRole("cell", { name: "fusion-learned-2026.11.0" })).toBeVisible()
    await expectAccessible(admin)

    // Rollback to original fusion version
    await admin.getByRole("button", { name: "Kelola & Rollback Model" }).click()
    const rollbackDialog = admin.getByRole("dialog")
    await expect(rollbackDialog).toBeVisible()
    await expectAccessible(admin)

    const rollbackSelect = rollbackDialog.getByLabel("Pilih model versi")
    const rollbackValue = await rollbackDialog.locator("option", { hasText: "fusion-linear-2026.11.0" }).getAttribute("value")
    if (rollbackValue) {
      await rollbackSelect.selectOption(rollbackValue)
    }
    await rollbackDialog.getByRole("button", { name: "Aktifkan Versi Ini" }).click()

    // Toast notification and table update reflecting rollback
    await expect(admin.getByText(/Versi model fusion-linear-2026\.11\.0 kini aktif/)).toBeVisible()
    await expect(admin.getByRole("cell", { name: "fusion-linear-2026.11.0" })).toBeVisible()
    await expectAccessible(admin)
  })
})
