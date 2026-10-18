import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { AuthCard } from "@/components/auth/auth-card"
import { SetPasswordForm } from "@/components/auth/set-password-form"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("auth.invite")
  return { title: t("title"), referrer: "no-referrer" }
}

export default async function InvitePage({ searchParams }: PageProps<"/undangan">) {
  const { token } = await searchParams
  const t = await getTranslations("auth.invite")
  return (
    <AuthCard title={t("title")} lead={t("lead")}>
      {typeof token === "string" && token ? <SetPasswordForm token={token} purpose="invite" /> : <p className="text-muted-foreground">{t("missing")}</p>}
    </AuthCard>
  )
}
