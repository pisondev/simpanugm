import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { PageHeader } from "@/components/layout/page-header"
import { PageTransition } from "@/components/motion/primitives"
import { PreferencesForm } from "@/components/notifications/preferences-form"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("notifications")
  return { title: t("prefsTitle") }
}

export default async function NotificationPreferencesPage() {
  const t = await getTranslations("notifications")
  return (
    <PageTransition className="mx-auto flex max-w-2xl flex-col gap-4">
      <PageHeader title={t("prefsTitle")} lead={t("prefsLead")} />
      <section className="surface p-4">
        <PreferencesForm />
      </section>
    </PageTransition>
  )
}
