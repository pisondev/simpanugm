import { useTranslations } from "next-intl"

import { daysUntil, formatDate } from "@/lib/format"
import { cn } from "@/lib/utils"

export function DueLabel({ iso, now }: { iso: string; now: Date }) {
  const t = useTranslations("officer")
  const days = daysUntil(iso, now)
  return (
    <span className={cn("flex flex-col", days <= 7 && "font-medium text-warning", days < 0 && "text-destructive")}>
      <span>{formatDate(iso)}</span>
      <span className="text-xs">{days < 0 ? t("overdue", { count: -days }) : t("dueIn", { count: days })}</span>
    </span>
  )
}
