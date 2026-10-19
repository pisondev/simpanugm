import { Download, History } from "lucide-react"
import type { Metadata } from "next"

import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { Pager } from "@/components/layout/pager"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ButtonLink } from "@/components/ui/button-link"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { load, serverApi } from "@/lib/api/server"
import type { AuditLogList } from "@/lib/api/types"
import { formatDateTime } from "@/lib/format"
import { pick, queryString } from "@/lib/query"

export async function generateMetadata(): Promise<Metadata> {
  return { title: "Audit Log" }
}

export default async function AuditPage({ searchParams }: PageProps<"/admin/audit">) {
  const params = await searchParams
  const filter = {
    action: pick(params.action),
    entity_type: pick(params.entity_type),
    from: pick(params.from),
    to: pick(params.to),
  }
  const cursor = pick(params.cursor)

  const api = await serverApi()
  const rawLogs = await load(
    api.GET("/admin/audit-logs", {
      params: {
        query: {
          ...filter,
          cursor,
          limit: 50,
        },
      },
    }),
  )
  const logs: AuditLogList = typeof rawLogs === "string" ? { items: [], next_cursor: null } : rawLogs

  const csvParams = { ...filter, format: "csv" }
  const csvUrl = `/api/v1/admin/audit-logs${queryString(csvParams)}`

  return (
    <div className="mx-auto flex max-w-7xl flex-col gap-6">
      <PageHeader
        title="Audit Log"
        lead="Catatan aktivitas dan perubahan data sistem untuk kepatuhan dan keamanan (US-44)."
        actions={
          <ButtonLink href={csvUrl} variant="outline" target="_blank" download>
            <Download className="size-4" aria-hidden />
            Unduh CSV
          </ButtonLink>
        }
      />

      {/* Filter bar */}
      <form method="get" role="search" className="flex flex-col gap-3 rounded-xl border bg-card p-4 sm:flex-row sm:items-end">
        <div className="flex flex-1 flex-col gap-2">
          <label htmlFor="filter-action" className="text-sm font-medium">
            Aksi
          </label>
          <Input id="filter-action" name="action" placeholder="Contoh: auth.login, report.hide" defaultValue={filter.action} />
        </div>

        <div className="flex flex-1 flex-col gap-2">
          <label htmlFor="filter-entity" className="text-sm font-medium">
            Tipe Entitas
          </label>
          <Input id="filter-entity" name="entity_type" placeholder="Contoh: user, lost_report" defaultValue={filter.entity_type} />
        </div>

        <div className="flex flex-col gap-2 sm:w-40">
          <label htmlFor="filter-from" className="text-sm font-medium">
            Dari Tanggal
          </label>
          <Input id="filter-from" name="from" type="date" defaultValue={filter.from} />
        </div>

        <div className="flex flex-col gap-2 sm:w-40">
          <label htmlFor="filter-to" className="text-sm font-medium">
            Sampai Tanggal
          </label>
          <Input id="filter-to" name="to" type="date" defaultValue={filter.to} />
        </div>

        <Button type="submit" variant="outline">
          Terapkan
        </Button>
      </form>

      {logs.items.length === 0 ? (
        <EmptyState icon={History}>Tidak ada catatan audit yang cocok dengan filter.</EmptyState>
      ) : (
        <div className="rounded-xl border bg-card">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Waktu (WIB)</TableHead>
                <TableHead>Aktor</TableHead>
                <TableHead>Aksi</TableHead>
                <TableHead>Entitas</TableHead>
                <TableHead>IP</TableHead>
                <TableHead>Perubahan</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.items.map((log) => (
                <TableRow key={log.id}>
                  <TableCell className="font-mono text-xs whitespace-nowrap text-muted-foreground">{formatDateTime(log.occurred_at)}</TableCell>
                  <TableCell>
                    <div className="flex flex-col">
                      <span className="text-sm font-medium">{log.actor_name || "Sistem"}</span>
                      {log.actor_role ? <span className="text-xs text-muted-foreground capitalize">{log.actor_role}</span> : null}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className="font-mono text-xs">
                      {log.action}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col text-xs">
                      <span className="font-medium">{log.entity_type}</span>
                      {log.entity_id ? <span className="max-w-[120px] truncate font-mono text-[10px] text-muted-foreground">{log.entity_id}</span> : null}
                    </div>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">{log.ip || "-"}</TableCell>
                  <TableCell className="max-w-xs truncate font-mono text-xs text-muted-foreground">{log.changes ? JSON.stringify(log.changes) : "-"}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <Pager path="/admin/audit" filter={filter} cursor={cursor} next={logs.next_cursor} />
    </div>
  )
}
