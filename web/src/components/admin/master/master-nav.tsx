import Link from "next/link"
import { useTranslations } from "next-intl"

import { cn } from "@/lib/utils"

const tabs = [
  { key: "categories", href: "/admin/master/kategori" },
  { key: "locations", href: "/admin/master/lokasi" },
  { key: "zones", href: "/admin/master/zona" },
  { key: "posts", href: "/admin/master/pos" },
] as const

export type MasterTab = (typeof tabs)[number]["key"]

// the four master data pages of information-architecture 2.4 as one set of tabs
export function MasterNav({ current }: { current: MasterTab }) {
  const t = useTranslations("admin.master.tabs")
  return (
    <nav aria-label={t("categories")} className="mb-5 flex gap-1 overflow-x-auto border-b">
      {tabs.map((tab) => (
        <Link
          key={tab.key}
          href={tab.href}
          aria-current={tab.key === current ? "page" : undefined}
          className={cn(
            "-mb-px inline-flex min-h-11 items-center border-b-2 px-3 text-sm font-medium whitespace-nowrap",
            tab.key === current ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground",
          )}
        >
          {t(tab.key)}
        </Link>
      ))}
    </nav>
  )
}

export function ActiveBadge({ active, edited }: { active: boolean; edited?: boolean }) {
  const t = useTranslations("admin.master")
  return (
    <span className="flex flex-wrap gap-1">
      <span
        className={cn(
          "inline-flex w-fit items-center rounded-full border px-2 py-0.5 text-xs font-medium",
          active ? "border-success/30 bg-success/10 text-success-text" : "border-border bg-secondary text-secondary-foreground",
        )}
      >
        {active ? t("active") : t("inactive")}
      </span>
      {edited ? <span className="inline-flex w-fit items-center rounded-full border px-2 py-0.5 text-xs text-muted-foreground">{t("edited")}</span> : null}
    </span>
  )
}
