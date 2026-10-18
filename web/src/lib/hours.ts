export const weekdays = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"] as const

export type Weekday = (typeof weekdays)[number]

export type OpeningHours = { [day: string]: string[][] }

function wibParts(now: Date): { day: Weekday; minutes: number } {
  const p = new Intl.DateTimeFormat("en-GB", {
    timeZone: "Asia/Jakarta",
    weekday: "short",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).formatToParts(now)
  const get = (type: string) => p.find((x) => x.type === type)?.value ?? ""
  const day = get("weekday").toLowerCase().slice(0, 3) as Weekday
  return { day, minutes: Number(get("hour")) * 60 + Number(get("minute")) }
}

function toMinutes(hhmm: string): number {
  const [h, m] = hhmm.split(":").map(Number)
  return (h ?? 0) * 60 + (m ?? 0)
}

export function rangesFor(hours: OpeningHours, day: Weekday): string[][] {
  return (hours[day] ?? []).filter((r) => r.length === 2)
}

export function formatRanges(ranges: string[][]): string {
  return ranges.map(([from, to]) => `${(from ?? "").replace(":", ".")} sampai ${(to ?? "").replace(":", ".")}`).join(", ")
}

export function isOpenNow(hours: OpeningHours, now: Date = new Date()): boolean {
  const { day, minutes } = wibParts(now)
  return rangesFor(hours, day).some(([from, to]) => minutes >= toMinutes(from ?? "") && minutes < toMinutes(to ?? ""))
}

export function today(now: Date = new Date()): Weekday {
  return wibParts(now).day
}

const span = /^([01]\d|2[0-3]):([0-5]\d)\s*-\s*([01]\d|2[0-3]):([0-5]\d)$/

// "08:00-11:30, 13:00-16:00" into [["08:00","11:30"],["13:00","16:00"]], null when a part is not a span from before to
export function parseRanges(text: string): string[][] | null {
  const parts = text
    .split(",")
    .map((p) => p.trim())
    .filter(Boolean)
  const out: string[][] = []
  for (const p of parts) {
    const m = span.exec(p)
    if (!m) return null
    const from = `${m[1]}:${m[2]}`
    const to = `${m[3]}:${m[4]}`
    if (from >= to) return null
    out.push([from, to])
  }
  return out
}

export function rangesText(ranges: string[][]): string {
  return ranges.map(([from, to]) => `${from}-${to}`).join(", ")
}
