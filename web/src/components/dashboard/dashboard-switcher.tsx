import { HandHeart, Search } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"

import { cn } from "@/lib/utils"

export function DashboardSwitcher({ mode }: { mode: "pemilik" | "penemu" }) {
  const t = useTranslations("dashboardMode")
  return (
    <nav aria-label={t("label")} className="flex flex-wrap items-center gap-2">
      <p className="mr-2 text-xs font-semibold text-muted-foreground">{t("label")}</p>
      {(
        [
          { value: "pemilik", icon: Search },
          { value: "penemu", icon: HandHeart },
        ] as const
      ).map(({ value, icon: Icon }) => (
        <Link
          key={value}
          href={`/beranda?mode=${value}`}
          aria-current={mode === value ? "page" : undefined}
          className={cn(
            "inline-flex min-h-10 items-center gap-2 rounded-full border bg-white px-4 text-sm font-semibold text-muted-foreground hover:border-blue-300 hover:text-primary",
            mode === value && "border-blue-200 bg-blue-50 text-primary",
          )}
        >
          <Icon className="size-4" aria-hidden />
          {t(value)}
        </Link>
      ))}
    </nav>
  )
}
