"use client"

import { Check } from "lucide-react"
import { useTranslations } from "next-intl"

import type { Colour } from "@/lib/api/types"
import { cn } from "@/lib/utils"

export function ColourPicker({
  palette,
  value,
  onChange,
  describedBy,
}: {
  palette: Colour[]
  value: string[]
  onChange: (v: string[]) => void
  describedBy?: string
}) {
  const t = useTranslations("form.colours")
  function toggle(name: string) {
    if (value.includes(name)) {
      onChange(value.filter((c) => c !== name))
    } else if (value.length < 2) {
      onChange([...value, name])
    }
  }
  return (
    <div className="flex flex-wrap gap-2" aria-describedby={describedBy}>
      {palette.map((c) => {
        const on = value.includes(c.name)
        return (
          <button
            key={c.name}
            type="button"
            aria-pressed={on}
            aria-label={on ? t("selected", { name: c.name }) : c.name}
            onClick={() => toggle(c.name)}
            disabled={!on && value.length >= 2}
            className={cn(
              "inline-flex min-h-11 items-center gap-2 rounded-full border bg-card px-3 text-sm transition-colors hover:bg-muted disabled:opacity-40",
              on && "border-primary bg-primary-soft text-primary hover:bg-primary-soft",
            )}
          >
            <span className="size-5 rounded-full border border-foreground/15" style={{ backgroundColor: c.hex }} aria-hidden />
            {c.name}
            {on ? <Check className="size-4" aria-hidden /> : null}
          </button>
        )
      })}
    </div>
  )
}
