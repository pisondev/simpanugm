import { screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { StatusBadge } from "@/components/report/status-badge"

import { renderUi } from "./render"

// NFR-USE-05: a status is a word and an icon, never colour alone
describe("StatusBadge", () => {
  it("renders the label next to a hidden icon", () => {
    const { container } = renderUi(<StatusBadge kind="found" status="potential_match" />)
    expect(screen.getByText("Ada kandidat pemilik")).toBeInTheDocument()
    expect(container.querySelector("svg")).toHaveAttribute("aria-hidden")
  })

  it("uses the custody vocabulary for custody", () => {
    renderUi(<StatusBadge kind="custody" status="in_custody" />)
    expect(screen.getByText("Disimpan di pos")).toBeInTheDocument()
  })

  it("uses the claim vocabulary for claims", () => {
    renderUi(<StatusBadge kind="claim" status="under_review" />)
    expect(screen.getByText("Sedang Ditinjau")).toBeInTheDocument()
  })
})
