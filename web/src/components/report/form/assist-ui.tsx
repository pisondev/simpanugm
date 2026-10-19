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
      <div role="status" className="ai-panel flex items-start gap-3 p-4 text-sm">
        <LoaderCircle className="mt-0.5 size-5 shrink-0 animate-spin text-primary" aria-hidden />
        <div>
          <p className="font-medium">{t("loading")}</p>
          <p className="mt-1 text-muted-foreground">{t("loadingHint")}</p>
        </div>
      </div>
    )
  }
  if (assist.status === "unavailable") {
    return (
      <div role="status" className="surface p-4 text-sm">
        <p className="font-medium">{t("unavailable")}</p>
        <p className="mt-1 text-muted-foreground">{t("unavailableHint")}</p>
        <Button type="button" variant="outline" className="mt-3" onClick={assist.retry}>
          {t("retry")}
        </Button>
      </div>
    )
  }
  if (assist.status === "ready") {
    return (
      <div role="status" className="ai-panel motion-fade flex items-start gap-3 p-4 text-sm">
        <Sparkles className="mt-0.5 size-5 shrink-0 text-primary" aria-hidden />
        <div>
          <p className="font-medium">{t("ready")}</p>
          <p className="mt-1 text-muted-foreground">{t("readyHint")}</p>
        </div>
      </div>
    )
  }
  return null
}

// the small label of AI-03 on a field that still holds the suggestion
export function SuggestedTag({ show }: { show: boolean }) {
  const t = useTranslations("form.assist")
  if (!show) return null
  return (
    <span className="motion-fade inline-flex w-fit items-center gap-1 rounded-full border border-primary/20 bg-primary-soft px-2 py-0.5 text-xs font-medium text-primary">
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
