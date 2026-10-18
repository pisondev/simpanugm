import { ShieldCheck, TriangleAlert, Users, Warehouse } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { PageHeader } from "@/components/layout/page-header"
import { ButtonLink } from "@/components/ui/button-link"
import { load, serverApi } from "@/lib/api/server"
import type { AdminUser } from "@/lib/api/types"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("admin")
  return { title: t("homeTitle") }
}

async function everyone(api: Awaited<ReturnType<typeof serverApi>>, role: "officer" | "admin"): Promise<AdminUser[]> {
  const out: AdminUser[] = []
  let cursor: string | undefined
  // staff accounts number in the tens, a few pages at most
  for (let page = 0; page < 10; page++) {
    const list = await load(api.GET("/admin/users", { params: { query: { role, cursor, limit: 100 } } }))
    out.push(...list.items)
    if (!list.next_cursor) {
      break
    }
    cursor = list.next_cursor
  }
  return out
}

// US-04: every post needs accountable staff, so posts without an active officer are surfaced first
export default async function AdminHomePage() {
  const api = await serverApi()
  const [officers, admins, posts, t] = await Promise.all([everyone(api, "officer"), everyone(api, "admin"), load(api.GET("/posts")), getTranslations("admin")])
  const active = officers.filter((o) => o.status === "active")
  const covered = new Set(active.flatMap((o) => o.post_ids))
  const uncovered = posts.items.filter((p) => !covered.has(p.id))
  const cards = [
    { label: t("officers"), value: active.length, icon: Users },
    { label: t("admins"), value: admins.filter((a) => a.status === "active").length, icon: ShieldCheck },
    { label: t("postsCount"), value: posts.items.length, icon: Warehouse },
  ]
  return (
    <>
      <PageHeader title={t("homeTitle")} lead={t("homeLead")} actions={<ButtonLink href="/admin/pengguna">{t("usersTitle")}</ButtonLink>} />
      <div className="mb-8 grid gap-3 sm:grid-cols-3">
        {cards.map((c) => (
          <div key={c.label} className="flex items-center gap-4 rounded-xl border bg-card p-4 shadow-sm">
            <span className="flex size-11 items-center justify-center rounded-full bg-primary-soft text-primary">
              <c.icon className="size-5" aria-hidden />
            </span>
            <span>
              <span className="block text-2xl font-semibold">{c.value}</span>
              <span className="block text-sm text-muted-foreground">{c.label}</span>
            </span>
          </div>
        ))}
      </div>
      <section aria-labelledby="tanpa-petugas" className="flex flex-col gap-3">
        <h2 id="tanpa-petugas" className="text-lg font-semibold">
          {t("uncoveredTitle")}
        </h2>
        {uncovered.length === 0 ? (
          <p className="rounded-xl border bg-card p-4 text-muted-foreground">{t("allCovered")}</p>
        ) : (
          <ul className="flex flex-col divide-y rounded-xl border border-warning/30 bg-warning/5">
            {uncovered.map((p) => (
              <li key={p.id} className="flex items-center gap-3 p-3">
                <TriangleAlert className="size-5 shrink-0 text-warning" aria-hidden />
                <span className="flex-1">
                  <span className="block font-medium">{p.name}</span>
                  <span className="block text-sm text-muted-foreground">
                    {p.location_name} | {p.zone_name}
                  </span>
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </>
  )
}
