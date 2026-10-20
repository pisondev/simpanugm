import { act, screen } from "@testing-library/react"
import { afterEach, describe, expect, it, vi } from "vitest"

import { CandidateStatus } from "@/components/report/candidate-status"

import { renderUi } from "./render"

describe("CandidateStatus", () => {
  afterEach(() => vi.useRealTimers())

  it("replaces the searching state after the two-minute window without a page refresh", async () => {
    vi.useFakeTimers()
    renderUi(<CandidateStatus candidateCount={0} createdAt="2026-10-20T20:00:00.000Z" now={Date.parse("2026-10-20T20:01:00.000Z")} />)

    expect(screen.getByText("Mencari kandidat...")).toBeVisible()
    await act(async () => vi.advanceTimersByTime(60_000))
    expect(screen.getByText("Belum ada kandidat. Kami akan memberi tahu kamu saat ada barang yang mirip.")).toBeVisible()
  })
})
