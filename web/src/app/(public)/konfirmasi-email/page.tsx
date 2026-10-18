import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { AuthCard } from "@/components/auth/auth-card"
import { ConfirmEmail } from "@/components/auth/confirm-email"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("auth.confirmEmail")
  return { title: t("title"), referrer: "no-referrer" }
}

export default async function ConfirmEmailPage({ searchParams }: PageProps<"/konfirmasi-email">) {
  const { token } = await searchParams
  const t = await getTranslations("auth.confirmEmail")
  return (
    <AuthCard title={t("title")}>
      <ConfirmEmail token={typeof token === "string" ? token : undefined} />
    </AuthCard>
  )
}
