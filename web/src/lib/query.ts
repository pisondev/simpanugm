export const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i
export const datePattern = /^\d{4}-\d{2}-\d{2}$/

type Param = string | string[] | undefined

// search params come from the address bar, only well formed values reach the api
export function pick(value: Param, pattern?: RegExp): string | undefined {
  if (typeof value !== "string" || value === "") {
    return undefined
  }
  return pattern && !pattern.test(value) ? undefined : value
}

export function pickOne<T extends string>(value: Param, allowed: readonly T[]): T | undefined {
  return typeof value === "string" && (allowed as readonly string[]).includes(value) ? (value as T) : undefined
}

export function queryString(values: Record<string, string | undefined>, extra?: Record<string, string>): string {
  const p = new URLSearchParams()
  for (const [k, v] of Object.entries({ ...values, ...extra })) {
    if (v) {
      p.set(k, v)
    }
  }
  const s = p.toString()
  return s ? `?${s}` : ""
}
