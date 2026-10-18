import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { AuthCard } from "@/components/auth/auth-card"
import { VerifyEmail } from "@/components/auth/verify-email"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("auth.verify")
  return { title: t("title"), referrer: "no-referrer" }
}

export default async function VerifyPage({ searchParams }: PageProps<"/verifikasi">) {
  const { token } = await searchParams
  const t = await getTranslations("auth.verify")
  return (
    <AuthCard title={t("title")}>
      <VerifyEmail token={typeof token === "string" ? token : undefined} />
    </AuthCard>
  )
}
