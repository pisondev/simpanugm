import { screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { ScoreBars } from "@/components/matching/score-bars"
import { percent, TierBadge } from "@/components/matching/tier-badge"
import type { MatchExplanation } from "@/lib/api/types"

import { renderUi } from "./render"

const explanation: MatchExplanation = {
  visual_mode: "cross_modal",
  visual: { level: "high", reason: "Deskripsimu sangat cocok dengan foto barang temuan.", missing: false },
  text: { level: "medium", reason: "Deskripsi cukup mirip.", chips: ["Warna sama: cokelat"], missing: false },
  location: { reason: "Lokasi kehilangan tidak diketahui.", missing: true },
  time: { level: "high", reason: "Ditemukan 5 jam setelah perkiraan waktu hilang.", missing: false },
  category: { reason: "Kategori sama.", missing: false },
}

// US-20 criteria 1 to 3
describe("ScoreBars", () => {
  it("names the visual bar after the cross modal comparison and keeps every reason as text", () => {
    renderUi(<ScoreBars explanation={explanation} />)
    expect(screen.getByText("Kecocokan deskripsi dengan foto")).toBeInTheDocument()
    expect(screen.queryByText("Kemiripan foto")).not.toBeInTheDocument()
    for (const label of ["Kemiripan deskripsi", "Kedekatan lokasi", "Kedekatan waktu"]) {
      expect(screen.getByText(label)).toBeInTheDocument()
    }
    expect(screen.getByText("Ditemukan 5 jam setelah perkiraan waktu hilang.")).toBeInTheDocument()
    expect(screen.getByText("Warna sama: cokelat")).toBeInTheDocument()
    expect(screen.getByText("Kategori sama.")).toBeInTheDocument()
  })

  it("exposes each bar as a meter with its reason, an absent signal reads zero", () => {
    renderUi(<ScoreBars explanation={explanation} />)
    const meters = screen.getAllByRole("meter")
    expect(meters).toHaveLength(4)
    expect(meters[0]).toHaveAttribute("aria-valuetext", "Deskripsimu sangat cocok dengan foto barang temuan.")
    expect(meters[2]).toHaveAttribute("aria-valuenow", "0")
    expect(screen.getByText("Belum ada")).toBeInTheDocument()
  })

  it("uses the officer numbers when they are given", () => {
    renderUi(<ScoreBars explanation={{ ...explanation, visual_mode: "image" }} scores={{ visual: 0.42, category_factor: 1 }} />)
    expect(screen.getByText("Kemiripan foto")).toBeInTheDocument()
    expect(screen.getAllByRole("meter")[0]).toHaveAttribute("aria-valuenow", "42")
  })
})

describe("TierBadge", () => {
  it("shows the tier word of ui-guidelines 4.4 and a whole percent", () => {
    renderUi(<TierBadge tier="high" />)
    expect(screen.getByText("Sangat mirip")).toBeInTheDocument()
    expect(percent(0.8124)).toBe("81%")
  })
})
