import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { PageHeader } from "@/components/layout/page-header"
import { LostReportForm } from "@/components/report/form/lost-report-form"
import { formData } from "@/lib/masterdata"
import { requireStaff } from "@/lib/session"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("officer")
  return { title: t("proxyLostTitle") }
}

// US-12: a walk in visitor without an account
export default async function ProxyLostPage() {
  const [, data, t] = await Promise.all([requireStaff(), formData(), getTranslations("officer")])
  return (
    <div className="max-w-2xl">
      <PageHeader title={t("proxyLostTitle")} lead={t("proxyLead")} />
      <LostReportForm mode="proxy" categories={data.categories} colours={data.colours} />
    </div>
  )
}
