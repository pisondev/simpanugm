import { PackageOpen } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { OfficerCandidates } from "@/components/matching/officer-candidates"
import { CustodyDispositionDialog } from "@/components/officer/custody-actions"
import { FoundReportView } from "@/components/report/found-report-view"
import { ButtonLink } from "@/components/ui/button-link"
import { load, serverApi } from "@/lib/api/server"

export async function generateMetadata({ params }: PageProps<"/petugas/laporan/temuan/[code]">): Promise<Metadata> {
  return { title: (await params).code }
}

export default async function OfficerFoundPage({ params, searchParams }: PageProps<"/petugas/laporan/temuan/[code]">) {
  const [{ code }, query] = await Promise.all([params, searchParams])
  const showLow = query.rendah === "1"
  const [api, t] = await Promise.all([serverApi(), getTranslations("officer")])
  const [report, history, matches] = await Promise.all([
    load(api.GET("/found-reports/{id}", { params: { path: { id: code } } })),
    load(api.GET("/found-reports/{id}/history", { params: { path: { id: code } } })),
    load(api.GET("/found-reports/{id}/matches", { params: { path: { id: code }, query: { include_low: showLow } } })),
  ])
  const canDispose = report.status === "unclaimed" || report.custody_status === "in_custody"
  const actions = (
    <div className="flex flex-wrap gap-2">
      {report.custody_status === "with_finder" ? (
        <ButtonLink href={`/petugas/terima?code=${report.code}`}>
          <PackageOpen aria-hidden />
          {t("receiveThis")}
        </ButtonLink>
      ) : null}
      {canDispose ? <CustodyDispositionDialog report={report} /> : null}
    </div>
  )
  const candidates = (
    <OfficerCandidates
      items={matches.items}
      side="found"
      reportId={report.id}
      showLow={showLow}
      toggleHref={showLow ? `/petugas/laporan/temuan/${report.code}` : `/petugas/laporan/temuan/${report.code}?rendah=1`}
    />
  )
  return <FoundReportView report={report} history={history.items} actions={actions} staff matching={candidates} />
}
