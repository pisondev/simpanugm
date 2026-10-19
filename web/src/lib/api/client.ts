import createClient from "openapi-fetch"

import type { paths } from "./schema"

// an absolute base keeps Request construction valid outside a page too, for example under test
const base = typeof window === "undefined" ? "/api/v1" : `${window.location.origin}/api/v1`

// fetch is looked up per call so test doubles and polyfills installed later are honoured
export const api = createClient<paths>({ baseUrl: base, credentials: "same-origin", fetch: (request) => globalThis.fetch(request) })

export function mediaUrl(id: string, variant: "thumb" | "display" | "original" | "thumb_blur" = "display"): string {
  return `/api/v1/media/${id}?variant=${variant}`
}

export function idempotencyKey(): string {
  return crypto.randomUUID()
}

export type UploadPurpose = "lost_report" | "found_report" | "claim_evidence"

// openapi-fetch does not stream multipart progress, so uploads use XMLHttpRequest
export function uploadMedia(file: Blob, purpose: UploadPurpose, onProgress?: (fraction: number) => void): Promise<{ id: string }> {
  return new Promise((resolve, reject) => {
    const form = new FormData()
    form.append("purpose", purpose)
    form.append("file", file, file instanceof File ? file.name : "foto.jpg")
    const xhr = new XMLHttpRequest()
    xhr.open("POST", "/api/v1/media")
    xhr.withCredentials = true
    xhr.upload.onprogress = (e) => {
      if (e.lengthComputable && onProgress) {
        onProgress(e.loaded / e.total)
      }
    }
    xhr.onload = () => {
      let body: unknown = null
      try {
        body = JSON.parse(xhr.responseText)
      } catch {
        body = null
      }
      if (xhr.status === 201 && body && typeof body === "object" && "id" in body) {
        resolve({ id: String((body as { id: string }).id) })
      } else {
        reject(Object.assign(new Error("upload failed"), { status: xhr.status, body }))
      }
    }
    xhr.onerror = () => reject(new TypeError("network"))
    xhr.send(form)
  })
}
