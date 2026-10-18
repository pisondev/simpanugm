import { describe, expect, it } from "vitest"

import { daysUntil, formatDate, formatDateTime, formatDistance, formatRange, formatRelative } from "@/lib/format"

// ui-guidelines 6.3, the test clock runs in UTC so every expectation proves the WIB conversion
describe("format", () => {
  it("writes a date and time in WIB", () => {
    expect(formatDateTime("2026-11-02T07:30:00Z")).toBe("Senin, 2 Nov 2026, 14.30 WIB")
  })

  it("moves an evening UTC time to the next WIB day", () => {
    expect(formatDate("2026-11-01T18:00:00Z")).toBe("2 Nov 2026")
  })

  it("shortens a range within one day", () => {
    expect(formatRange("2026-11-02T01:00:00Z", "2026-11-02T04:00:00Z")).toBe("Senin, 2 Nov 2026, 08.00 sampai 11.00 WIB")
  })

  it("rounds distances to 50 m and uses a decimal comma", () => {
    expect(formatDistance(130)).toBe("sekitar 150 m")
    expect(formatDistance(10)).toBe("sekitar 50 m")
    expect(formatDistance(1234)).toBe("1,2 km")
  })

  it("uses relative time under seven days and an absolute date beyond", () => {
    const now = new Date("2026-11-10T00:00:00Z")
    expect(formatRelative("2026-11-09T23:55:00Z", now)).toBe("5 menit yang lalu")
    expect(formatRelative("2026-11-09T00:00:00Z", now)).toBe("kemarin")
    expect(formatRelative("2026-10-01T00:00:00Z", now)).toBe("1 Okt 2026")
  })

  it("counts days until a due date", () => {
    const now = new Date("2026-11-10T00:00:00Z")
    expect(daysUntil("2026-11-12T00:00:00Z", now)).toBe(2)
    expect(daysUntil("2026-11-09T00:00:00Z", now)).toBe(-1)
  })
})
