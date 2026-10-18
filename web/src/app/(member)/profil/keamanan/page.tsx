import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { SignOutAll } from "@/components/auth/sign-out-all"
import { PageHeader } from "@/components/layout/page-header"
import { PasswordForm } from "@/components/profile/password-form"
import { SessionList } from "@/components/profile/session-list"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("profile")
  return { title: t("securityTitle") }
}

export default async function SecurityPage() {
  const [t, tn] = await Promise.all([getTranslations("profile"), getTranslations("nav")])
  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-8">
      <PageHeader title={t("securityTitle")} lead={t("securityLead")} />
      <section aria-labelledby="kata-sandi" className="flex flex-col gap-4">
        <h2 id="kata-sandi" className="text-lg font-semibold">
          {t("passwordTitle")}
        </h2>
        <div className="rounded-xl border bg-card p-4">
          <PasswordForm />
        </div>
      </section>
      <section aria-labelledby="perangkat" className="flex flex-col gap-4">
        <div>
          <h2 id="perangkat" className="text-lg font-semibold">
            {t("sessionsTitle")}
          </h2>
          <p className="text-sm text-muted-foreground">{t("sessionsLead")}</p>
        </div>
        <SessionList />
      </section>
      <section className="flex flex-col gap-3 rounded-xl border bg-card p-4">
        <h2 className="font-semibold">{tn("signOutAll")}</h2>
        <p className="text-sm text-muted-foreground">{t("signOutAllHint")}</p>
        <SignOutAll />
      </section>
    </div>
  )
}
