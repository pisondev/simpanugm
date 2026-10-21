export function hotspotMarkerStyle(count: number, maxCount: number) {
  const visibleCount = count < 3 ? 0 : count
  const background = visibleCount >= 10 ? "#e11d48" : visibleCount >= 4 ? "#b45309" : visibleCount === 0 ? "#64748b" : "#047857"
  return {
    diameter: visibleCount === 0 ? 28 : Math.min(52, 28 + (visibleCount / Math.max(maxCount, 1)) * 24),
    background,
  }
}
