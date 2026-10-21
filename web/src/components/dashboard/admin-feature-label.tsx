import { useTranslations } from "next-intl"

export type AdminFeature = "monitoring" | "drift" | "fusion" | "clusters" | "hotspots" | "trends"

export function AdminFeatureLabel({ feature }: { feature: AdminFeature }) {
  const t = useTranslations("adminDashboard")
  return <p className="mb-2 text-xs font-bold tracking-wide text-primary">{t(`features.${feature}.title`)}</p>
}
