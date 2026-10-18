import { CalendarClock, Hand, PackageOpen, Warehouse } from "lucide-react"
import type { Metadata } from "next"
import Link from "next/link"
import { getTranslations } from "next-intl/server"

import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { DueLabel } from "@/components/officer/due-label"
import { StatusBadge } from "@/components/report/status-badge"
import { ButtonLink } from "@/components/ui/button-link"
import { load, serverApi } from "@/lib/api/server"
import type { InventoryItem, SearchResult } from "@/lib/api/types"
import { daysUntil, formatRelative } from "@/lib/format"
import { requestTime } from "@/lib/clock"
import { requireStaff } from "@/lib/session"
import { staffPosts } from "@/lib/staff"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("officer")
  return { title: t("homeTitle") }
}

export default async function OfficerHomePage() {
  const now = await requestTime()
  const me = await requireStaff()
  const [api, posts, t] = await Promise.all([serverApi(), staffPosts(me), getTranslations("officer")])
  if (posts.length === 0) {
    return (
      <>
        <PageHeader title={t("homeTitle")} />
        <EmptyState icon={Warehouse}>{t("noPosts")}</EmptyState>
      </>
    )
  }
  const [inventories, awaitingLists] = await Promise.all([
    Promise.all(posts.map((p) => load(api.GET("/posts/{id}/inventory", { params: { path: { id: p.id } } })))),
    Promise.all(posts.map((p) => load(api.GET("/search", { params: { query: { type: "found", custody: "with_finder", post_id: p.id, limit: 20 } } })))),
  ])
  const stock: (InventoryItem & { post: string })[] = inventories.flatMap((inv) => inv.items.map((i) => ({ ...i, post: inv.post.name })))
  const due = stock.filter((i) => daysUntil(i.retention_due_at, now) <= 7).sort((a, b) => a.retention_due_at.localeCompare(b.retention_due_at))
  const seen = new Set<string>()
  const awaiting: SearchResult[] = awaitingLists.flatMap((l) => l.items).filter((r) => (seen.has(r.id) ? false : (seen.add(r.id), true)))
  const counters = [
    { label: t("countInCustody"), value: stock.length, icon: Warehouse, href: "/petugas/inventaris" },
    { label: t("countDueSoon"), value: due.length, icon: CalendarClock, href: "/petugas/inventaris" },
    { label: t("countWithFinder"), value: awaiting.length, icon: Hand, href: "#menunggu" },
  ]
  return (
    <>
      <PageHeader
        title={t("homeTitle")}
        lead={t("homeLead", { posts: posts.map((p) => p.name).join(", ") })}
        actions={
          <ButtonLink href="/petugas/terima">
            <PackageOpen aria-hidden />
            {t("receiveTitle")}
          </ButtonLink>
        }
      />
      <div className="mb-8 grid gap-3 sm:grid-cols-3">
        {counters.map((c) => (
          <Link key={c.label} href={c.href} className="flex items-center gap-4 rounded-xl border bg-card p-4 shadow-sm hover:border-primary/40">
            <span className="flex size-11 items-center justify-center rounded-full bg-primary-soft text-primary">
              <c.icon className="size-5" aria-hidden />
            </span>
            <span>
              <span className="block text-2xl font-semibold">{c.value}</span>
              <span className="block text-sm text-muted-foreground">{c.label}</span>
            </span>
          </Link>
        ))}
      </div>
      <div className="grid gap-8 xl:grid-cols-2">
        <section id="menunggu" aria-labelledby="menunggu-judul" className="flex flex-col gap-3">
          <h2 id="menunggu-judul" className="text-lg font-semibold">
            {t("awaitingTitle")}
          </h2>
          {awaiting.length === 0 ? (
            <EmptyState icon={Hand}>{t("awaitingEmpty")}</EmptyState>
          ) : (
            <ul className="flex flex-col divide-y rounded-xl border bg-card">
              {awaiting.map((r, i) => (
                <li key={r.id} className={i === 0 ? "bg-primary-soft/30" : undefined}>
                  <div className="flex flex-wrap items-center gap-3 p-3">
                    <div className="min-w-0 flex-1">
                      <Link href={`/petugas/laporan/temuan/${r.code}`} className="font-medium hover:underline">
                        {r.item_name}
                      </Link>
                      <p className="text-sm text-muted-foreground">
                        <span className="font-mono">{r.code}</span> | {r.category_name} | {r.place_name ?? ""} | {formatRelative(r.occurred_at, now)}
                      </p>
                    </div>
                    <ButtonLink href={`/petugas/terima?code=${r.code}`} size="sm" variant={i === 0 ? "default" : "outline"}>
                      {t("receive")}
                    </ButtonLink>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </section>
        <section aria-labelledby="jatuh-tempo" className="flex flex-col gap-3">
          <h2 id="jatuh-tempo" className="text-lg font-semibold">
            {t("dueTitle")}
          </h2>
          {due.length === 0 ? (
            <EmptyState icon={CalendarClock}>{t("dueEmpty")}</EmptyState>
          ) : (
            <ul className="flex flex-col divide-y rounded-xl border bg-card">
              {due.slice(0, 10).map((item) => (
                <li key={item.id} className="flex flex-wrap items-center gap-3 p-3">
                  <div className="min-w-0 flex-1">
                    <Link href={`/petugas/laporan/temuan/${item.code}`} className="font-medium hover:underline">
                      {item.item_name}
                    </Link>
                    <p className="text-sm text-muted-foreground">
                      <span className="font-mono">{item.code}</span> | {item.post} | {item.storage_slot}
                    </p>
                  </div>
                  <StatusBadge kind="found" status={item.status} />
                  <DueLabel iso={item.retention_due_at} now={now} />
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </>
  )
}
