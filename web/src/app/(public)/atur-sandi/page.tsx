import type { Metadata } from "next"
import Link from "next/link"
import { getTranslations } from "next-intl/server"

import { AuthCard } from "@/components/auth/auth-card"
import { SetPasswordForm } from "@/components/auth/set-password-form"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("auth.reset")
  return { title: t("title"), referrer: "no-referrer" }
}

export default async function ResetPage({ searchParams }: PageProps<"/atur-sandi">) {
  const { token } = await searchParams
  const t = await getTranslations("auth.reset")
  return (
    <AuthCard title={t("title")} lead={t("lead")}>
      {typeof token === "string" && token ? (
        <SetPasswordForm token={token} purpose="reset" />
      ) : (
        <p className="text-muted-foreground">
          {t("missing")}{" "}
          <Link href="/lupa-sandi" className="text-primary underline underline-offset-4">
            {t("requestNew")}
          </Link>
        </p>
      )}
    </AuthCard>
  )
}
