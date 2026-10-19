import { screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { CampusHotspotMap } from "@/components/admin/campus-hotspot-map"
import { ClusterRunTrigger } from "@/components/admin/cluster-run-trigger"
import { ExportCsvForm } from "@/components/admin/export-csv-form"
import { ModelVersionActivator } from "@/components/admin/model-version-activator"
import type { HotspotLocation, ModelVersionItem } from "@/lib/api/types"

import { renderUi } from "./render"

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    refresh: vi.fn(),
  }),
}))

const mockLocations: HotspotLocation[] = [
  {
    location_id: "l1-0000-0000-0000-000000000001",
    location_name: "Perpustakaan Pusat UGM",
    zone_id: "z1-0000-0000-0000-000000000001",
    zone_name: "Bulaksumur",
    lat: -7.7713,
    lng: 110.3775,
    lost_count: 5,
    found_count: 7,
    total_count: 12,
    display_lost: "5",
    display_found: "7",
    display_total: "12",
    return_rate: 0.85,
    median_return_hours: 4.5,
  },
  {
    location_id: "l2-0000-0000-0000-000000000002",
    location_name: "Kantin Vokasi Sekip",
    zone_id: "z2-0000-0000-0000-000000000002",
    zone_name: "Sekip",
    lat: -7.773,
    lng: 110.372,
    lost_count: 2,
    found_count: 0,
    total_count: 2,
    display_lost: "< 3",
    display_found: "0",
    display_total: "< 3",
    return_rate: 0,
    median_return_hours: null,
  },
]

const mockModelVersions: ModelVersionItem[] = [
  {
    id: "mv1-0000-0000-0000-000000000001",
    component: "fusion",
    name: "fusion-linear-2026.11.0",
    status: "active",
    created_at: "2026-10-01T00:00:00Z",
  },
  {
    id: "mv2-0000-0000-0000-000000000002",
    component: "fusion",
    name: "fusion-linear-2026.10.0",
    status: "retired",
    created_at: "2026-09-01T00:00:00Z",
  },
]

describe("Admin AI & Analytics UI Components", () => {
  it("renders CampusHotspotMap with nodes and BR-PRV-05 count labels", () => {
    renderUi(<CampusHotspotMap locations={mockLocations} selectedType="all" />)

    expect(screen.getByText("Peta Kampus UGM (Hotspot Sebaran)")).toBeInTheDocument()
    expect(screen.getByText("<3")).toBeInTheDocument()
    expect(screen.getByText("12")).toBeInTheDocument()
  })

  it("renders ClusterRunTrigger with action button", () => {
    renderUi(<ClusterRunTrigger />)

    expect(screen.getByRole("button", { name: /Jalankan Pengelompokan Baru/i })).toBeInTheDocument()
  })

  it("renders ModelVersionActivator dialog trigger button", () => {
    renderUi(<ModelVersionActivator allVersions={mockModelVersions} />)

    expect(screen.getByRole("button", { name: /Kelola & Rollback Model/i })).toBeInTheDocument()
  })

  it("renders ExportCsvForm with month input and download button", () => {
    renderUi(<ExportCsvForm />)

    expect(screen.getByLabelText("Pilih bulan untuk ekspor CSV")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Unduh CSV/i })).toBeInTheDocument()
  })
})
