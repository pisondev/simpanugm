import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { PageHeader } from "@/components/layout/page-header"
import { OfficerFeatureLabel } from "@/components/dashboard/officer-feature-label"
import { DuplicateList } from "@/components/officer/duplicate-list"
import { load, serverApi } from "@/lib/api/server"
import { requireStaff } from "@/lib/session"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("officer")
  return { title: t("duplicatesTitle") }
}

interface PageProps {
  searchParams?: Promise<{ status?: string; type?: string }>
}

export default async function OfficerDuplicatesPage({ searchParams }: PageProps) {
  await requireStaff()
  const params = searchParams ? await searchParams : {}
  const status = params.status === "confirmed" || params.status === "rejected" ? params.status : "suggested"
  const reportType = params.type === "lost" || params.type === "found" ? params.type : undefined

  const [api, t] = await Promise.all([serverApi(), getTranslations("officer")])

  const list = await load(
    api.GET("/officer/duplicates", {
      params: {
        query: {
          status: status as "suggested" | "confirmed" | "rejected",
          type: reportType as "lost" | "found" | undefined,
          limit: 50,
        },
      },
    }),
  )

  return (
    <div className="max-w-5xl">
      <OfficerFeatureLabel feature="duplicates" />
      <PageHeader title={t("duplicatesTitle")} lead={t("duplicatesLead")} />
      <DuplicateList initialItems={list.items} currentStatus={status} />
    </div>
  )
}
