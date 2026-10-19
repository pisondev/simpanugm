import { screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { ReviewBoardSearch, ReviewBoardView } from "@/components/officer/review-board"
import type { OfficerBoard } from "@/lib/api/types"

import { renderUi } from "./render"

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    refresh: vi.fn(),
  }),
}))

const mockBoard: OfficerBoard = {
  report_type: "found",
  found_report: {
    id: "f1a2b3c4-0000-0000-0000-000000000001",
    code: "TEM-2610-0001",
    item_name: "KTM Mahasiswa UGM",
    status: "reported",
    custody_status: "in_custody",
    custody_keep: false,
    proxy: false,
    version: 1,
    description: "Ditemukan di Perpustakaan Pusat lantai 2.",
    hidden_details: "Nama di KTM: Budi Santoso, NIM 22/123456/TK/54321",
    found_at: "2026-10-18T10:00:00Z",
    created_at: "2026-10-18T10:30:00Z",
    updated_at: "2026-10-18T10:30:00Z",
    category: {
      id: "cat-1",
      slug: "kartu-identitas",
      name: "Kartu Identitas",
      parent_name: "Dokumen",
      sensitivity: "identity",
    },
    found_location: {
      id: "loc-1",
      name: "Perpustakaan Pusat",
      zone_id: "z1",
      zone_name: "Bulaksumur",
    },
    holding_post: {
      id: "post-1",
      name: "Pos Induk Gelanggang",
    },
    photos: [],
    colours: ["Kuning"],
  },
  candidates: [
    {
      id: "m1a2b3c4-0000-0000-0000-000000000001",
      tier: "high",
      overall: 0.92,
      pinned: true,
      pinned_at: "2026-10-19T08:00:00Z",
      scored_at: "2026-10-19T08:00:00Z",
      status: "active",
      flags: ["name_hit"],
      lost_report: {
        id: "l1a2b3c4-0000-0000-0000-000000000001",
        code: "HIL-2610-0001",
        item_name: "KTM UGM Kuning",
        owner_name: "Budi Santoso",
        lost_from: "2026-10-18T09:00:00Z",
        lost_until: "2026-10-18T11:00:00Z",
        category: {
          id: "cat-1",
          slug: "kartu-identitas",
          name: "Kartu Identitas",
          parent_name: "Dokumen",
          sensitivity: "identity",
        },
      },
      found_report: {
        id: "f1a2b3c4-0000-0000-0000-000000000001",
        code: "TEM-2610-0001",
        item_name: "KTM Mahasiswa UGM",
        category: {
          id: "cat-1",
          slug: "kartu-identitas",
          name: "Kartu Identitas",
          parent_name: "Dokumen",
          sensitivity: "identity",
        },
        found_at: "2026-10-18T10:00:00Z",
        found_location: {
          id: "loc-1",
          name: "Perpustakaan Pusat",
          zone_id: "z1",
          zone_name: "Bulaksumur",
        },
        description: "Ditemukan di Perpustakaan Pusat lantai 2.",
        hidden_details: "Nama di KTM: Budi Santoso",
        colours: ["Kuning"],
        custody_status: "in_custody",
        photos: [],
      },
      scores: {
        visual: 0.88,
        text: 0.95,
        location: 0.9,
        time: 0.95,
        category_factor: 1,
      },
      explanation: {
        visual_mode: "cross_modal",
        visual: { level: "high", reason: "Deskripsi sesuai dengan foto KTM.", missing: false },
        text: { level: "high", reason: "Nama dan teks pada kartu identitas cocok.", missing: false },
        location: { level: "high", reason: "Lokasi kehilangan dan penemuan sama.", missing: false },
        time: { level: "high", reason: "Waktu kehilangan dan penemuan sangat dekat.", missing: false },
        category: { reason: "Kategori sama: Kartu Identitas.", missing: false },
      },
    },
  ],
}

describe("ReviewBoardSearch", () => {
  it("renders report type select and code input", () => {
    renderUi(<ReviewBoardSearch initialType="found" initialId="TEM-2610-0001" />)
    expect(screen.getByRole("combobox")).toHaveValue("found")
    expect(screen.getByRole("textbox")).toHaveValue("TEM-2610-0001")
    expect(screen.getByRole("button", { name: "Buka Papan" })).toBeInTheDocument()
  })
})

describe("ReviewBoardView", () => {
  it("renders selected report details side by side with candidates", () => {
    renderUi(<ReviewBoardView board={mockBoard} currentType="found" currentId="TEM-2610-0001" />)

    // Selected report
    expect(screen.getByText("TEM-2610-0001")).toBeInTheDocument()
    expect(screen.getByText("KTM Mahasiswa UGM")).toBeInTheDocument()
    expect(screen.getByText("Detail Tersembunyi")).toBeInTheDocument()
    expect(screen.getByText("Nama di KTM: Budi Santoso, NIM 22/123456/TK/54321")).toBeInTheDocument()

    // Candidate
    expect(screen.getByText("HIL-2610-0001")).toBeInTheDocument()
    expect(screen.getByText("KTM UGM Kuning")).toBeInTheDocument()
    expect(screen.getByText("92%")).toBeInTheDocument()
    expect(screen.getByText("Kandidat kuat")).toBeInTheDocument()
    expect(screen.getByText("Nama cocok")).toBeInTheDocument()
    expect(screen.getByText("Pemilik: Budi Santoso")).toBeInTheDocument()

    // Actions
    expect(screen.getByRole("button", { name: /Lepas tanda/i })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Undang klaim/i })).toBeInTheDocument()
  })
})
