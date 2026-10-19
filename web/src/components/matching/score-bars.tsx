import { useTranslations } from "next-intl"
import { useId, type CSSProperties } from "react"

import { AnimatedBar } from "@/components/motion/primitives"

import type { ExplanationLine, MatchExplanation, MatchScores } from "@/lib/api/types"
import { cn } from "@/lib/utils"

// members receive levels only, so a level stands for a typical value of its band (matching-engine 11)
const levelValue = { high: 90, medium: 65, low: 30 } as const

type Bar = { key: "visual" | "text" | "location" | "time"; line: ExplanationLine; score?: number | null }

function value(bar: Bar): number {
  if (bar.line.missing) return 0
  if (bar.score !== undefined && bar.score !== null) return Math.round(bar.score * 100)
  return bar.line.level ? levelValue[bar.line.level] : 0
}

// US-20: four bars with a label and a one line reason, the reason is text so no reader depends on the bar
export function ScoreBars({ explanation, scores }: { explanation: MatchExplanation; scores?: MatchScores }) {
  const t = useTranslations("matching")
  const prefix = useId()
  const bars: Bar[] = [
    { key: "visual", line: explanation.visual, score: scores?.visual },
    { key: "text", line: explanation.text, score: scores?.text },
    { key: "location", line: explanation.location, score: scores?.location },
    { key: "time", line: explanation.time, score: scores?.time },
  ]
  return (
    <ul className="motion-stagger flex flex-col gap-5">
      {bars.map((bar, index) => {
        const label = bar.key === "visual" && explanation.visual_mode === "cross_modal" ? t("bars.visualCrossModal") : t(`bars.${bar.key}`)
        const v = value(bar)
        const id = `${prefix}-bar-${bar.key}`
        return (
          <li key={bar.key} style={{ "--motion-index": index } as CSSProperties} className="flex flex-col gap-1.5">
            <div className="flex items-baseline justify-between gap-3 text-sm">
              <span id={id} className="font-medium">
                {label}
              </span>
              <span className="text-muted-foreground">{bar.line.missing ? t("level.missing") : bar.line.level ? t(`level.${bar.line.level}`) : null}</span>
            </div>
            <div
              role="meter"
              aria-labelledby={id}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={v}
              aria-valuetext={bar.line.reason}
              className="h-2 w-full overflow-hidden rounded-full bg-muted"
            >
              <AnimatedBar
                value={v}
                className={cn(
                  "h-full rounded-full",
                  bar.line.level === "high" ? "bg-success" : bar.line.level === "medium" ? "bg-info" : "bg-muted-foreground/50",
                )}
              />
            </div>
            <p className={cn("text-sm", bar.line.missing ? "text-muted-foreground" : "text-foreground")}>{bar.line.reason}</p>
            {bar.line.chips && bar.line.chips.length > 0 ? (
              <ul className="flex flex-wrap gap-1.5" aria-label={label}>
                {bar.line.chips.map((c) => (
                  <li key={c} className="rounded-full border px-2 py-0.5 text-xs">
                    {c}
                  </li>
                ))}
              </ul>
            ) : null}
          </li>
        )
      })}
      <li className="text-sm text-muted-foreground">{explanation.category.reason}</li>
    </ul>
  )
}
