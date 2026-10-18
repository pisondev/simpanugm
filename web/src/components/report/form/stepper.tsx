"use client"

import { Check } from "lucide-react"
import { useTranslations } from "next-intl"

import { cn } from "@/lib/utils"

export function Stepper({ steps, current, onJump }: { steps: string[]; current: number; onJump: (index: number) => void }) {
  const t = useTranslations("form")
  const total = steps.length
  const onReview = current >= total
  return (
    <nav aria-label={t("stepOf", { current: Math.min(current + 1, total), total })} className="mb-6">
      <p className="mb-3 text-sm text-muted-foreground">{onReview ? t("review") : t("stepOf", { current: current + 1, total })}</p>
      <ol className="flex gap-2">
        {steps.map((title, i) => {
          const done = i < current
          const active = i === current
          return (
            <li key={title} className="flex min-w-0 flex-1 flex-col gap-2">
              <button
                type="button"
                onClick={() => onJump(i)}
                disabled={i > current}
                aria-current={active ? "step" : undefined}
                className="group flex min-h-11 flex-col gap-2 text-left disabled:cursor-default"
              >
                <span className={cn("h-1.5 w-full rounded-full bg-muted", (done || active) && "bg-primary", onReview && "bg-primary")} />
                <span className={cn("hidden items-center gap-1 truncate text-xs font-medium text-muted-foreground sm:flex", active && "text-primary")}>
                  {done ? <Check className="size-3.5 text-success" aria-hidden /> : null}
                  {title}
                </span>
              </button>
            </li>
          )
        })}
      </ol>
    </nav>
  )
}
