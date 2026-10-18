import type { Metadata } from "next"
import Link from "next/link"
import { redirect } from "next/navigation"
import { getTranslations } from "next-intl/server"

import { AuthCard } from "@/components/auth/auth-card"
import { FormAlert } from "@/components/auth/form-alert"
import { LoginForm } from "@/components/auth/login-form"
import { homeFor, safeNext } from "@/lib/paths"
import { getMe } from "@/lib/session"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("auth.login")
  return { title: t("title") }
}

export default async function SignInPage({ searchParams }: PageProps<"/masuk">) {
  const params = await searchParams
  const next = typeof params.next === "string" ? params.next : undefined
  const me = await getMe().catch(() => null)
  if (me) {
    redirect(safeNext(next, homeFor(me.role)))
  }
  const t = await getTranslations("auth.login")
  return (
    <AuthCard
      title={t("title")}
      lead={t("lead")}
      footer={
        <>
          {t("noAccount")}{" "}
          <Link href="/daftar" className="font-medium text-primary underline-offset-4 hover:underline">
            {t("register")}
          </Link>
        </>
      }
    >
      <div className="flex flex-col gap-5">
        {params.expired ? <FormAlert>{t("expired")}</FormAlert> : null}
        <LoginForm next={next} />
      </div>
    </AuthCard>
  )
}
