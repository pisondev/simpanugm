import { screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { ModerateReportDialog } from "@/components/admin/moderate-report-dialog"
import { SettingsForm } from "@/components/admin/settings-form"
import { UserModerationMenu } from "@/components/admin/user-moderation-dialog"
import { DeleteAccountDialog } from "@/components/profile/delete-account-dialog"
import type { AdminUser, SystemSettingItem } from "@/lib/api/types"

import { renderUi } from "./render"

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    refresh: vi.fn(),
  }),
}))

const mockSettings: SystemSettingItem[] = [
  { key: "lost.stale_days", value: 60 },
  { key: "lost.expiry_grace_days", value: 14 },
  { key: "lost.reopen_days", value: 30 },
  { key: "found.with_finder_max_hours", value: 48 },
  { key: "custody.retention_days.normal", value: 30 },
  { key: "custody.retention_days.valuable", value: 60 },
  { key: "custody.retention_days.identity", value: 30 },
  { key: "claim.max_open", value: 3 },
  { key: "claim.reject_flag_count", value: 3 },
  { key: "claim.reject_flag_window_days", value: 30 },
  { key: "claim.info_timeout_days", value: 7 },
  { key: "claim.approved_timeout_days", value: 7 },
  { key: "notify.match_throttle_hours", value: 6 },
  { key: "notify.email_quiet_hours", value: { from: "22:00", to: "06:00" } },
  { key: "auth.allowed_email_domains", value: ["mail.ugm.ac.id", "ugm.ac.id"] },
  { key: "analytics.min_cell_count", value: 3 },
]

const mockUser: AdminUser = {
  id: "u1a2b3c4-0000-0000-0000-000000000001",
  email: "test.member@mail.ugm.ac.id",
  full_name: "Budi Uji Coba",
  role: "member",
  status: "active",
  email_verified: true,
  post_ids: [],
  created_at: "2026-01-01T00:00:00Z",
  last_login_at: "2026-01-02T00:00:00Z",
  claim_restricted_at: null,
}

describe("Admin Settings and Moderation UI", () => {
  it("renders SettingsForm with business rule groupings and initial values", () => {
    renderUi(<SettingsForm initialItems={mockSettings} />)

    expect(screen.getByText("Batas Waktu Laporan Kehilangan")).toBeInTheDocument()
    expect(screen.getByText("Barang Temuan & Retensi Penitipan")).toBeInTheDocument()
    expect(screen.getByText("Batas & Ambang Klaim")).toBeInTheDocument()
    expect(screen.getByText("Notifikasi & Jam Hening")).toBeInTheDocument()
    expect(screen.getByText("Autentikasi & Analitik")).toBeInTheDocument()

    const staleInput = screen.getByLabelText("Masa Aktif Laporan (Hari)") as HTMLInputElement
    expect(staleInput.value).toBe("60")

    const claimMaxInput = screen.getByLabelText("Maksimal Klaim Terbuka") as HTMLInputElement
    expect(claimMaxInput.value).toBe("3")
  })

  it("renders UserModerationMenu with accessible moderation trigger", () => {
    renderUi(<UserModerationMenu user={mockUser} />)

    const trigger = screen.getByRole("button", { name: /moderasi budi uji coba/i })
    expect(trigger).toBeInTheDocument()
  })

  it("renders ModerateReportDialog with hide trigger", () => {
    renderUi(<ModerateReportDialog type="lost" id="lost-001" />)

    const trigger = screen.getByRole("button", { name: /sembunyikan/i })
    expect(trigger).toBeInTheDocument()
  })

  it("renders DeleteAccountDialog with destructive button", () => {
    renderUi(<DeleteAccountDialog />)

    const trigger = screen.getByRole("button", { name: /hapus akunku/i })
    expect(trigger).toBeInTheDocument()
  })
})
