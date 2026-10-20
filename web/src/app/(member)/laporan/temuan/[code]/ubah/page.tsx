import type { Metadata } from "next"
import { redirect } from "next/navigation"
import { getTranslations } from "next-intl/server"

import { PageHeader } from "@/components/layout/page-header"
import { PageTransition } from "@/components/motion/primitives"
import { FoundReportForm } from "@/components/report/form/found-report-form"
import { load, serverApi } from "@/lib/api/server"
import { formData } from "@/lib/masterdata"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("form.found")
  return { title: t("titleEdit") }
}

export default async function EditFoundPage({ params }: PageProps<"/laporan/temuan/[code]/ubah">) {
  const { code } = await params
  const [api, data, t] = await Promise.all([serverApi(), formData(), getTranslations("form.found")])
  const report = await load(api.GET("/found-reports/{id}", { params: { path: { id: code } } }))
  if (!report.editable) {
    redirect(`/laporan/temuan/${report.code}`)
  }
  return (
    <PageTransition className="mx-auto flex w-full max-w-2xl flex-col gap-5">
      <PageHeader title={t("titleEdit")} lead={report.code} />
      <FoundReportForm mode="edit" categories={data.categories} colours={data.colours} initial={report} />
    </PageTransition>
  )
}
