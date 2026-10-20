import { ImageOff } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"

import { mediaUrl } from "@/lib/api/client"
import type { FoundReportSummary, LostReportSummary } from "@/lib/api/types"
import { formatDateTime, formatRelative } from "@/lib/format"

import { StatusBadge } from "./status-badge"

function Thumb({ id, alt }: { id?: string | null; alt: string }) {
  if (!id) {
    return (
      <div className="flex size-16 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
        <ImageOff className="size-5" aria-hidden />
      </div>
    )
  }
  return <img src={mediaUrl(id, "thumb")} alt={alt} className="size-16 shrink-0 rounded-lg bg-muted object-cover" loading="lazy" />
}

export function LostCard({ report, href }: { report: LostReportSummary; href: string }) {
  const t = useTranslations("member")
  const tc = useTranslations("common")
  return (
    <li>
      <Link href={href} className="surface motion-lift motion-press flex gap-3 p-3 focus-visible:ring-3 focus-visible:ring-ring/50">
        <Thumb id={report.thumb_media_id} alt={tc("photoAlt", { kind: tc("photoKindLost"), name: report.item_name })} />
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-xs text-muted-foreground">{report.code}</span>
            <StatusBadge kind="lost" status={report.status} />
          </div>
          <p className="truncate font-medium">{report.item_name}</p>
          <p className="text-sm text-muted-foreground">
            {report.category_name} | {t("candidates", { count: report.candidate_count })}
          </p>
          <p className="text-xs text-muted-foreground">
            {report.reporter_name ? `${report.reporter_name} | ` : ""}
            {t("updated", { when: formatRelative(report.updated_at) })}
          </p>
        </div>
      </Link>
    </li>
  )
}

export function FoundCard({ report, href }: { report: FoundReportSummary; href: string }) {
  const t = useTranslations("member")
  const tc = useTranslations("common")
  return (
    <li>
      <Link href={href} className="surface motion-lift motion-press flex gap-3 p-3 focus-visible:ring-3 focus-visible:ring-ring/50">
        <Thumb id={report.thumb_media_id} alt={tc("photoAlt", { kind: tc("photoKindFound"), name: report.item_name })} />
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-xs text-muted-foreground">{report.code}</span>
            <StatusBadge kind="found" status={report.status} />
            <StatusBadge kind="custody" status={report.custody_status} />
          </div>
          <p className="truncate font-medium">{report.item_name}</p>
          <p className="text-sm text-muted-foreground">
            {report.category_name} | {report.location_name}
          </p>
          <p className="text-xs text-muted-foreground">
            {report.finder_name ? `${report.finder_name} | ` : ""}
            {t("foundAt", { when: formatDateTime(report.found_at) })}
          </p>
        </div>
      </Link>
    </li>
  )
}
