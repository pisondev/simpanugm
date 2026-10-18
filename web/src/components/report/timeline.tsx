import { useTranslations } from "next-intl"

import type { HistoryEntry } from "@/lib/api/types"
import { formatDateTime } from "@/lib/format"

type Translate = ReturnType<typeof useTranslations<"status">>

function statusLabel(t: Translate, entry: HistoryEntry, status: string): string {
  const group = entry.entity === "custody" ? "custody" : entry.entity === "lost_report" ? "lost" : "found"
  const key = `${group}.${status}`
  return t.has(key as Parameters<Translate>[0]) ? t(key as Parameters<Translate>[0]) : status
}

// FR-LOST-08 and FR-AUDIT-04, every status change with its time in WIB
export function Timeline({ items }: { items: HistoryEntry[] }) {
  const t = useTranslations("report")
  const ts = useTranslations("status")
  if (items.length === 0) {
    return <p className="text-muted-foreground">{t("timelineEmpty")}</p>
  }
  return (
    <ol className="relative flex flex-col gap-5 border-l pl-6">
      {items.map((entry, i) => (
        <li key={`${entry.created_at}-${i}`} className="relative">
          <span className="absolute top-1.5 -left-[1.95rem] size-3 rounded-full border-2 border-card bg-primary ring-2 ring-primary/20" aria-hidden />
          <p className="font-medium">
            {entry.from_status
              ? t("transitionLabel", { from: statusLabel(ts, entry, entry.from_status), to: statusLabel(ts, entry, entry.to_status) })
              : entry.entity === "custody"
                ? statusLabel(ts, entry, entry.to_status)
                : t("created")}
          </p>
          <p className="text-sm text-muted-foreground">
            <time dateTime={entry.created_at}>{formatDateTime(entry.created_at)}</time>
            {" | "}
            {entry.actor_kind === "system" ? t("bySystem") : (entry.actor_name ?? "")}
          </p>
          {entry.reason ? <p className="mt-1 text-sm">{entry.reason}</p> : null}
        </li>
      ))}
    </ol>
  )
}
