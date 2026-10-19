import { ExternalLink, Layers, Network } from "lucide-react"
import type { Metadata } from "next"
import Image from "next/image"

import { ClusterRunTrigger } from "@/components/admin/cluster-run-trigger"
import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { Badge } from "@/components/ui/badge"
import { ButtonLink } from "@/components/ui/button-link"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { load, serverApi } from "@/lib/api/server"
import type { AiClustersResponse } from "@/lib/api/types"
import { formatDateTime } from "@/lib/format"
import { pick } from "@/lib/query"

export async function generateMetadata(): Promise<Metadata> {
  return { title: "Klaster Kasus (AI-11)" }
}

export default async function AiClustersPage({ searchParams }: { searchParams: Promise<Record<string, string | string[] | undefined>> }) {
  const params = await searchParams
  const runId = pick(params.run_id)

  const api = await serverApi()
  const rawClusters = await load(
    api.GET("/admin/ai/clusters", {
      params: {
        query: {
          run_id: runId,
        },
      },
    }),
  )

  const data: AiClustersResponse = rawClusters ?? { run: undefined, clusters: [] }

  return (
    <div className="mx-auto flex max-w-7xl flex-col gap-6">
      <PageHeader
        title="Pengelompokan Kasus & Deteksi Duplikat"
        lead="Kelompokkan laporan terbuka 30 hari terakhir berdasarkan kemiripan visual dan semantik menggunakan HDBSCAN (US-48, AI-11)."
        actions={<ClusterRunTrigger />}
      />

      {data.run ? (
        <Card className="bg-muted/40">
          <CardContent className="flex flex-wrap items-center justify-between gap-4 p-4 text-sm">
            <div className="flex items-center gap-2">
              <span className="font-semibold">Status Pengelompokan:</span>
              <Badge variant="outline" className="bg-background capitalize">
                {data.run.status}
              </Badge>
            </div>
            <div className="flex items-center gap-4 text-muted-foreground">
              <span>Waktu: {formatDateTime(data.run.started_at)}</span>
              <span>Total Kasus Dianalisis: {data.run.item_count}</span>
              <span className="font-medium text-foreground">Klaster Ditemukan: {data.run.cluster_count}</span>
            </div>
          </CardContent>
        </Card>
      ) : null}

      {data.clusters.length === 0 ? (
        <EmptyState icon={Network} action={<ClusterRunTrigger />}>
          Belum ada klaster yang terbentuk. Jalankan pengelompokan baru untuk mendeteksi pola kasus dan laporan berulang di kampus.
        </EmptyState>
      ) : (
        <div className="flex flex-col gap-6">
          {data.clusters.map((cluster) => (
            <Card key={cluster.cluster_id} className="overflow-hidden">
              <CardHeader className="bg-muted/30 pb-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="flex items-center gap-3">
                    <CardTitle className="flex items-center gap-2 text-lg">
                      <Layers className="size-5 text-primary" aria-hidden />
                      Klaster #{cluster.cluster_id + 1}
                    </CardTitle>
                    <Badge variant="secondary">{cluster.size} Laporan</Badge>
                  </div>
                  <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                    <span className="rounded border bg-background px-2 py-1 font-medium text-foreground">Kategori: {cluster.dominant_category}</span>
                    <span className="rounded border bg-background px-2 py-1 font-medium text-foreground">Zona: {cluster.dominant_zone}</span>
                    {cluster.earliest_at && cluster.latest_at ? (
                      <span className="rounded border bg-background px-2 py-1">
                        Rentang: {formatDateTime(cluster.earliest_at)} - {formatDateTime(cluster.latest_at)}
                      </span>
                    ) : null}
                  </div>
                </div>
              </CardHeader>

              <CardContent className="flex flex-col gap-4 p-4">
                {cluster.thumbnails.length > 0 ? (
                  <div className="flex flex-col gap-1.5">
                    <span className="text-xs font-semibold text-muted-foreground">Contoh Foto Barang:</span>
                    <div className="flex flex-wrap gap-2">
                      {cluster.thumbnails.map((thumbId) => (
                        <div key={thumbId} className="relative size-16 overflow-hidden rounded-md border bg-muted">
                          <Image src={`/api/v1/media/${thumbId}`} alt="Foto representatif klaster" fill className="object-cover" unoptimized />
                        </div>
                      ))}
                    </div>
                  </div>
                ) : null}

                <div className="rounded-md border">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Tipe</TableHead>
                        <TableHead>Kode</TableHead>
                        <TableHead>Nama Barang</TableHead>
                        <TableHead>Kategori</TableHead>
                        <TableHead>Lokasi</TableHead>
                        <TableHead>Waktu Kejadian</TableHead>
                        <TableHead className="text-right">Probabilitas</TableHead>
                        <TableHead className="text-right">Aksi</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {cluster.members.map((mem) => {
                        const isLost = mem.report_type === "lost"
                        const detailUrl = isLost ? `/laporan/hilang/${mem.code}` : `/laporan/temuan/${mem.code}`

                        return (
                          <TableRow key={`${mem.report_type}-${mem.report_id}`}>
                            <TableCell>
                              <Badge variant={isLost ? "destructive" : "default"} className="text-xs uppercase">
                                {isLost ? "Hilang" : "Temuan"}
                              </Badge>
                            </TableCell>
                            <TableCell className="font-mono text-xs font-medium">{mem.code}</TableCell>
                            <TableCell className="font-medium">{mem.item_name}</TableCell>
                            <TableCell className="text-sm text-muted-foreground">{mem.category_name}</TableCell>
                            <TableCell className="text-sm text-muted-foreground">{mem.location_name}</TableCell>
                            <TableCell className="text-xs text-muted-foreground">{formatDateTime(mem.occurred_at)}</TableCell>
                            <TableCell className="text-right font-mono text-xs">{(mem.probability * 100).toFixed(0)}%</TableCell>
                            <TableCell className="text-right">
                              <div className="flex justify-end gap-1">
                                <ButtonLink href={detailUrl} variant="ghost" size="sm" title="Lihat detail laporan">
                                  <ExternalLink className="size-3.5" aria-hidden />
                                </ButtonLink>
                                <ButtonLink href={`/petugas/papan?report=${mem.code}`} variant="outline" size="sm" title="Buka di Papan Tinjauan">
                                  Papan
                                </ButtonLink>
                              </div>
                            </TableCell>
                          </TableRow>
                        )
                      })}
                    </TableBody>
                  </Table>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
