import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { LegalPage } from "@/components/layout/legal-page"
import { privacyVersion } from "@/lib/paths"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("legal")
  return { title: t("privacyTitle") }
}

export default async function PrivacyPage() {
  const t = await getTranslations("legal")
  return <LegalPage title={t("privacyTitle")} version={t("privacyVersion", { version: privacyVersion })} sections={t.raw("privacy") as Section[]} />
}

type Section = { h: string; p: string }
