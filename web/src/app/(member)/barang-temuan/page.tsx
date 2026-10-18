import { PackageSearch, Warehouse } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { Pager } from "@/components/layout/pager"
import { StatusBadge } from "@/components/report/status-badge"
import { Button } from "@/components/ui/button"
import { ButtonLink } from "@/components/ui/button-link"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { load, serverApi } from "@/lib/api/server"
import { formatDate } from "@/lib/format"
import { zonesAndCategories } from "@/lib/masterdata"
import { datePattern, pick, uuidPattern } from "@/lib/query"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("publicFound")
  return { title: t("title") }
}

// US-17: category, item name, zone, date and post only, never a photo, description or finder
export default async function PublicFoundPage({ searchParams }: PageProps<"/barang-temuan">) {
  const params = await searchParams
  const filter = {
    category_id: pick(params.category_id, uuidPattern),
    zone_id: pick(params.zone_id, uuidPattern),
    from: pick(params.from, datePattern),
    to: pick(params.to, datePattern),
  }
  const cursor = pick(params.cursor)
  const [api, master, t, tc] = await Promise.all([serverApi(), zonesAndCategories(), getTranslations("publicFound"), getTranslations("common")])
  const list = await load(api.GET("/found-reports/public", { params: { query: { ...filter, cursor, limit: 30 } } }))
  return (
    <>
      <PageHeader title={t("title")} lead={t("lead")} />
      <form method="get" className="mb-6 grid gap-3 rounded-xl border bg-card p-4 sm:grid-cols-2 lg:grid-cols-5 lg:items-end">
        <div className="flex flex-col gap-2">
          <label htmlFor="f-category" className="text-sm font-medium">
            {t("category")}
          </label>
          <NativeSelect id="f-category" name="category_id" defaultValue={filter.category_id ?? ""}>
            <option value="">{tc("all")}</option>
            {master.categories.map((parent) => (
              <optgroup key={parent.id} label={parent.name}>
                <option value={parent.id}>{parent.name}</option>
                {(parent.children ?? []).map((leaf) => (
                  <option key={leaf.id} value={leaf.id}>
                    {leaf.name}
                  </option>
                ))}
              </optgroup>
            ))}
          </NativeSelect>
        </div>
        <div className="flex flex-col gap-2">
          <label htmlFor="f-zone" className="text-sm font-medium">
            {t("zone")}
          </label>
          <NativeSelect id="f-zone" name="zone_id" defaultValue={filter.zone_id ?? ""}>
            <option value="">{tc("all")}</option>
            {master.zones.map((z) => (
              <option key={z.id} value={z.id}>
                {z.name}
              </option>
            ))}
          </NativeSelect>
        </div>
        <div className="flex flex-col gap-2">
          <label htmlFor="f-from" className="text-sm font-medium">
            {t("from")}
          </label>
          <Input id="f-from" type="date" name="from" defaultValue={filter.from} />
        </div>
        <div className="flex flex-col gap-2">
          <label htmlFor="f-to" className="text-sm font-medium">
            {t("to")}
          </label>
          <Input id="f-to" type="date" name="to" defaultValue={filter.to} />
        </div>
        <div className="flex gap-2">
          <Button type="submit" className="flex-1">
            {t("filter")}
          </Button>
          <ButtonLink href="/barang-temuan" variant="ghost">
            {tc("reset")}
          </ButtonLink>
        </div>
      </form>
      {list.items.length === 0 ? (
        <EmptyState
          icon={PackageSearch}
          action={
            <ButtonLink href="/lapor/hilang" variant="outline">
              {t("reportLost")}
            </ButtonLink>
          }
        >
          {t("empty")}
        </EmptyState>
      ) : (
        <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {list.items.map((item) => (
            <li key={item.id} className="flex flex-col gap-2 rounded-xl border bg-card p-4 shadow-sm">
              <div className="flex items-start justify-between gap-2">
                <p className="font-medium">{item.item_name}</p>
                <StatusBadge kind="custody" status={item.custody_status} />
              </div>
              <p className="text-sm text-muted-foreground">
                {item.category.name} | {item.zone.name}
              </p>
              <p className="text-sm">{t("foundDate", { date: formatDate(`${item.found_date}T12:00:00+07:00`) })}</p>
              <p className="flex items-center gap-2 text-sm text-muted-foreground">
                <Warehouse className="size-4" aria-hidden />
                {item.holding_post ? t("atPost", { post: item.holding_post.name }) : t("withFinder")}
              </p>
            </li>
          ))}
        </ul>
      )}
      <Pager path="/barang-temuan" filter={filter} cursor={cursor} next={list.next_cursor} />
    </>
  )
}
