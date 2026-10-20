import { CalendarClock, ClipboardCheck, CopyCheck, Hand, PackageOpen, Sparkles, Warehouse } from "lucide-react"
import type { Metadata } from "next"
import Link from "next/link"
import { getTranslations } from "next-intl/server"

import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { StatusBadge } from "@/components/report/status-badge"
import { Badge } from "@/components/ui/badge"
import { ButtonLink } from "@/components/ui/button-link"
import { load, serverApi } from "@/lib/api/server"
import type { OfficerQueueItem, OfficerQueueItemType } from "@/lib/api/types"
import { formatRelative } from "@/lib/format"
import { requestTime } from "@/lib/clock"
import { requireStaff } from "@/lib/session"
import { staffPosts } from "@/lib/staff"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("officer")
  return { title: t("homeTitle") }
}

interface PageProps {
  searchParams?: Promise<{ scope?: string; type?: string }>
}

export default async function OfficerHomePage({ searchParams }: PageProps) {
  const params = searchParams ? await searchParams : {}
  const scope = params.scope === "campus" ? "campus" : "posts"
  const typeFilter = params.type as OfficerQueueItemType | undefined

  const now = await requestTime()
  const me = await requireStaff()
  const [api, posts, t] = await Promise.all([serverApi(), staffPosts(me), getTranslations("officer")])

  if (posts.length === 0 && me.role !== "admin") {
    return (
      <>
        <PageHeader title={t("homeTitle")} />
        <EmptyState icon={Warehouse}>{t("noPosts")}</EmptyState>
      </>
    )
  }

  const [summary, queueList] = await Promise.all([
    load(api.GET("/officer/summary", { params: { query: { scope } } })),
    load(api.GET("/officer/queue", { params: { query: { scope, type: typeFilter, limit: 30 } } })),
  ])

  const counters = [
    { label: t("countInCustody"), value: summary.in_custody, icon: Warehouse, href: "/petugas/inventaris" },
    { label: t("countDueSoon"), value: summary.retention_due, icon: CalendarClock, href: "/petugas/inventaris" },
    { label: t("countWithFinder"), value: summary.with_finder, icon: Hand, href: "/petugas/terima" },
    { label: t("countClaimsPending"), value: summary.claims_pending, icon: ClipboardCheck, href: "/petugas/klaim" },
    {
      label: t("countHighMatch"),
      value: summary.high_match_no_claim,
      icon: Sparkles,
      href: `/petugas?scope=${scope}&type=high_match_no_claim`,
    },
  ]

  const leadText = scope === "campus" ? "Tampilan seluruh pos kampus UGM." : t("homeLead", { posts: posts.map((p) => p.name).join(", ") })

  return (
    <>
      <PageHeader
        title={t("homeTitle")}
        lead={leadText}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <div className="inline-flex rounded-lg border bg-muted p-1 text-sm">
              <Link
                href="/petugas"
                className={`rounded-md px-3 py-1 font-medium transition-colors ${
                  scope === "posts" ? "bg-background text-foreground shadow-xs" : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {t("scopePosts")}
              </Link>
              <Link
                href="/petugas?scope=campus"
                className={`rounded-md px-3 py-1 font-medium transition-colors ${
                  scope === "campus" ? "bg-background text-foreground shadow-xs" : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {t("scopeCampus")}
              </Link>
            </div>
            <ButtonLink href="/petugas/papan" variant="outline">
              <Sparkles aria-hidden />
              {t("boardTitle")}
            </ButtonLink>
            <ButtonLink href="/petugas/duplikat" variant="outline">
              <CopyCheck aria-hidden />
              {t("duplicatesLink")}
            </ButtonLink>
            <ButtonLink href="/petugas/terima">
              <PackageOpen aria-hidden />
              {t("receiveTitle")}
            </ButtonLink>
          </div>
        }
      />

      <nav aria-label={t("homeTitle")} className="mb-8 grid gap-2 sm:grid-cols-2 lg:grid-cols-5">
        {counters.map((c) => (
          <Link
            key={c.label}
            href={c.href}
            className="group flex min-w-0 items-center gap-3 rounded-lg border bg-card/80 p-3 transition-colors hover:border-primary/40 hover:bg-card focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none"
          >
            <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-muted/70 text-muted-foreground transition-colors group-hover:bg-primary-soft group-hover:text-primary">
              <c.icon className="size-4" aria-hidden />
            </span>
            <span className="min-w-0">
              <span className="block text-xl leading-tight font-semibold tabular-nums">{c.value}</span>
              <span className="block truncate text-xs text-muted-foreground">{c.label}</span>
            </span>
          </Link>
        ))}
      </nav>

      <section id="antrean" aria-labelledby="antrean-judul" className="flex flex-col gap-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h2 id="antrean-judul" className="text-xl font-semibold">
              {t("queueTitle")}
            </h2>
            <p className="text-sm text-muted-foreground">{t("queueLead")}</p>
          </div>
          {typeFilter && (
            <Link href={scope === "campus" ? "/petugas?scope=campus" : "/petugas"} className="text-xs text-primary hover:underline">
              Reset filter
            </Link>
          )}
        </div>

        {queueList.items.length === 0 ? (
          <EmptyState icon={Sparkles}>{t("queueEmpty")}</EmptyState>
        ) : (
          <ul className="flex flex-col divide-y rounded-xl border bg-card shadow-xs">
            {queueList.items.map((item: OfficerQueueItem, index: number) => (
              <li key={item.id} className={`p-4 transition-colors hover:bg-muted/30 ${index === 0 && !typeFilter ? "bg-primary-soft/20" : ""}`}>
                <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                  <div className="min-w-0 flex-1 space-y-1.5">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge variant={index === 0 && !typeFilter ? "default" : "secondary"} className="font-semibold">
                        {t("scoreLabel", { score: Math.round(item.priority_score * 100) })}
                      </Badge>
                      <StatusBadge kind="custody" status={item.custody_status} />
                      <span className="text-xs font-medium text-muted-foreground">{item.category_name}</span>
                      {item.post_name && <span className="text-xs text-muted-foreground">• {item.post_name}</span>}
                    </div>

                    <div>
                      <Link href={item.action_url} className="text-base font-semibold hover:underline">
                        {item.title}
                      </Link>
                      <p className="text-xs text-muted-foreground">
                        <span className="font-mono">{item.found_code}</span>
                        {item.claim_code && (
                          <>
                            {" "}
                            | <span className="font-mono">{item.claim_code}</span>
                          </>
                        )}
                        {item.occurred_at && ` • ${formatRelative(item.occurred_at, now)}`}
                      </p>
                    </div>

                    {item.reason_chips.length > 0 && (
                      <div className="flex flex-wrap items-center gap-1.5 pt-1">
                        {item.reason_chips.map((chip: string) => (
                          <span key={chip} className="inline-flex items-center rounded-md bg-secondary/80 px-2 py-0.5 text-xs text-secondary-foreground">
                            {chip}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>

                  <div className="flex shrink-0 items-center">
                    <ButtonLink href={item.action_url} size="sm" variant={index === 0 && !typeFilter ? "default" : "outline"}>
                      {item.action_label}
                    </ButtonLink>
                  </div>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </>
  )
}
