import { useTranslations } from "next-intl"

export function FinderFeatureLabel({ feature }: { feature: "reporting" | "description" | "suggestions" | "duplicates" | "handover" | "history" }) {
  const t = useTranslations("finderDashboard")
  return <p className="mb-2 text-xs font-bold tracking-wide text-primary">{t(`features.${feature}.title`)}</p>
}
