import "server-only"

import { redirect } from "next/navigation"
import { cache } from "react"

import { serverApi, signInUrl } from "./api/server"
import type { Me } from "./api/types"
import { homeFor } from "./paths"

export { homeFor }

export const getMe = cache(async (): Promise<Me | null> => {
  const api = await serverApi()
  const { data, response } = await api.GET("/me")
  if (response.status === 401) {
    return null
  }
  if (!response.ok || !data) {
    throw new Error(`me: ${response.status}`)
  }
  return data
})

export async function requireMe(): Promise<Me> {
  const me = await getMe()
  if (!me) {
    redirect(await signInUrl(true))
  }
  return me
}

export async function requireStaff(): Promise<Me> {
  const me = await requireMe()
  if (me.role === "member") {
    redirect("/beranda")
  }
  return me
}

export async function requireAdmin(): Promise<Me> {
  const me = await requireMe()
  if (me.role !== "admin") {
    redirect(homeFor(me.role))
  }
  return me
}

export function can(me: Me, permission: string): boolean {
  return me.permissions.includes(permission)
}
