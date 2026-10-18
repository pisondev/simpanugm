import { describe, expect, it } from "vitest"

import { ApiError, call, fieldErrors, messageFor, type ErrorTranslator } from "@/lib/api/problem"
import messages from "@/messages/id.json"

const t: ErrorTranslator = (key, values) => {
  let text: string = messages.errors[key]
  for (const [k, v] of Object.entries(values ?? {})) {
    text = text.replace(`{${k}}`, String(v))
  }
  return text
}

function problem(status: number, extra: Record<string, unknown> = {}) {
  return new ApiError(status, { type: "about:blank", title: "Judul", status, code: "x", ...extra })
}

describe("problem mapping, ui-guidelines 6.4", () => {
  it("rounds the retry time up to whole minutes", () => {
    expect(messageFor(problem(429, { retry_after_seconds: 61 }), t)).toBe("Terlalu banyak percobaan. Coba lagi dalam 2 menit.")
  })

  it("shows the request id for server errors", () => {
    expect(messageFor(problem(500, { request_id: "req-1" }), t)).toBe("Terjadi kesalahan di sisi kami. Kode: req-1")
  })

  it("prefers the Indonesian detail from api", () => {
    expect(messageFor(problem(422, { detail: "Email sudah terdaftar" }), t)).toBe("Email sudah terdaftar")
  })

  it("names a dropped connection", () => {
    expect(messageFor(new TypeError("fetch failed"), t)).toBe("Gagal terhubung. Coba lagi.")
  })

  it("keeps the first message per field", () => {
    const err = problem(422, {
      errors: [
        { field: "description", code: "too_short", message: "pertama" },
        { field: "description", code: "x", message: "kedua" },
      ],
    })
    expect(fieldErrors(err)).toEqual({ description: "pertama" })
  })

  it("turns a non 2xx answer into an ApiError", async () => {
    const response = new Response(null, { status: 404 })
    await expect(
      call(Promise.resolve({ error: { title: "Tidak ditemukan", code: "not_found", type: "about:blank", status: 404 }, response })),
    ).rejects.toMatchObject({
      status: 404,
      code: "not_found",
    })
  })
})
