// WIB has no daylight saving, so a fixed offset is exact
const offset = "+07:00"

export function todayInWib(now: Date = new Date()): string {
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Jakarta" }).format(now)
}

export function nowTimeInWib(now: Date = new Date()): string {
  return new Intl.DateTimeFormat("en-GB", { timeZone: "Asia/Jakarta", hour: "2-digit", minute: "2-digit", hour12: false }).format(now)
}

export function wibToIso(date: string, time: string): string {
  return new Date(`${date}T${time}:00${offset}`).toISOString()
}

export function isoToWib(iso: string): { date: string; time: string } {
  const d = new Date(iso)
  return { date: todayInWib(d), time: nowTimeInWib(d) }
}

export const presets = {
  pagi: { from: "06:00", until: "11:00" },
  siang: { from: "11:00", until: "15:00" },
  sore: { from: "15:00", until: "18:00" },
  malam: { from: "18:00", until: "23:59" },
  sehari: { from: "00:00", until: "23:59" },
} as const

export type Preset = keyof typeof presets

export type WindowInput = {
  date: string
  endDate?: string
  preset: Preset | "custom"
  from?: string
  until?: string
}

// the end never lies in the future, BR-LST-01 allows only five minutes of clock skew
export function lostWindow(input: WindowInput, now: Date = new Date()): { lost_from: string; lost_until: string } {
  const range = input.preset === "custom" ? { from: input.from ?? "00:00", until: input.until ?? "23:59" } : presets[input.preset]
  const fromIso = wibToIso(input.date, range.from)
  let untilIso = wibToIso(input.endDate ?? input.date, range.until)
  if (new Date(untilIso).getTime() > now.getTime()) {
    untilIso = now.toISOString()
  }
  return { lost_from: fromIso, lost_until: untilIso }
}

export type WindowProblem = "order" | "maxSpan" | "maxAge" | "future"

// mirrors BR-LST-01 so the form refuses early, the api stays the authority
export function windowProblem(window: { lost_from: string; lost_until: string }, now: Date = new Date()): WindowProblem | null {
  const from = new Date(window.lost_from).getTime()
  const until = new Date(window.lost_until).getTime()
  if (from > now.getTime() + 5 * 60000) return "future"
  if (from > until) return "order"
  if (until - from > 7 * 86400000) return "maxSpan"
  if (from < now.getTime() - 90 * 86400000) return "maxAge"
  return null
}

export type FoundTimeProblem = "future" | "maxAge"

export function foundTimeProblem(iso: string, now: Date = new Date()): FoundTimeProblem | null {
  const at = new Date(iso).getTime()
  if (at > now.getTime() + 5 * 60000) return "future"
  if (at < now.getTime() - 30 * 86400000) return "maxAge"
  return null
}
