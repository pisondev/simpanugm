import { useTranslations } from "next-intl"

export function OwnerFeatureLabel({ feature }: { feature: "reporting" | "assistant" | "candidates" | "explanation" | "claim" | "history" }) {
  const t = useTranslations("ownerDashboard")
  return <p className="mb-2 text-xs font-bold tracking-wide text-primary">{t(`features.${feature}.title`)}</p>
}
