import type { Metadata } from "next"

import { ModerateReportDialog } from "@/components/admin/moderate-report-dialog"
import { OfficerCandidates } from "@/components/matching/officer-candidates"
import { LostReportView } from "@/components/report/lost-report-view"
import { load, serverApi } from "@/lib/api/server"
import { requestTime } from "@/lib/clock"

export async function generateMetadata({ params }: PageProps<"/petugas/laporan/hilang/[code]">): Promise<Metadata> {
  return { title: (await params).code }
}

export default async function OfficerLostPage({ params, searchParams }: PageProps<"/petugas/laporan/hilang/[code]">) {
  const now = await requestTime()
  const [{ code }, query] = await Promise.all([params, searchParams])
  const showLow = query.rendah === "1"
  const api = await serverApi()
  const [report, history, matches] = await Promise.all([
    load(api.GET("/lost-reports/{id}", { params: { path: { id: code } } })),
    load(api.GET("/lost-reports/{id}/history", { params: { path: { id: code } } })),
    load(api.GET("/lost-reports/{id}/matches", { params: { path: { id: code }, query: { include_low: showLow } } })),
  ])
  const candidates = (
    <OfficerCandidates
      items={matches.items}
      side="lost"
      reportId={report.id}
      showLow={showLow}
      toggleHref={showLow ? `/petugas/laporan/hilang/${report.code}` : `/petugas/laporan/hilang/${report.code}?rendah=1`}
    />
  )
  const actions = (
    <div className="flex flex-wrap gap-2">
      <ModerateReportDialog type="lost" id={report.id} />
    </div>
  )
  return <LostReportView report={report} history={history.items} actions={actions} staff now={now.getTime()} candidates={candidates} />
}
