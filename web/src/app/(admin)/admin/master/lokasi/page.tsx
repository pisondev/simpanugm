import { MapPin } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { LocationDialog } from "@/components/admin/master/location-dialog"
import { ActiveBadge, MasterNav } from "@/components/admin/master/master-nav"
import { ToggleActive } from "@/components/admin/master/toggle-active"
import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { load, serverApi } from "@/lib/api/server"
import { pick } from "@/lib/query"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("admin.master")
  return { title: `${t("title")} | ${t("tabs.locations")}` }
}

export default async function LocationsPage({ searchParams }: PageProps<"/admin/master/lokasi">) {
  const params = await searchParams
  const q = pick(params.q)?.slice(0, 100)
  const api = await serverApi()
  const zones = await load(api.GET("/admin/zones"))
  const zone = zones.items.find((z) => z.id === pick(params.zona))?.id
  const [locations, t, tc] = await Promise.all([
    load(api.GET("/admin/locations", { params: { query: { q, zone_id: zone } } })),
    getTranslations("admin.master"),
    getTranslations("common"),
  ])
  return (
    <>
      <PageHeader title={t("title")} lead={t("lead")} actions={<LocationDialog zones={zones.items} />} />
      <MasterNav current="locations" />
      <form method="get" role="search" className="mb-5 flex flex-col gap-3 rounded-xl border bg-card p-4 sm:flex-row sm:items-end">
        <div className="flex flex-1 flex-col gap-2">
          <label htmlFor="loc-q" className="text-sm font-medium">
            {tc("search")}
          </label>
          <Input id="loc-q" name="q" type="search" defaultValue={q} placeholder={t("locations.searchPlaceholder")} />
        </div>
        <div className="flex flex-col gap-2 sm:w-56">
          <label htmlFor="loc-zone-filter" className="text-sm font-medium">
            {t("locations.zone")}
          </label>
          <NativeSelect id="loc-zone-filter" name="zona" defaultValue={zone ?? ""}>
            <option value="">{tc("all")}</option>
            {zones.items.map((z) => (
              <option key={z.id} value={z.id}>
                {z.name}
              </option>
            ))}
          </NativeSelect>
        </div>
        <Button type="submit" variant="outline">
          {tc("apply")}
        </Button>
      </form>
      {locations.items.length === 0 ? (
        <EmptyState icon={MapPin}>{t("empty")}</EmptyState>
      ) : (
        <div className="rounded-xl border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("colName")}</TableHead>
                <TableHead>{t("locations.zone")}</TableHead>
                <TableHead>{t("locations.kind")}</TableHead>
                <TableHead>{t("locations.coordinate")}</TableHead>
                <TableHead>{t("colStatus")}</TableHead>
                <TableHead>
                  <span className="sr-only">{t("colActions")}</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {locations.items.map((l) => (
                <TableRow key={l.id}>
                  <TableCell className="max-w-72 whitespace-normal">
                    <span className="flex flex-col">
                      <span className="font-medium">{l.name}</span>
                      {l.aliases.length > 0 ? <span className="text-xs text-muted-foreground">{l.aliases.join(", ")}</span> : null}
                    </span>
                  </TableCell>
                  <TableCell>{l.zone_name}</TableCell>
                  <TableCell>{t(`locations.kinds.${l.kind}`)}</TableCell>
                  <TableCell className="tabular-nums">
                    {l.lat.toFixed(5)}, {l.lng.toFixed(5)}
                  </TableCell>
                  <TableCell>
                    <ActiveBadge active={l.active} edited={Boolean(l.edited_at)} />
                  </TableCell>
                  <TableCell className="text-right whitespace-nowrap">
                    <LocationDialog location={l} zones={zones.items} />
                    <ToggleActive kind="location" id={l.id} name={l.name} active={l.active} />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </>
  )
}
