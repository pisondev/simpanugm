import { describe, expect, it } from "vitest"

import { foundTimeProblem, isoToWib, lostWindow, windowProblem, wibToIso } from "@/lib/time"

const now = new Date("2026-11-02T05:00:00Z") // 12.00 WIB

describe("WIB conversion", () => {
  it("round trips a WIB date and time", () => {
    const iso = wibToIso("2026-11-02", "08:15")
    expect(iso).toBe("2026-11-02T01:15:00.000Z")
    expect(isoToWib(iso)).toEqual({ date: "2026-11-02", time: "08:15" })
  })
})

describe("lost window, BR-LST-01", () => {
  it("fills a preset part of day", () => {
    expect(lostWindow({ date: "2026-11-01", preset: "pagi" }, now)).toEqual({
      lost_from: "2026-10-31T23:00:00.000Z",
      lost_until: "2026-11-01T04:00:00.000Z",
    })
  })

  it("never ends in the future", () => {
    const w = lostWindow({ date: "2026-11-02", preset: "sehari" }, now)
    expect(w.lost_until).toBe(now.toISOString())
    expect(windowProblem(w, now)).toBeNull()
  })

  it("refuses an evening preset later today as a future time", () => {
    expect(windowProblem(lostWindow({ date: "2026-11-02", preset: "malam" }, now), now)).toBe("future")
  })

  it("refuses a window of eight days", () => {
    expect(windowProblem(lostWindow({ date: "2026-10-24", endDate: "2026-11-01", preset: "sehari" }, now), now)).toBe("maxSpan")
  })

  it("accepts exactly seven days", () => {
    expect(windowProblem({ lost_from: "2026-10-26T00:00:00Z", lost_until: "2026-11-02T00:00:00Z" }, now)).toBeNull()
  })

  it("refuses a start more than 90 days back", () => {
    expect(windowProblem(lostWindow({ date: "2026-07-01", preset: "sehari" }, now), now)).toBe("maxAge")
  })

  it("refuses an end before the start", () => {
    expect(windowProblem(lostWindow({ date: "2026-11-01", preset: "custom", from: "15:00", until: "09:00" }, now), now)).toBe("order")
  })
})

describe("found time, BR-FND-01", () => {
  it("accepts the last 30 days and refuses the rest", () => {
    expect(foundTimeProblem("2026-11-01T00:00:00Z", now)).toBeNull()
    expect(foundTimeProblem("2026-09-01T00:00:00Z", now)).toBe("maxAge")
    expect(foundTimeProblem("2026-11-02T06:00:00Z", now)).toBe("future")
  })
})
