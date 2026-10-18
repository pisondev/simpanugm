import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { PageHeader } from "@/components/layout/page-header"
import { PreferencesForm } from "@/components/notifications/preferences-form"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("notifications")
  return { title: t("prefsTitle") }
}

export default async function NotificationPreferencesPage() {
  const t = await getTranslations("notifications")
  return (
    <div className="mx-auto max-w-2xl">
      <PageHeader title={t("prefsTitle")} lead={t("prefsLead")} />
      <PreferencesForm />
    </div>
  )
}
