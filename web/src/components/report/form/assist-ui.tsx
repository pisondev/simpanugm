"use client"

import { LoaderCircle, Sparkles, WandSparkles } from "lucide-react"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"

import type { Assist } from "./use-assist"

// FR-ASSIST-04: the form carries on, this note is all a failed assistant leaves behind
export function AssistNote({ assist }: { assist: Assist }) {
  const t = useTranslations("form.assist")
  if (assist.status === "loading") {
    return (
      <p role="status" className="flex items-center gap-2 rounded-lg border border-info/30 bg-primary-soft/50 p-3 text-sm">
        <LoaderCircle className="size-4 animate-spin text-info" aria-hidden />
        {t("loading")}
      </p>
    )
  }
  if (assist.status === "unavailable") {
    return (
      <p role="status" className="rounded-lg border bg-muted/50 p-3 text-sm text-muted-foreground">
        {t("unavailable")}
      </p>
    )
  }
  return null
}

// the small label of AI-03 on a field that still holds the suggestion
export function SuggestedTag({ show }: { show: boolean }) {
  const t = useTranslations("form.assist")
  if (!show) return null
  return (
    <span className="inline-flex w-fit items-center gap-1 rounded-full border border-info/30 bg-primary-soft px-2 py-0.5 text-xs font-medium text-info">
      <Sparkles className="size-3" aria-hidden />
      {t("tag")}
    </span>
  )
}

export function CategoryAlternatives({ assist, onPick }: { assist: Assist; onPick: (id: string) => void }) {
  const t = useTranslations("form.assist")
  if (assist.alternatives.length === 0) return null
  return (
    <div className="flex flex-wrap items-center gap-2" role="group" aria-label={t("alternatives")}>
      <span className="text-sm text-muted-foreground">{t("alternatives")}</span>
      {assist.alternatives.map((a) => (
        <Button key={a.category.id} type="button" variant="outline" size="sm" onClick={() => onPick(a.category.id)}>
          {a.category.name}
        </Button>
      ))}
    </div>
  )
}

export function RedraftButton({ assist }: { assist: Assist }) {
  const t = useTranslations("form.assist")
  if (assist.status !== "ready") return null
  return (
    <Button type="button" variant="ghost" size="sm" className="w-fit" onClick={() => void assist.redraft()}>
      <WandSparkles aria-hidden />
      {t("redraft")}
    </Button>
  )
}
