import { BarChart3, CheckCircle2, Clock, FileText, Layers, ShieldAlert, TrendingUp } from "lucide-react"
import type { Metadata } from "next"

import { ExportCsvForm } from "@/components/admin/export-csv-form"
import { PageHeader } from "@/components/layout/page-header"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { load, serverApi } from "@/lib/api/server"
import { pick } from "@/lib/query"

export async function generateMetadata(): Promise<Metadata> {
  return { title: "Tren Operasional & Analitik" }
}

const dayNames = ["Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu", "Minggu"]

export default async function AnalyticsTrendsPage({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  const params = await searchParams
  const from = pick(params.from)
  const to = pick(params.to)
  const categoryId = pick(params.category_id)
  const zoneId = pick(params.zone_id)

  const api = await serverApi()
  const [summary, trends] = await Promise.all([
    load(api.GET("/analytics/summary", {})),
    load(
      api.GET("/analytics/trends", {
        params: {
          query: {
            from,
            to,
            category_id: categoryId,
            zone_id: zoneId,
          },
        },
      }),
    ),
  ])

  return (
    <div className="mx-auto flex max-w-7xl flex-col gap-6">
      <PageHeader
        title="Tren Operasional & Analitik"
        lead="Statistik perkembangan laporan kehilangan dan temuan, pola waktu kejadian, dan ekspor laporan bulanan (US-47)."
        actions={<ExportCsvForm />}
      />

      {/* KPI Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              Total Laporan Kehilangan
              <FileText className="size-4 text-destructive" aria-hidden />
            </CardDescription>
            <CardTitle className="text-2xl">{summary.total_lost}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xs text-muted-foreground">{summary.period_lost} laporan dalam 30 hari terakhir</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              Total Laporan Temuan
              <Layers className="size-4 text-primary" aria-hidden />
            </CardDescription>
            <CardTitle className="text-2xl">{summary.total_found}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xs text-muted-foreground">{summary.period_found} laporan dalam 30 hari terakhir</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              Tingkat Pengembalian
              <CheckCircle2 className="size-4 text-success-text" aria-hidden />
            </CardDescription>
            <CardTitle className="text-2xl">{(summary.return_rate * 100).toFixed(1)}%</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xs text-muted-foreground">{summary.total_returned} barang berhasil dikembalikan</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              Klaim Aktif Berjalan
              <ShieldAlert className="size-4 text-amber-600" aria-hidden />
            </CardDescription>
            <CardTitle className="text-2xl">{summary.active_claims}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xs text-muted-foreground">Sedang ditinjau atau menunggu serah terima</p>
          </CardContent>
        </Card>
      </div>

      {/* 12-Week Progression Table */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp className="size-5 text-primary" aria-hidden />
            Tren Mingguan (12 Minggu Terakhir)
          </CardTitle>
          <CardDescription>Jumlah laporan kehilangan, temuan, dan barang yang berhasil dikembalikan per minggu.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Awal Minggu</TableHead>
                  <TableHead className="text-right">Kehilangan</TableHead>
                  <TableHead className="text-right">Temuan</TableHead>
                  <TableHead className="text-right">Dikembalikan</TableHead>
                  <TableHead className="text-right">Tingkat Kembali</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {trends.weekly.map((w) => {
                  const rate = w.found_count > 0 ? ((w.returned_count / w.found_count) * 100).toFixed(0) + "%" : "-"
                  return (
                    <TableRow key={w.week_start}>
                      <TableCell className="font-mono text-xs">{w.week_start}</TableCell>
                      <TableCell className="text-right font-medium text-destructive">{w.lost_count}</TableCell>
                      <TableCell className="text-right font-medium text-primary">{w.found_count}</TableCell>
                      <TableCell className="text-right font-semibold text-success-text">{w.returned_count}</TableCell>
                      <TableCell className="text-right text-xs text-muted-foreground">{rate}</TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Category Breakdown */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BarChart3 className="size-5 text-primary" aria-hidden />
              Kategori Terbanyak
            </CardTitle>
            <CardDescription>Distribusi barang hilang dan barang temuan berdasarkan kategori.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Kategori</TableHead>
                    <TableHead className="text-right">Hilang</TableHead>
                    <TableHead className="text-right">Temuan</TableHead>
                    <TableHead className="text-right">Total</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {trends.categories.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={4} className="py-4 text-center text-muted-foreground">
                        Belum ada data kategori.
                      </TableCell>
                    </TableRow>
                  ) : (
                    trends.categories.map((c) => (
                      <TableRow key={c.category_id}>
                        <TableCell className="font-medium">{c.category_name}</TableCell>
                        <TableCell className="text-right text-destructive">{c.lost_count}</TableCell>
                        <TableCell className="text-right text-primary">{c.found_count}</TableCell>
                        <TableCell className="text-right font-semibold">{c.lost_count + c.found_count}</TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>

        {/* Time Pattern Overview (Day of Week & Hour) */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Clock className="size-5 text-primary" aria-hidden />
              Pola Waktu Kejadian (Hour by Day)
            </CardTitle>
            <CardDescription>
              Frekuensi pelaporan barang per jam dan hari. Sel dengan kurang dari 3 laporan disamarkan menjadi &lt; 3 (BR-PRV-05).
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col gap-3">
              <div className="grid grid-cols-8 gap-1 text-center font-mono text-[11px]">
                <div className="font-semibold text-muted-foreground">Jam</div>
                {dayNames.map((d) => (
                  <div key={d} className="font-semibold text-muted-foreground">
                    {d.slice(0, 3)}
                  </div>
                ))}
              </div>

              {/* Sample representative peak hours for clean visual representation */}
              {[8, 10, 12, 14, 16, 18, 20].map((hour) => (
                <div key={hour} className="grid grid-cols-8 items-center gap-1 text-center text-xs">
                  <div className="font-mono text-[10px] text-muted-foreground">{String(hour).padStart(2, "0")}:00</div>
                  {[1, 2, 3, 4, 5, 6, 7].map((wd) => {
                    const cell = trends.hourly.find((h) => h.weekday === wd && h.hour === hour)
                    const count = cell?.count ?? 0
                    const display = cell?.display_count ?? "0"

                    let bgClass = "bg-muted/40 text-muted-foreground"
                    if (count >= 5) {
                      bgClass = "bg-primary text-primary-foreground font-semibold"
                    } else if (count >= 3) {
                      bgClass = "bg-primary/20 text-foreground font-medium"
                    } else if (count > 0) {
                      bgClass = "bg-primary/10 text-muted-foreground"
                    }

                    return (
                      <div
                        key={wd}
                        className={`flex h-7 items-center justify-center rounded text-[10px] ${bgClass}`}
                        title={`${dayNames[wd - 1]} pukul ${hour}:00, ${display} laporan`}
                      >
                        {display}
                      </div>
                    )
                  })}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
