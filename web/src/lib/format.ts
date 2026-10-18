const zone = "Asia/Jakarta"

function parts(date: Date, options: Intl.DateTimeFormatOptions): Record<string, string> {
  const out: Record<string, string> = {}
  for (const p of new Intl.DateTimeFormat("id-ID", { timeZone: zone, ...options }).formatToParts(date)) {
    out[p.type] = p.value
  }
  return out
}

// "Senin, 2 Nov 2026, 14.30 WIB", ui-guidelines 6.3
export function formatDateTime(iso: string | Date): string {
  const d = typeof iso === "string" ? new Date(iso) : iso
  const p = parts(d, { weekday: "long", day: "numeric", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit", hour12: false })
  return `${p.weekday}, ${p.day} ${p.month} ${p.year}, ${p.hour}.${p.minute} WIB`
}

export function formatDate(iso: string | Date): string {
  const d = typeof iso === "string" ? new Date(iso) : iso
  const p = parts(d, { day: "numeric", month: "short", year: "numeric" })
  return `${p.day} ${p.month} ${p.year}`
}

export function formatTime(iso: string | Date): string {
  const d = typeof iso === "string" ? new Date(iso) : iso
  const p = parts(d, { hour: "2-digit", minute: "2-digit", hour12: false })
  return `${p.hour}.${p.minute}`
}

export function formatRange(fromIso: string, untilIso: string): string {
  const from = new Date(fromIso)
  const until = new Date(untilIso)
  if (formatDate(from) === formatDate(until)) {
    return `${formatDateTime(from).replace(" WIB", "")} sampai ${formatTime(until)} WIB`
  }
  return `${formatDateTime(from)} sampai ${formatDateTime(until)}`
}

const relative = new Intl.RelativeTimeFormat("id-ID", { numeric: "auto" })

export function formatRelative(iso: string | Date, now: Date = new Date()): string {
  const d = typeof iso === "string" ? new Date(iso) : iso
  const seconds = Math.round((d.getTime() - now.getTime()) / 1000)
  const abs = Math.abs(seconds)
  if (abs < 60) return relative.format(Math.trunc(seconds), "second")
  if (abs < 3600) return relative.format(Math.trunc(seconds / 60), "minute")
  if (abs < 86400) return relative.format(Math.trunc(seconds / 3600), "hour")
  if (abs < 7 * 86400) return relative.format(Math.trunc(seconds / 86400), "day")
  return formatDate(d)
}

export function formatDistance(meters: number): string {
  if (meters < 1000) {
    return `sekitar ${Math.max(50, Math.round(meters / 50) * 50)} m`
  }
  return `${(meters / 1000).toLocaleString("id-ID", { maximumFractionDigits: 1 })} km`
}

export function daysUntil(iso: string, now: Date = new Date()): number {
  return Math.ceil((new Date(iso).getTime() - now.getTime()) / 86400000)
}
