import { PackageOpen } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { FoundReportView } from "@/components/report/found-report-view"
import { ButtonLink } from "@/components/ui/button-link"
import { load, serverApi } from "@/lib/api/server"

export async function generateMetadata({ params }: PageProps<"/petugas/laporan/temuan/[code]">): Promise<Metadata> {
  return { title: (await params).code }
}

export default async function OfficerFoundPage({ params }: PageProps<"/petugas/laporan/temuan/[code]">) {
  const { code } = await params
  const [api, t] = await Promise.all([serverApi(), getTranslations("officer")])
  const [report, history] = await Promise.all([
    load(api.GET("/found-reports/{id}", { params: { path: { id: code } } })),
    load(api.GET("/found-reports/{id}/history", { params: { path: { id: code } } })),
  ])
  const actions =
    report.custody_status === "with_finder" ? (
      <ButtonLink href={`/petugas/terima?code=${report.code}`}>
        <PackageOpen aria-hidden />
        {t("receiveThis")}
      </ButtonLink>
    ) : null
  return <FoundReportView report={report} history={history.items} actions={actions} staff />
}
