import { Pencil } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { CandidateList } from "@/components/matching/candidate-card"
import { CloseReportDialog } from "@/components/report/close-report-dialog"
import { LostReportView } from "@/components/report/lost-report-view"
import { ButtonLink } from "@/components/ui/button-link"
import { load, serverApi } from "@/lib/api/server"
import { requestTime } from "@/lib/clock"

export async function generateMetadata({ params }: PageProps<"/laporan/hilang/[code]">): Promise<Metadata> {
  return { title: (await params).code }
}

export default async function LostDetailPage({ params }: PageProps<"/laporan/hilang/[code]">) {
  const now = await requestTime()
  const { code } = await params
  const [api, t] = await Promise.all([serverApi(), getTranslations("common")])
  const [report, history, matches] = await Promise.all([
    load(api.GET("/lost-reports/{id}", { params: { path: { id: code } } })),
    load(api.GET("/lost-reports/{id}/history", { params: { path: { id: code } } })),
    load(api.GET("/lost-reports/{id}/matches", { params: { path: { id: code } } })),
  ])
  const open = report.status !== "closed" && report.status !== "returned"
  const actions = (
    <>
      {report.editable ? (
        <ButtonLink href={`/laporan/hilang/${report.code}/ubah`} variant="outline">
          <Pencil aria-hidden />
          {t("edit")}
        </ButtonLink>
      ) : null}
      {open ? <CloseReportDialog kind="lost" id={report.id} /> : null}
    </>
  )
  const candidates =
    open || matches.items.length > 0 ? <CandidateList items={matches.items} hrefFor={(c) => `/laporan/hilang/${report.code}/kandidat/${c.id}`} /> : null
  return <LostReportView report={report} history={history.items} actions={actions} now={now.getTime()} candidates={candidates} />
}
