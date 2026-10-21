import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { PageHeader } from "@/components/layout/page-header"
import { FinderFeatureLabel } from "@/components/dashboard/finder-feature-label"
import { PageTransition } from "@/components/motion/primitives"
import { FoundReportForm } from "@/components/report/form/found-report-form"
import { formData } from "@/lib/masterdata"
import { requireMe } from "@/lib/session"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("form.found")
  return { title: t("title") }
}

export default async function ReportFoundPage({ searchParams }: PageProps<"/lapor/temuan">) {
  const params = await searchParams
  const tf = await getTranslations("finderDashboard")
  const [me, data, t] = await Promise.all([requireMe(), formData(), getTranslations("form.found")])
  return (
    <PageTransition className="mx-auto max-w-2xl">
      <FinderFeatureLabel feature="reporting" />
      <PageHeader title={t("title")} />
      {params.fitur === "duplikat" ? (
        <section className="ai-panel mb-5 p-4">
          <FinderFeatureLabel feature="duplicates" />
          <p className="text-sm text-muted-foreground">{tf("duplicateHint")}</p>
        </section>
      ) : null}
      {me.email_verified ? <FoundReportForm mode="create" categories={data.categories} colours={data.colours} /> : null}
    </PageTransition>
  )
}
