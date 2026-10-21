import { useTranslations } from "next-intl"

export function OfficerFeatureLabel({ feature }: { feature: "priority" | "board" | "checklist" | "search" | "duplicates" | "handover" }) {
  const t = useTranslations("officerDashboard")
  return <p className="mb-2 text-xs font-bold tracking-wide text-primary">{t(`features.${feature}.title`)}</p>
}
