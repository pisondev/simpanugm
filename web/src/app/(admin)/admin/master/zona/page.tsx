import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { ActiveBadge, MasterNav } from "@/components/admin/master/master-nav"
import { ToggleActive } from "@/components/admin/master/toggle-active"
import { ZoneDialog } from "@/components/admin/master/zone-dialog"
import { PageHeader } from "@/components/layout/page-header"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { load, serverApi } from "@/lib/api/server"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("admin.master")
  return { title: `${t("title")} | ${t("tabs.zones")}` }
}

export default async function ZonesPage() {
  const api = await serverApi()
  const [zones, t] = await Promise.all([load(api.GET("/admin/zones")), getTranslations("admin.master")])
  return (
    <>
      <PageHeader title={t("title")} lead={t("lead")} actions={<ZoneDialog />} />
      <MasterNav current="zones" />
      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("colName")}</TableHead>
              <TableHead>{t("colSlug")}</TableHead>
              <TableHead>{t("locations.coordinate")}</TableHead>
              <TableHead>{t("colStatus")}</TableHead>
              <TableHead>
                <span className="sr-only">{t("colActions")}</span>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {zones.items.map((z) => (
              <TableRow key={z.id}>
                <TableCell className="font-medium">{z.name}</TableCell>
                <TableCell className="text-muted-foreground">{z.slug}</TableCell>
                <TableCell className="tabular-nums">
                  {z.centroid_lat.toFixed(5)}, {z.centroid_lng.toFixed(5)}
                </TableCell>
                <TableCell>
                  <ActiveBadge active={z.active} edited={Boolean(z.edited_at)} />
                </TableCell>
                <TableCell className="text-right whitespace-nowrap">
                  <ZoneDialog zone={z} />
                  <ToggleActive kind="zone" id={z.id} name={z.name} active={z.active} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </>
  )
}
