import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { PageHeader } from "@/components/layout/page-header"
import { FoundReportForm } from "@/components/report/form/found-report-form"
import { formData } from "@/lib/masterdata"
import { requireMe } from "@/lib/session"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("form.found")
  return { title: t("title") }
}

export default async function ReportFoundPage() {
  const [me, data, t] = await Promise.all([requireMe(), formData(), getTranslations("form.found")])
  return (
    <div className="mx-auto max-w-2xl">
      <PageHeader title={t("title")} />
      {me.email_verified ? <FoundReportForm mode="create" categories={data.categories} colours={data.colours} /> : null}
    </div>
  )
}
