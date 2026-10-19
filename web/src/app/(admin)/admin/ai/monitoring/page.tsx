import { AlertTriangle, CheckCircle2, Cpu, HelpCircle, Layers, Sparkles, TrendingUp } from "lucide-react"
import type { Metadata } from "next"

import { ModelVersionActivator } from "@/components/admin/model-version-activator"
import { PageHeader } from "@/components/layout/page-header"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { load, serverApi } from "@/lib/api/server"
import type { ModelVersionItem } from "@/lib/api/types"
import { formatDateTime } from "@/lib/format"
import { pick } from "@/lib/query"

export async function generateMetadata(): Promise<Metadata> {
  return { title: "Monitoring AI & Mesin Pencocokan" }
}

export default async function AiMonitoringPage({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  const params = await searchParams
  const from = pick(params.from)
  const to = pick(params.to)
  const modelVersionId = pick(params.model_version_id)

  const api = await serverApi()
  const mon = await load(
    api.GET("/admin/ai/monitoring", {
      params: {
        query: {
          from,
          to,
          model_version_id: modelVersionId,
        },
      },
    }),
  )

  const rawVersions = await load(api.GET("/admin/model-versions", {}))
  const allVersions: ModelVersionItem[] = rawVersions?.items ?? []

  return (
    <div className="mx-auto flex max-w-7xl flex-col gap-6">
      <PageHeader
        title="Monitoring AI & Mesin Pencocokan"
        lead="Pantau akurasi, distribusi skor, drift populasi, dan aktifkan atau rollback versi model (US-49, AI-12)."
        actions={<ModelVersionActivator allVersions={allVersions} />}
      />

      {mon.psi_warning ? (
        <div className="flex items-start gap-3 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-destructive">
          <AlertTriangle className="mt-0.5 size-5 shrink-0" aria-hidden />
          <div className="flex flex-col gap-1">
            <p className="font-semibold">Peringatan Drift Skor Terdeteksi (PSI: {mon.score_psi.toFixed(3)})</p>
            <p className="text-sm">
              Distribusi skor kecocokan bergeser melebihi ambang batas kestabilan populasi (0.2). Evaluasi model atau periksa apakah terjadi perubahan pola
              barang yang dilaporkan pengguna.
            </p>
          </div>
        </div>
      ) : null}

      {/* Grid: Overview Stats */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              Recall@1 Terkonfirmasi
              <CheckCircle2 className="size-4 text-success-text" aria-hidden />
            </CardDescription>
            <CardTitle className="text-2xl">{(mon.confirmed_ranks.recall_at_1 * 100).toFixed(1)}%</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xs text-muted-foreground">
              {mon.confirmed_ranks.rank_1_count} dari {mon.confirmed_ranks.confirmed_total} klaim terkonfirmasi
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              Recall@5 Terkonfirmasi
              <Sparkles className="size-4 text-sky-600" aria-hidden />
            </CardDescription>
            <CardTitle className="text-2xl">{(mon.confirmed_ranks.recall_at_5 * 100).toFixed(1)}%</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xs text-muted-foreground">{mon.confirmed_ranks.rank_5_count} kecocokan di 5 kandidat teratas</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              Recall@10 Terkonfirmasi
              <Layers className="size-4 text-indigo-600" aria-hidden />
            </CardDescription>
            <CardTitle className="text-2xl">{(mon.confirmed_ranks.recall_at_10 * 100).toFixed(1)}%</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xs text-muted-foreground">{mon.confirmed_ranks.rank_10_count} kecocokan di 10 kandidat teratas</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardDescription className="flex items-center justify-between">
              Score PSI (Drift 4 Minggu)
              <TrendingUp className="size-4 text-amber-600" aria-hidden />
            </CardDescription>
            <CardTitle className="text-2xl">{mon.score_psi.toFixed(3)}</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-xs text-muted-foreground">
              {mon.psi_warning ? (
                <span className="font-semibold text-destructive">Drift terdeteksi (&gt; 0.2)</span>
              ) : (
                <span className="font-medium text-success-text">Distribusi stabil</span>
              )}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Active Model Versions */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Cpu className="size-5 text-primary" aria-hidden />
            Versi Model Aktif
          </CardTitle>
          <CardDescription>Model yang sedang aktif melayani inferensi embedding dan pencocokan kandidat laporan.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Komponen</TableHead>
                  <TableHead>Nama Versi</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Diaktifkan Pada</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {mon.active_versions.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="py-4 text-center text-muted-foreground">
                      Belum ada versi model yang aktif.
                    </TableCell>
                  </TableRow>
                ) : (
                  mon.active_versions.map((v) => (
                    <TableRow key={v.id}>
                      <TableCell className="font-mono text-xs">{v.component}</TableCell>
                      <TableCell className="font-medium">{v.name}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className="border-emerald-200 bg-emerald-50 text-emerald-700">
                          {v.status}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">{v.activated_at ? formatDateTime(v.activated_at) : "Bawaan"}</TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* Top 1 Score Shape Histogram & Dismiss Rates */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Distribusi Skor Kecocokan Top 1</CardTitle>
            <CardDescription>Bentuk histogram skor tertinggi per laporan baru untuk mendeteksi anomali skoring.</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-4">
            {mon.top1_histogram.map((b) => {
              const maxCount = Math.max(...mon.top1_histogram.map((h) => h.count), 1)
              const pct = Math.round((b.count / maxCount) * 100)
              return (
                <div key={b.bucket} className="flex flex-col gap-1">
                  <div className="flex justify-between text-xs font-medium">
                    <span>{b.bucket}</span>
                    <span className="text-muted-foreground">{b.count} laporan</span>
                  </div>
                  <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
                    <div className="h-full bg-primary transition-all" style={{ width: `${pct}%` }} />
                  </div>
                </div>
              )
            })}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Tingkat Pengabaian (Dismiss Rate) per Tier</CardTitle>
            <CardDescription>Persentase kandidat yang diabaikan oleh pemilik atau ditolak petugas per tier kepercayaan.</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Tier</TableHead>
                    <TableHead className="text-right">Total Match</TableHead>
                    <TableHead className="text-right">Diabaikan Pemilik</TableHead>
                    <TableHead className="text-right">Ditolak Petugas</TableHead>
                    <TableHead className="text-right">Dismiss Rate</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {mon.dismiss_rates.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="py-4 text-center text-muted-foreground">
                        Belum ada data interaksi match.
                      </TableCell>
                    </TableRow>
                  ) : (
                    mon.dismiss_rates.map((d) => (
                      <TableRow key={d.tier}>
                        <TableCell className="text-xs font-semibold uppercase">{d.tier}</TableCell>
                        <TableCell className="text-right">{d.total}</TableCell>
                        <TableCell className="text-right">{d.owner_dismissed}</TableCell>
                        <TableCell className="text-right">{d.officer_dismissed}</TableCell>
                        <TableCell className="text-right font-medium">{(d.dismiss_rate * 100).toFixed(1)}%</TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Assist Outcomes (AI-03) */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <HelpCircle className="size-5 text-muted-foreground" aria-hidden />
            Hasil Saran Asisten Laporan (AI-03)
          </CardTitle>
          <CardDescription>Tingkat penerimaan saran otomatis saat pelapor mengisi formulir kehilangan atau penemuan barang.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Bidang Isian</TableHead>
                  <TableHead>Hasil Pengguna</TableHead>
                  <TableHead className="text-right">Jumlah Interaksi</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {mon.assist_outcomes.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={3} className="py-4 text-center text-muted-foreground">
                      Belum ada data interaksi asisten formulir.
                    </TableCell>
                  </TableRow>
                ) : (
                  mon.assist_outcomes.map((a, idx) => (
                    <TableRow key={`${a.field}-${a.outcome}-${idx}`}>
                      <TableCell className="font-medium capitalize">{a.field}</TableCell>
                      <TableCell>
                        <Badge variant="outline" className="text-xs capitalize">
                          {a.outcome}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right font-semibold">{a.count}</TableCell>
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
