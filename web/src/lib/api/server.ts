import "server-only"

import { cookies, headers } from "next/headers"
import { notFound, redirect } from "next/navigation"
import createClient from "openapi-fetch"

import type { paths } from "./schema"

const base = `${process.env.API_INTERNAL_URL ?? "http://localhost:8090"}/api/v1`

// server components call api over the internal network, forwarding the session and the client address
export async function serverApi() {
  const [c, h] = await Promise.all([cookies(), headers()])
  const forward: Record<string, string> = { cookie: c.toString() }
  const xff = h.get("x-forwarded-for")
  if (xff) {
    forward["x-forwarded-for"] = xff
  }
  const requestId = h.get("x-request-id")
  if (requestId) {
    forward["x-request-id"] = requestId
  }
  return createClient<paths>({ baseUrl: base, headers: forward, cache: "no-store" })
}

export async function currentPath(): Promise<string> {
  return (await headers()).get("x-simpan-path") ?? "/"
}

export async function signInUrl(expired = false): Promise<string> {
  const next = encodeURIComponent(await currentPath())
  return `/masuk?next=${next}${expired ? "&expired=1" : ""}`
}

type Result<T> = { data?: T; error?: unknown; response: Response }

// records out of scope answer 404 so a page never confirms that a record exists (information-architecture 4)
export async function load<T>(request: Promise<Result<T>>): Promise<T> {
  const { data, response } = await request
  if (response.ok && data !== undefined) {
    return data
  }
  if (response.status === 401) {
    redirect(await signInUrl(true))
  }
  if (response.status === 403 || response.status === 404) {
    notFound()
  }
  throw new Error(`api answered ${response.status} for ${new URL(response.url).pathname}`)
}
