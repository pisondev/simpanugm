import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { SignOutAll } from "@/components/auth/sign-out-all"
import { PageHeader } from "@/components/layout/page-header"
import { DeleteAccountDialog } from "@/components/profile/delete-account-dialog"
import { PasswordForm } from "@/components/profile/password-form"
import { SessionList } from "@/components/profile/session-list"
import { PageTransition } from "@/components/motion/primitives"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("profile")
  return { title: t("securityTitle") }
}

export default async function SecurityPage() {
  const [t, tn] = await Promise.all([getTranslations("profile"), getTranslations("nav")])
  return (
    <PageTransition className="mx-auto flex max-w-2xl flex-col gap-8">
      <PageHeader title={t("securityTitle")} lead={t("securityLead")} />
      <section aria-labelledby="kata-sandi" className="flex flex-col gap-4">
        <h2 id="kata-sandi" className="text-lg font-semibold">
          {t("passwordTitle")}
        </h2>
        <div className="surface p-4">
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
      <section className="surface flex flex-col gap-3 p-4">
        <h2 className="font-semibold">{tn("signOutAll")}</h2>
        <p className="text-sm text-muted-foreground">{t("signOutAllHint")}</p>
        <SignOutAll />
      </section>
      <section className="surface flex flex-col gap-3 border-destructive/30 bg-destructive/5 p-4">
        <h2 className="font-semibold text-destructive">{t("deleteAccountTitle")}</h2>
        <p className="text-sm text-muted-foreground">{t("deleteAccountLead")}</p>
        <div>
          <DeleteAccountDialog />
        </div>
      </section>
    </PageTransition>
  )
}
