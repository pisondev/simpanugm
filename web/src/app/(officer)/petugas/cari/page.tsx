import { Search, SearchX } from "lucide-react"
import type { Metadata } from "next"
import Link from "next/link"
import { redirect } from "next/navigation"
import { getTranslations } from "next-intl/server"

import { EmptyState } from "@/components/layout/empty-state"
import { OfficerFeatureLabel } from "@/components/dashboard/officer-feature-label"
import { PageHeader } from "@/components/layout/page-header"
import { Pager } from "@/components/layout/pager"
import { StatusBadge } from "@/components/report/status-badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { load, serverApi } from "@/lib/api/server"
import type { CustodyStatus, FoundStatus, LostStatus, SearchResult } from "@/lib/api/types"
import { formatDateTime } from "@/lib/format"
import { codePattern, reportPath } from "@/lib/paths"
import { datePattern, pick, pickOne, uuidPattern } from "@/lib/query"
import { requireStaff } from "@/lib/session"
import { staffPosts } from "@/lib/staff"
import { custodyStatus, foundStatus, lostStatus } from "@/lib/status"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("officer")
  return { title: t("searchTitle") }
}

const types = ["lost", "found"] as const
const custodies = Object.keys(custodyStatus) as CustodyStatus[]

function Status({ r }: { r: SearchResult }) {
  if (r.type === "lost" && r.status in lostStatus) {
    return <StatusBadge kind="lost" status={r.status as LostStatus} />
  }
  if (r.type === "found" && r.status in foundStatus) {
    return <StatusBadge kind="found" status={r.status as FoundStatus} />
  }
  return <span className="text-sm">{r.status}</span>
}

// US-37: keyword and trigram search across every report, an exact code opens the report
export default async function SearchPage({ searchParams }: PageProps<"/petugas/cari">) {
  const me = await requireStaff()
  const params = await searchParams
  const q = pick(params.q)?.trim().slice(0, 200)
  if (q && codePattern.test(q)) {
    redirect(reportPath(q, true))
  }
  const semantic = pick(params.semantic) === "true"
  const filter = {
    q,
    semantic: semantic ? "true" : undefined,
    type: pickOne(params.type, types),
    from: pick(params.from, datePattern),
    to: pick(params.to, datePattern),
    custody: pickOne(params.custody, custodies),
    post_id: pick(params.post_id, uuidPattern),
  }
  const cursor = pick(params.cursor)
  const [posts, t, tc, ts] = await Promise.all([staffPosts(me), getTranslations("officer"), getTranslations("common"), getTranslations("status")])
  const active = Object.values(filter).some(Boolean)
  const api = await serverApi()
  const results = active
    ? await load(
        api.GET("/search", {
          params: {
            query: {
              ...filter,
              semantic: semantic ? true : undefined,
              cursor,
              limit: 25,
            },
          },
        }),
      )
    : null
  return (
    <>
      <OfficerFeatureLabel feature="search" />
      <PageHeader title={t("searchTitle")} lead={t("searchLead")} />
      <form method="get" role="search" className="mb-6 flex flex-col gap-3 rounded-xl border bg-card p-4 shadow-xs">
        <div className="flex flex-col gap-2">
          <div className="flex gap-2">
            <label htmlFor="s-q" className="sr-only">
              {t("searchTitle")}
            </label>
            <Input id="s-q" name="q" type="search" defaultValue={q} placeholder={t("searchPlaceholder")} autoComplete="off" autoFocus />
            <Button type="submit">
              <Search aria-hidden />
              {tc("search")}
            </Button>
          </div>
          <div className="flex flex-wrap items-center gap-2 pt-0.5">
            <input
              type="checkbox"
              id="s-semantic"
              name="semantic"
              value="true"
              defaultChecked={semantic}
              aria-describedby="s-semantic-hint"
              className="h-4 w-4 cursor-pointer rounded border-input text-primary focus:ring-primary"
            />
            <label htmlFor="s-semantic" className="cursor-pointer text-sm font-medium text-foreground select-none">
              {t("semanticSearch")}
            </label>
            <span id="s-semantic-hint" className="text-xs text-muted-foreground">
              {t("semanticSearchDesc")}
            </span>
          </div>
        </div>
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-5">
          <div className="flex flex-col gap-2">
            <label htmlFor="s-type" className="text-sm font-medium">
              {t("type")}
            </label>
            <NativeSelect id="s-type" name="type" defaultValue={filter.type ?? ""}>
              <option value="">{tc("all")}</option>
              <option value="lost">{t("typeLost")}</option>
              <option value="found">{t("typeFound")}</option>
            </NativeSelect>
          </div>
          <div className="flex flex-col gap-2">
            <label htmlFor="s-custody" className="text-sm font-medium">
              {t("custody")}
            </label>
            <NativeSelect id="s-custody" name="custody" defaultValue={filter.custody ?? ""}>
              <option value="">{tc("all")}</option>
              {custodies.map((c) => (
                <option key={c} value={c}>
                  {ts(`custody.${c}`)}
                </option>
              ))}
            </NativeSelect>
          </div>
          <div className="flex flex-col gap-2">
            <label htmlFor="s-post" className="text-sm font-medium">
              {t("postFilter")}
            </label>
            <NativeSelect id="s-post" name="post_id" defaultValue={filter.post_id ?? ""}>
              <option value="">{tc("all")}</option>
              {posts.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </NativeSelect>
          </div>
          <div className="flex flex-col gap-2">
            <label htmlFor="s-from" className="text-sm font-medium">
              {t("from")}
            </label>
            <Input id="s-from" type="date" name="from" defaultValue={filter.from} />
          </div>
          <div className="flex flex-col gap-2">
            <label htmlFor="s-to" className="text-sm font-medium">
              {t("to")}
            </label>
            <Input id="s-to" type="date" name="to" defaultValue={filter.to} />
          </div>
        </div>
      </form>
      {!results ? (
        <EmptyState icon={Search}>{t("searchPrompt")}</EmptyState>
      ) : results.items.length === 0 ? (
        <EmptyState icon={SearchX}>{t("searchEmpty")}</EmptyState>
      ) : (
        <>
          <p className="mb-3 text-sm text-muted-foreground" role="status">
            {t("results", { count: results.items.length })}
          </p>
          <ul className="flex flex-col divide-y rounded-xl border bg-card">
            {results.items.map((r) => (
              <li key={`${r.type}-${r.id}`}>
                <Link href={reportPath(r.code, true)} className="flex flex-col gap-1 p-3 hover:bg-muted/50 sm:flex-row sm:items-center sm:gap-4">
                  <span className="flex min-w-0 flex-1 flex-col">
                    <span className="flex flex-wrap items-center gap-2">
                      <span className="font-mono text-xs text-muted-foreground">{r.code}</span>
                      <span className="rounded-full border px-2 py-0.5 text-xs">{r.type === "lost" ? t("typeLost") : t("typeFound")}</span>
                      <Status r={r} />
                      {r.custody_status ? <StatusBadge kind="custody" status={r.custody_status} /> : null}
                    </span>
                    <span className="truncate font-medium">{r.item_name}</span>
                    <span className="text-sm text-muted-foreground">
                      {r.category_name}
                      {r.place_name ? ` | ${r.place_name}` : ""}
                      {r.person_name ? ` | ${r.person_name}` : ""}
                    </span>
                  </span>
                  {r.relevance != null ? (
                    <span className="flex items-center gap-1 text-xs font-semibold text-primary sm:flex-col sm:items-end">
                      <span className="text-[10px] font-normal tracking-wider text-muted-foreground uppercase">{t("relevance")}</span>
                      <span>{Math.round(r.relevance * 100)}%</span>
                    </span>
                  ) : null}
                  <span className="text-sm text-muted-foreground">{formatDateTime(r.occurred_at)}</span>
                </Link>
              </li>
            ))}
          </ul>
          <Pager path="/petugas/cari" filter={filter} cursor={cursor} next={results.next_cursor} />
        </>
      )}
    </>
  )
}
