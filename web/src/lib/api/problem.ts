import type { Problem } from "./types"

export class ApiError extends Error {
  readonly status: number
  readonly problem: Problem

  constructor(status: number, problem: Problem) {
    super(problem.detail ?? problem.title)
    this.name = "ApiError"
    this.status = status
    this.problem = problem
  }

  get code(): string {
    return this.problem.code
  }

  get requestId(): string | undefined {
    return this.problem.request_id
  }
}

function asProblem(body: unknown, status: number): Problem {
  if (body && typeof body === "object" && "code" in body && "title" in body) {
    return body as Problem
  }
  return { type: "about:blank", title: "Terjadi kesalahan", status, code: status >= 500 ? "internal" : "unknown" }
}

type Result<T> = { data?: T; error?: unknown; response: Response }

// openapi-fetch resolves on every status, this turns non 2xx answers into thrown ApiErrors
export async function call<T>(request: Promise<Result<T>>): Promise<T> {
  const { data, error, response } = await request
  if (response.ok) {
    return data as T
  }
  throw new ApiError(response.status, asProblem(error, response.status))
}

export function fieldErrors(err: unknown): Record<string, string> {
  if (!(err instanceof ApiError) || !err.problem.errors) {
    return {}
  }
  const out: Record<string, string> = {}
  for (const f of err.problem.errors) {
    if (f.field && !out[f.field]) {
      out[f.field] = f.message
    }
  }
  return out
}

export type ErrorKey = "network" | "server" | "rateLimited" | "rateLimitedLater" | "generic"

export type ErrorTranslator = (key: ErrorKey, values?: Record<string, string | number>) => string

// the api already answers in Indonesian, only transport level failures need our own words
export function messageFor(err: unknown, t: ErrorTranslator): string {
  if (err instanceof ApiError) {
    if (err.status === 429) {
      const wait = err.problem.retry_after_seconds
      return wait ? t("rateLimited", { minutes: Math.max(1, Math.ceil(wait / 60)) }) : t("rateLimitedLater")
    }
    if (err.status >= 500) {
      return t("server", { id: err.requestId ?? "-" })
    }
    return err.problem.detail ?? err.problem.title
  }
  if (err instanceof TypeError) {
    return t("network")
  }
  return t("generic")
}

export function isApiError(err: unknown, status?: number, code?: string): err is ApiError {
  return err instanceof ApiError && (status === undefined || err.status === status) && (code === undefined || err.code === code)
}
