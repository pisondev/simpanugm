import { Bell, ChevronRight, ShieldCheck } from "lucide-react"
import type { Metadata } from "next"
import Link from "next/link"
import { getTranslations } from "next-intl/server"

import { PageHeader } from "@/components/layout/page-header"
import { EmailChangeDialog } from "@/components/profile/email-change-dialog"
import { ProfileForm } from "@/components/profile/profile-form"
import { ButtonLink } from "@/components/ui/button-link"
import { requireMe } from "@/lib/session"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("profile")
  return { title: t("title") }
}

// US-05: contact details, the email change, then security and notification settings
export default async function ProfilePage() {
  const [me, t, ts, tn] = await Promise.all([requireMe(), getTranslations("profile"), getTranslations("status"), getTranslations("nav")])
  const links = [
    { href: "/profil/keamanan", label: t("linksSecurity"), icon: ShieldCheck },
    { href: "/profil/notifikasi", label: t("linksNotifications"), icon: Bell },
  ]
  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-6">
      <PageHeader title={t("title")} lead={t("lead")} />
      <section className="flex flex-col gap-4 rounded-xl border bg-card p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-sm text-muted-foreground">{t("email")}</p>
            <p className="font-medium break-all">{me.email}</p>
            <p className={me.email_verified ? "text-sm text-success-text" : "text-sm text-warning-text"}>
              {me.email_verified ? t("verified") : t("notVerified")}
            </p>
          </div>
          <EmailChangeDialog />
        </div>
        <p className="text-sm text-muted-foreground">
          {t("role")}: {ts(`role.${me.role}`)}
        </p>
      </section>
      <section className="rounded-xl border bg-card p-4">
        <ProfileForm me={me} />
      </section>
      <nav aria-label={t("title")} className="flex flex-col divide-y rounded-xl border bg-card">
        {links.map((l) => (
          <Link key={l.href} href={l.href} className="flex min-h-14 items-center gap-3 px-4 hover:bg-muted">
            <l.icon className="size-5 text-primary" aria-hidden />
            <span className="flex-1 font-medium">{l.label}</span>
            <ChevronRight className="size-4 text-muted-foreground" aria-hidden />
          </Link>
        ))}
      </nav>
      <ButtonLink href="/laporan" variant="outline" className="w-fit">
        {me.role === "member" ? tn("myReports") : t("myReports")}
      </ButtonLink>
    </div>
  )
}
