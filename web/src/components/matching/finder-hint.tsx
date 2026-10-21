import { Search } from "lucide-react"
import { useTranslations } from "next-intl"

import type { MatchHintItem } from "@/lib/api/types"
import { FinderFeatureLabel } from "@/components/dashboard/finder-feature-label"

// US-14: the finder learns that someone may be looking, with the lost item name and category only
export function FinderHint({ items }: { items: MatchHintItem[] }) {
  const t = useTranslations("matching")
  const tf = useTranslations("finderDashboard")
  return (
    <section id="saran-pemilik" aria-labelledby="petunjuk-pemilik" className="ai-panel motion-fade flex scroll-mt-24 items-start gap-3 p-5">
      <Search className="mt-0.5 size-5 shrink-0 text-info" aria-hidden />
      <div className="flex flex-col gap-1">
        <FinderFeatureLabel feature="suggestions" />
        <h2 id="petunjuk-pemilik" className="font-semibold">
          {t("hintTitle")}
        </h2>
        <p>{items.length ? t("hintBody") : tf("noHints")}</p>
        <ul className="mt-1 list-inside list-disc text-sm text-muted-foreground">
          {items.map((i) => (
            <li key={`${i.item_name}-${i.category_name}`}>{t("hintItem", { name: i.item_name, category: i.category_name })}</li>
          ))}
        </ul>
      </div>
    </section>
  )
}
