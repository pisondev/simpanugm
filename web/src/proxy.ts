import { NextResponse, type NextRequest } from "next/server"

const protectedPrefixes = ["/beranda", "/lapor", "/laporan", "/barang-temuan", "/profil", "/petugas", "/admin"]

// only checks that a session cookie exists, the layouts ask api whether it is valid and which role it has
export function proxy(request: NextRequest) {
  const path = request.nextUrl.pathname
  const guarded = protectedPrefixes.some((p) => path === p || path.startsWith(`${p}/`))
  if (guarded && !request.cookies.get("simpan_session")?.value) {
    const url = request.nextUrl.clone()
    url.pathname = "/masuk"
    url.search = `?next=${encodeURIComponent(path + request.nextUrl.search)}`
    return NextResponse.redirect(url)
  }
  const headers = new Headers(request.headers)
  headers.set("x-simpan-path", path + request.nextUrl.search)
  return NextResponse.next({ request: { headers } })
}

export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico|icon.svg).*)"],
}
