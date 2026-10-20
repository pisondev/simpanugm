import { Pencil } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { FinderHint } from "@/components/matching/finder-hint"
import { PageTransition } from "@/components/motion/primitives"
import { CloseReportDialog } from "@/components/report/close-report-dialog"
import { FoundReportView } from "@/components/report/found-report-view"
import { ButtonLink } from "@/components/ui/button-link"
import { load, serverApi } from "@/lib/api/server"

export async function generateMetadata({ params }: PageProps<"/laporan/temuan/[code]">): Promise<Metadata> {
  return { title: (await params).code }
}

export default async function FoundDetailPage({ params }: PageProps<"/laporan/temuan/[code]">) {
  const { code } = await params
  const [api, t] = await Promise.all([serverApi(), getTranslations("common")])
  const [report, history, hint] = await Promise.all([
    load(api.GET("/found-reports/{id}", { params: { path: { id: code } } })),
    load(api.GET("/found-reports/{id}/history", { params: { path: { id: code } } })),
    load(api.GET("/found-reports/{id}/match-hint", { params: { path: { id: code } } })),
  ])
  const post =
    report.intended_post && report.custody_status === "with_finder"
      ? (await load(api.GET("/posts"))).items.find((p) => p.id === report.intended_post?.id)
      : undefined
  const actions = report.editable ? (
    <>
      <ButtonLink href={`/laporan/temuan/${report.code}/ubah`} variant="outline">
        <Pencil aria-hidden />
        {t("edit")}
      </ButtonLink>
      <CloseReportDialog kind="found" id={report.id} />
    </>
  ) : null
  return (
    <PageTransition>
      <FoundReportView report={report} history={history.items} post={post} actions={actions} matching={<FinderHint items={hint.items} />} />
    </PageTransition>
  )
}
