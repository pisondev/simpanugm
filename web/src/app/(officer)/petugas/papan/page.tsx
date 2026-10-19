import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { PageHeader } from "@/components/layout/page-header"
import { ReviewBoardSearch, ReviewBoardView } from "@/components/officer/review-board"
import { Card, CardContent } from "@/components/ui/card"
import { load, serverApi } from "@/lib/api/server"
import type { OfficerBoard } from "@/lib/api/types"
import { requireStaff } from "@/lib/session"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("officer")
  return { title: t("boardTitle") }
}

interface PageProps {
  searchParams?: Promise<{ report_type?: string; report_id?: string }>
}

export default async function OfficerReviewBoardPage({ searchParams }: PageProps) {
  await requireStaff()
  const params = searchParams ? await searchParams : {}
  const reportType = params.report_type === "lost" ? "lost" : "found"
  const reportId = params.report_id?.trim()

  const [api, t] = await Promise.all([serverApi(), getTranslations("officer")])

  let board: OfficerBoard | null = null
  let notFound = false

  if (reportId) {
    try {
      board = await load(
        api.GET("/officer/board", {
          params: {
            query: {
              report_type: reportType,
              report_id: reportId,
            },
          },
        }),
      )
    } catch {
      notFound = true
    }
  }

  if (board && reportId) {
    return (
      <div className="max-w-7xl space-y-6">
        <PageHeader title={t("boardTitle")} lead={t("boardLead")} />
        <ReviewBoardView board={board} currentType={reportType} currentId={reportId} />
      </div>
    )
  }

  if (notFound) {
    return (
      <div className="max-w-6xl space-y-6">
        <PageHeader title={t("boardTitle")} lead={t("boardLead")} />
        <Card className="border-destructive/30">
          <CardContent className="space-y-4 pt-6">
            <p className="text-sm font-medium text-destructive">{t("boardReportNotFound")}</p>
            <p className="text-sm text-muted-foreground">{t("boardReportPrompt")}</p>
            <ReviewBoardSearch initialType={reportType} initialId={reportId} />
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="max-w-6xl space-y-6">
      <PageHeader title={t("boardTitle")} lead={t("boardLead")} />
      <Card>
        <CardContent className="space-y-4 pt-6">
          <p className="text-sm text-muted-foreground">{t("boardReportPrompt")}</p>
          <ReviewBoardSearch initialType={reportType} />
        </CardContent>
      </Card>
    </div>
  )
}
