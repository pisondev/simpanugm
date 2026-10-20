import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { http, HttpResponse } from "msw"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { ClaimForm, type LostOption, type Question } from "@/components/claims/claim-form"
import type { Category, ClaimCreate, Colour } from "@/lib/api/types"

import { apiBase, server } from "./msw"
import { renderUi } from "./render"

const push = vi.fn()
vi.mock("next/navigation", () => ({ useRouter: () => ({ push, back: vi.fn(), refresh: vi.fn(), replace: vi.fn() }) }))

const dummyCategories: Category[] = [
  {
    id: "cat-dompet-parent",
    slug: "barang-pribadi",
    name: "Barang Pribadi",
    children: [
      {
        id: "cat-dompet-child",
        slug: "dompet",
        name: "Dompet",
        parent_id: "cat-dompet-parent",
        verification_questions: [{ key: "isi_dompet", question: "Sebutkan isi dompet", required: true }],
      },
    ],
  },
]

const dummyColours: Colour[] = [
  { name: "Cokelat", hex: "#8B4513" },
  { name: "Hitam", hex: "#000000" },
]

const dummyQuestions: Question[] = [{ key: "isi_dompet", question: "Sebutkan isi dompet", required: true }]

const dummyLostOptions: LostOption[] = [{ id: "lost-001", code: "LST-2611-0001", label: "LST-2611-0001 - Dompet Kulit" }]

describe("ClaimForm inline lost report, US-26", () => {
  beforeEach(() => {
    push.mockReset()
  })

  it("renders inline lost report section when user has no active lost reports", async () => {
    renderUi(
      <ClaimForm
        foundId="found-123"
        foundName="Dompet Cokelat"
        questions={dummyQuestions}
        lostOptions={[]}
        categories={dummyCategories}
        colours={dummyColours}
        defaultCategoryId="cat-dompet-child"
      />,
    )

    expect(screen.getByText("Buat Laporan Kehilangan Sekaligus")).toBeInTheDocument()
    expect(screen.getByText("Detail Laporan Kehilangan Baru")).toBeInTheDocument()
    expect(screen.getByLabelText("Nama barang yang hilang")).toBeInTheDocument()
    expect(screen.getByLabelText("Deskripsi barang")).toBeInTheDocument()

    // Submit while empty triggers validation
    await userEvent.click(screen.getByRole("button", { name: "Kirim Pengajuan Klaim" }))
    expect(await screen.findAllByText("Wajib diisi")).not.toHaveLength(0)
    expect(push).not.toHaveBeenCalled()
  })

  it("submits claim with inline lost report payload", async () => {
    let capturedBody: ClaimCreate | null = null
    server.use(
      http.post(`${apiBase}/claims`, async ({ request }) => {
        capturedBody = (await request.json()) as ClaimCreate
        return HttpResponse.json(
          {
            id: "claim-999",
            code: "CLM-2611-0001",
            status: "submitted",
            found_report: { id: "found-123", code: "FND-2611-0001", item_name: "Dompet Cokelat" },
            lost_report: { id: "lost-new", code: "LST-2611-0099", item_name: "Dompet Cokelat Saya" },
            claimant: { id: "user-1", email: "user@mail.ugm.ac.id", full_name: "User UGM" },
            created_at: "2026-03-30T10:00:00Z",
            updated_at: "2026-03-30T10:00:00Z",
            version: 1,
          },
          { status: 201 },
        )
      }),
    )

    renderUi(
      <ClaimForm
        foundId="found-123"
        foundName="Dompet Cokelat"
        questions={dummyQuestions}
        lostOptions={[]}
        categories={dummyCategories}
        colours={dummyColours}
        defaultCategoryId="cat-dompet-child"
      />,
    )

    await userEvent.clear(screen.getByLabelText("Nama barang yang hilang"))
    await userEvent.type(screen.getByLabelText("Nama barang yang hilang"), "Dompet Cokelat Saya")
    await userEvent.type(screen.getByLabelText("Deskripsi barang"), "Dompet kulit cokelat merek Fossil dengan kartu identitas di dalamnya.")
    await userEvent.type(
      screen.getByLabelText("Mengapa barang Dompet Cokelat ini milikmu?"),
      "Ini dompet saya yang tertinggal di perpustakaan kemarin sore setelah belajar kelompok.",
    )
    await userEvent.type(screen.getByLabelText("Sebutkan isi dompet"), "KTM dan uang tunai")

    await userEvent.click(screen.getByRole("button", { name: "Kirim Pengajuan Klaim" }))

    await waitFor(() => {
      expect(push).toHaveBeenCalledWith("/klaim/CLM-2611-0001")
    })

    expect(capturedBody).not.toBeNull()
    const body = capturedBody!
    expect(body.found_report_id).toBe("found-123")
    expect(body.lost_report_id).toBeUndefined()
    expect(body.lost_report).toBeDefined()
    expect(body.lost_report?.item_name).toBe("Dompet Cokelat Saya")
    expect(body.lost_report?.category_id).toBe("cat-dompet-child")
    expect(body.lost_report?.location_unknown).toBe(true)
  }, 10_000)

  it("allows switching between existing lost report and creating a new one", async () => {
    renderUi(
      <ClaimForm
        foundId="found-123"
        foundName="Dompet Cokelat"
        questions={dummyQuestions}
        lostOptions={dummyLostOptions}
        categories={dummyCategories}
        colours={dummyColours}
        defaultCategoryId="cat-dompet-child"
      />,
    )

    // Initially selects existing report and does not show inline form
    expect(screen.queryByText("Detail Laporan Kehilangan Baru")).not.toBeInTheDocument()

    // Switch to "+ Buat laporan kehilangan baru sekaligus"
    await userEvent.selectOptions(screen.getByLabelText("Laporan kehilangan terkait"), "new")
    expect(screen.getByText("Detail Laporan Kehilangan Baru")).toBeInTheDocument()

    // Switch back to existing
    await userEvent.selectOptions(screen.getByLabelText("Laporan kehilangan terkait"), "lost-001")
    expect(screen.queryByText("Detail Laporan Kehilangan Baru")).not.toBeInTheDocument()
  })
})
