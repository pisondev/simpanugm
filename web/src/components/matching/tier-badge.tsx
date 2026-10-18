import { CircleDashed, Sparkle, Sparkles } from "lucide-react"
import { useTranslations } from "next-intl"

import type { MatchTier } from "@/lib/api/types"
import { cn } from "@/lib/utils"

// ui-guidelines 4.4, the word and the icon carry the tier, colour only repeats it
const tiers = {
  high: { icon: Sparkles, className: "border-success bg-success text-white" },
  medium: { icon: Sparkle, className: "border-info/40 bg-transparent text-info" },
  low: { icon: CircleDashed, className: "border-border bg-transparent text-muted-foreground" },
} satisfies Record<MatchTier, { icon: typeof Sparkles; className: string }>

export function TierBadge({ tier, className }: { tier: MatchTier; className?: string }) {
  const t = useTranslations("matching.tier")
  const { icon: Icon, className: tone } = tiers[tier]
  return (
    <span
      className={cn("inline-flex w-fit shrink-0 items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium whitespace-nowrap", tone, className)}
    >
      <Icon className="size-3.5" aria-hidden />
      {t(tier)}
    </span>
  )
}

// "81%", ui-guidelines 6.3
export function percent(overall: number): string {
  return `${Math.round(overall * 100)}%`
}
