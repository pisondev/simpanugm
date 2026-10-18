import type { Role } from "./api/types"

// only same origin paths survive, an open redirect through ?next= is refused
export function safeNext(next: string | null | undefined, fallback: string): string {
  if (!next || !next.startsWith("/") || next.startsWith("//") || next.startsWith("/\\")) {
    return fallback
  }
  return next
}

export const codePattern = /^(LST|FND)-\d{4}-\d{4,}$/i

export function reportPath(code: string, staff: boolean): string {
  const upper = code.toUpperCase()
  const kind = upper.startsWith("LST") ? "hilang" : "temuan"
  return staff ? `/petugas/laporan/${kind}/${upper}` : `/laporan/${kind}/${upper}`
}

export function homeFor(role: Role): string {
  switch (role) {
    case "admin":
      return "/admin"
    case "officer":
      return "/petugas"
    default:
      return "/beranda"
  }
}

// the version the register form sends, it must equal CurrentPrivacyVersion in api/internal/auth
export const privacyVersion = "2026.10"
