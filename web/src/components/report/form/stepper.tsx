"use client"

import { Check } from "lucide-react"
import { useTranslations } from "next-intl"

import { cn } from "@/lib/utils"

export function Stepper({ steps, current, onJump }: { steps: string[]; current: number; onJump: (index: number) => void }) {
  const t = useTranslations("form")
  const total = steps.length
  const onReview = current >= total
  return (
    <nav aria-label={t("stepOf", { current: Math.min(current + 1, total), total })}>
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
                aria-label={title}
                className="group flex min-h-11 flex-col gap-2 rounded-sm text-left focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none disabled:cursor-default"
              >
                <span className={cn("h-1.5 w-full rounded-full bg-muted", (done || active) && "bg-primary", onReview && "bg-primary")} />
                <span className={cn("flex items-start gap-1 text-xs font-medium text-muted-foreground", active && "text-primary")}>
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
