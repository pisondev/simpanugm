import { FileSearch, HandHeart } from "lucide-react"
import type { Metadata } from "next"
import Link from "next/link"
import { getTranslations } from "next-intl/server"

import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { Pager } from "@/components/layout/pager"
import { FoundCard, LostCard } from "@/components/report/report-card"
import { ButtonLink } from "@/components/ui/button-link"
import { load, serverApi } from "@/lib/api/server"
import { pick } from "@/lib/query"
import { requireMe } from "@/lib/session"
import { cn } from "@/lib/utils"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("member")
  return { title: t("reportsTitle") }
}

export default async function MyReportsPage({ searchParams }: PageProps<"/laporan">) {
  const params = await searchParams
  const tab = params.tab === "temuan" ? "temuan" : "hilang"
  const cursor = pick(params.cursor)
  const [me, api, t, ts] = await Promise.all([requireMe(), serverApi(), getTranslations("member"), getTranslations("status.role")])
  const query = { mine: true, cursor, limit: 20 }
  const list =
    tab === "hilang"
      ? { kind: "lost" as const, data: await load(api.GET("/lost-reports", { params: { query } })) }
      : { kind: "found" as const, data: await load(api.GET("/found-reports", { params: { query } })) }
  const tabs = [
    { id: "hilang", label: t("tabLost") },
    { id: "temuan", label: t("tabFound") },
  ]
  return (
    <>
      <PageHeader title={t("reportsTitle")} lead={me.role === "member" ? undefined : t("staffNote", { role: ts(me.role) })} />
      <nav aria-label={t("reportsTitle")} className="mb-5 inline-flex rounded-lg bg-muted p-1">
        {tabs.map((x) => (
          <Link
            key={x.id}
            href={x.id === "hilang" ? "/laporan" : "/laporan?tab=temuan"}
            aria-current={tab === x.id ? "page" : undefined}
            className={cn(
              "inline-flex min-h-10 items-center rounded-md px-4 text-sm font-medium text-muted-foreground",
              tab === x.id && "bg-card text-foreground shadow-sm",
            )}
          >
            {x.label}
          </Link>
        ))}
      </nav>
      {list.data.items.length === 0 ? (
        <EmptyState
          icon={list.kind === "lost" ? FileSearch : HandHeart}
          action={
            <ButtonLink href={list.kind === "lost" ? "/lapor/hilang" : "/lapor/temuan"}>{list.kind === "lost" ? t("reportLost") : t("reportFound")}</ButtonLink>
          }
        >
          {list.kind === "lost" ? t("emptyReportsLost") : t("emptyReportsFound")}
        </EmptyState>
      ) : (
        <ul className="flex flex-col gap-3">
          {list.kind === "lost"
            ? list.data.items.map((r) => <LostCard key={r.id} report={r} href={`/laporan/hilang/${r.code}`} />)
            : list.data.items.map((r) => <FoundCard key={r.id} report={r} href={`/laporan/temuan/${r.code}`} />)}
        </ul>
      )}
      <Pager path="/laporan" filter={{ tab: tab === "temuan" ? "temuan" : undefined }} cursor={cursor} next={list.data.next_cursor} />
    </>
  )
}
