import { AlertTriangle, Filter, MapPin } from "lucide-react"
import type { Metadata } from "next"

import { CampusHotspotMap } from "@/components/admin/campus-hotspot-map"
import { PageHeader } from "@/components/layout/page-header"
import { ButtonLink } from "@/components/ui/button-link"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { load, serverApi } from "@/lib/api/server"
import type { HotspotsResponse } from "@/lib/api/types"
import { pick, queryString } from "@/lib/query"

export async function generateMetadata(): Promise<Metadata> {
  return { title: "Peta Rawan Kehilangan & Temuan (AI-10)" }
}

export default async function CampusHotspotsPage({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  const params = await searchParams
  const days = pick(params.days) ?? "30"
  const type = (pick(params.type) as "lost" | "found" | "all") ?? "all"
  const zoneId = pick(params.zone_id)

  let from: string | undefined
  if (days !== "all") {
    const d = parseInt(days, 10)
    if (!isNaN(d)) {
      const date = new Date()
      date.setDate(date.getDate() - d)
      from = date.toISOString()
    }
  }

  const api = await serverApi()
  const rawData = await load(
    api.GET("/analytics/hotspots", {
      params: {
        query: {
          from,
          type,
          zone_id: zoneId,
        },
      },
    }),
  )

  const hotspots: HotspotsResponse = rawData ?? { locations: [], insights: [] }

  // Sort locations by lost count descending, then total count
  const sortedLocations = [...hotspots.locations].sort((a, b) => {
    if (type === "lost") return b.lost_count - a.lost_count
    if (type === "found") return b.found_count - a.found_count
    return b.total_count - a.total_count
  })

  return (
    <div className="mx-auto flex max-w-7xl flex-col gap-6">
      <PageHeader
        title="Peta Rawan Kehilangan & Temuan (AI-10)"
        lead="Visualisasi titik rawan kehilangan dan temuan barang di lingkungan kampus UGM untuk evaluasi keamanan (US-46, AI-10)."
      />

      {/* Filter Controls Bar */}
      <div className="flex flex-wrap items-center justify-between gap-4 rounded-lg border bg-card p-3 shadow-sm">
        <div className="flex flex-wrap items-center gap-2">
          <span className="mr-1 flex items-center gap-1.5 text-xs font-semibold text-muted-foreground">
            <Filter className="size-3.5" aria-hidden />
            Rentang Waktu:
          </span>
          {[
            { label: "7 Hari", value: "7" },
            { label: "30 Hari", value: "30" },
            { label: "90 Hari", value: "90" },
            { label: "Semua", value: "all" },
          ].map((d) => {
            const active = days === d.value
            return (
              <ButtonLink
                key={d.value}
                href={`/admin/peta-rawan${queryString({ ...params, days: d.value })}`}
                variant={active ? "default" : "outline"}
                size="sm"
                className="h-8 text-xs"
              >
                {d.label}
              </ButtonLink>
            )
          })}
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <span className="mr-1 text-xs font-semibold text-muted-foreground">Tipe Laporan:</span>
          {[
            { label: "Semua", value: "all" },
            { label: "Kehilangan", value: "lost" },
            { label: "Temuan", value: "found" },
          ].map((t) => {
            const active = type === t.value
            return (
              <ButtonLink
                key={t.value}
                href={`/admin/peta-rawan${queryString({ ...params, type: t.value })}`}
                variant={active ? "secondary" : "ghost"}
                size="sm"
                className="h-8 text-xs font-medium"
              >
                {t.label}
              </ButtonLink>
            )
          })}
        </div>
      </div>

      {/* Surge Insights Banner */}
      {hotspots.insights.length > 0 ? (
        <div className="flex items-start gap-3 rounded-lg border border-amber-500/50 bg-amber-500/10 p-4 text-amber-800 dark:text-amber-200">
          <AlertTriangle className="mt-0.5 size-5 shrink-0" aria-hidden />
          <div className="flex flex-col gap-1">
            <p className="text-sm font-semibold">Lonjakan Kasus Terdeteksi (Analisis Statistik AI-10)</p>
            <ul className="list-disc space-y-1 pl-5 text-xs">
              {hotspots.insights.map((ins, i) => (
                <li key={i}>{ins}</li>
              ))}
            </ul>
          </div>
        </div>
      ) : null}

      {/* Map Component */}
      <CampusHotspotMap locations={sortedLocations} selectedType={type} />

      {/* Ranked Locations Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <MapPin className="size-5 text-primary" aria-hidden />
            Peringkat Titik Lokasi Kampus
          </CardTitle>
          <CardDescription>
            Urutan lokasi berdasarkan frekuensi pelaporan. Sel dengan kurang dari 3 laporan disamarkan menjadi &lt; 3 untuk menjaga privasi pelapor (BR-PRV-05).
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div tabIndex={0} className="overflow-x-auto rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-12 text-center">#</TableHead>
                  <TableHead>Nama Lokasi</TableHead>
                  <TableHead>Zona Kampus</TableHead>
                  <TableHead className="text-right">Kehilangan</TableHead>
                  <TableHead className="text-right">Temuan</TableHead>
                  <TableHead className="text-right font-semibold">Total</TableHead>
                  <TableHead className="text-right">Tingkat Kembali</TableHead>
                  <TableHead className="text-right">Median Kembali</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sortedLocations.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={8} className="py-6 text-center text-muted-foreground">
                      Tidak ada laporan pada periode ini.
                    </TableCell>
                  </TableRow>
                ) : (
                  sortedLocations.map((loc, idx) => (
                    <TableRow key={loc.location_id}>
                      <TableCell className="text-center font-mono text-xs text-muted-foreground">{idx + 1}</TableCell>
                      <TableCell className="font-medium">{loc.location_name}</TableCell>
                      <TableCell className="text-xs text-muted-foreground">{loc.zone_name}</TableCell>
                      <TableCell className="text-right text-destructive">{loc.display_lost}</TableCell>
                      <TableCell className="text-right text-primary">{loc.display_found}</TableCell>
                      <TableCell className="text-right font-semibold">{loc.display_total}</TableCell>
                      <TableCell className="text-right font-medium text-success-text">{(loc.return_rate * 100).toFixed(0)}%</TableCell>
                      <TableCell className="text-right text-xs text-muted-foreground">
                        {loc.median_return_hours ? `${loc.median_return_hours} jam` : "-"}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
