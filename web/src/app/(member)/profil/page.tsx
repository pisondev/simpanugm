import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { SignOutAll } from "@/components/auth/sign-out-all"
import { PageHeader } from "@/components/layout/page-header"
import { Detail, DetailList } from "@/components/report/detail-list"
import { ButtonLink } from "@/components/ui/button-link"
import { requireMe } from "@/lib/session"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("profile")
  return { title: t("title") }
}

export default async function ProfilePage() {
  const [me, t, tc, ts, tn] = await Promise.all([
    requireMe(),
    getTranslations("profile"),
    getTranslations("common"),
    getTranslations("status"),
    getTranslations("nav"),
  ])
  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-6">
      <PageHeader title={t("title")} />
      <section className="rounded-xl border bg-card p-4">
        <DetailList>
          <Detail label={t("name")}>{me.full_name}</Detail>
          <Detail label={t("email")}>
            {me.email} <span className={me.email_verified ? "text-success" : "text-warning"}>({me.email_verified ? t("verified") : t("notVerified")})</span>
          </Detail>
          <Detail label={t("role")}>{ts(`role.${me.role}`)}</Detail>
          <Detail label={t("nimNip")}>{me.nim_nip ?? tc("emptyValue")}</Detail>
          <Detail label={t("phone")}>{me.phone ?? tc("emptyValue")}</Detail>
        </DetailList>
        <p className="mt-4 text-sm text-muted-foreground">{t("editLater")}</p>
      </section>
      <ButtonLink href="/laporan" variant="outline" className="w-fit">
        {me.role === "member" ? tn("myReports") : t("myReports")}
      </ButtonLink>
      <section className="flex flex-col gap-3 rounded-xl border bg-card p-4">
        <h2 className="font-semibold">{tn("signOutAll")}</h2>
        <p className="text-sm text-muted-foreground">{t("signOutAllHint")}</p>
        <SignOutAll />
      </section>
    </div>
  )
}
