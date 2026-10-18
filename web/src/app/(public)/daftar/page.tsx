import type { Metadata } from "next"
import Link from "next/link"
import { redirect } from "next/navigation"
import { getTranslations } from "next-intl/server"

import { AuthCard } from "@/components/auth/auth-card"
import { RegisterForm } from "@/components/auth/register-form"
import { homeFor } from "@/lib/paths"
import { getMe } from "@/lib/session"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("auth.register")
  return { title: t("title") }
}

export default async function RegisterPage() {
  const me = await getMe().catch(() => null)
  if (me) {
    redirect(homeFor(me.role))
  }
  const t = await getTranslations("auth.register")
  return (
    <AuthCard
      title={t("title")}
      lead={t("lead")}
      footer={
        <>
          {t("haveAccount")}{" "}
          <Link href="/masuk" className="font-medium text-primary underline-offset-4 hover:underline">
            {t("signIn")}
          </Link>
        </>
      }
    >
      <RegisterForm />
    </AuthCard>
  )
}
