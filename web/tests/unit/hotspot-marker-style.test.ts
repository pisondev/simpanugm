import { describe, expect, it } from "vitest"

import { hotspotMarkerStyle } from "@/components/admin/hotspot-marker-style"

function contrastAgainstWhite(hex: string) {
  const channels = [1, 3, 5].map((start) => {
    const value = parseInt(hex.slice(start, start + 2), 16) / 255
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4
  })
  const [r = 0, g = 0, b = 0] = channels
  return 1.05 / (0.2126 * r + 0.7152 * g + 0.0722 * b + 0.05)
}

describe("hotspot markers", () => {
  it("keeps masked counts visually indistinguishable", () => {
    const reference = hotspotMarkerStyle(0, 100)
    for (const count of [1, 2]) expect(hotspotMarkerStyle(count, 100)).toEqual(reference)
    expect(hotspotMarkerStyle(3, 100).diameter).toBeGreaterThan(reference.diameter)
  })

  it("keeps white normal-size labels above WCAG AA contrast in every count band", () => {
    for (const count of [0, 1, 2, 3, 4, 9, 10, 100]) {
      expect(contrastAgainstWhite(hotspotMarkerStyle(count, 100).background)).toBeGreaterThanOrEqual(4.5)
    }
  })
})
