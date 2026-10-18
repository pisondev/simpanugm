import type { Metadata } from "next"

import { LostReportView } from "@/components/report/lost-report-view"
import { load, serverApi } from "@/lib/api/server"
import { requestTime } from "@/lib/clock"

export async function generateMetadata({ params }: PageProps<"/petugas/laporan/hilang/[code]">): Promise<Metadata> {
  return { title: (await params).code }
}

export default async function OfficerLostPage({ params }: PageProps<"/petugas/laporan/hilang/[code]">) {
  const now = await requestTime()
  const { code } = await params
  const api = await serverApi()
  const [report, history] = await Promise.all([
    load(api.GET("/lost-reports/{id}", { params: { path: { id: code } } })),
    load(api.GET("/lost-reports/{id}/history", { params: { path: { id: code } } })),
  ])
  return <LostReportView report={report} history={history.items} staff now={now.getTime()} />
}
