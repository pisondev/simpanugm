"use client"

import { useTranslations } from "next-intl"
import { useSyncExternalStore } from "react"

import { formatRanges, isOpenNow, type OpeningHours, rangesFor, today } from "@/lib/hours"
import { cn } from "@/lib/utils"

// the open state depends on the visitor's clock, so it renders after hydration only
const noop = () => () => {}

export function OpenNow({ hours, className }: { hours: OpeningHours; className?: string }) {
  const t = useTranslations("form.custody")
  const open = useSyncExternalStore(
    noop,
    () => isOpenNow(hours),
    () => null,
  )
  if (open === null) {
    return null
  }
  return (
    <span className={cn("inline-flex items-center gap-1.5 text-sm font-medium", open ? "text-success-text" : "text-muted-foreground", className)}>
      <span className={cn("size-2 rounded-full", open ? "bg-success" : "bg-neutral")} aria-hidden />
      {open ? t("openNow") : t("closedNow")}
    </span>
  )
}

export function TodayHours({ hours }: { hours: OpeningHours }) {
  const t = useTranslations("form.custody")
  const day = useSyncExternalStore(
    noop,
    () => today(),
    () => null,
  )
  if (day === null) {
    return null
  }
  const ranges = rangesFor(hours, day)
  return <span className="text-sm text-muted-foreground">{ranges.length > 0 ? t("hoursToday", { hours: formatRanges(ranges) }) : t("closedToday")}</span>
}
