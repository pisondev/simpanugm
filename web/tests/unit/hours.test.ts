import { describe, expect, it } from "vitest"

import { parseRanges, rangesText } from "@/lib/hours"

// the opening hours editor of US-07 follows the seed contract: spans from before to, an empty day is closed
describe("opening hours text", () => {
  it("reads spans separated by commas and writes them back", () => {
    const ranges = parseRanges("08:00-11:30, 13:00 - 16:00")
    expect(ranges).toEqual([
      ["08:00", "11:30"],
      ["13:00", "16:00"],
    ])
    expect(rangesText(ranges ?? [])).toBe("08:00-11:30, 13:00-16:00")
  })

  it("treats an empty day as closed", () => {
    expect(parseRanges("  ")).toEqual([])
  })

  it("refuses malformed or backwards spans", () => {
    expect(parseRanges("8-16")).toBeNull()
    expect(parseRanges("16:00-08:00")).toBeNull()
    expect(parseRanges("08:00-16:00, 25:00-26:00")).toBeNull()
  })
})
