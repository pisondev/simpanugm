import { Warehouse } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { FoundReportForm } from "@/components/report/form/found-report-form"
import { formData } from "@/lib/masterdata"
import { requireStaff } from "@/lib/session"
import { staffPosts } from "@/lib/staff"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("officer")
  return { title: t("proxyFoundTitle") }
}

// an item handed in at the desk goes straight into custody at one of the officer's posts
export default async function ProxyFoundPage() {
  const me = await requireStaff()
  const [data, posts, t] = await Promise.all([formData(), staffPosts(me), getTranslations("officer")])
  return (
    <div className="max-w-2xl">
      <PageHeader title={t("proxyFoundTitle")} lead={t("proxyLead")} />
      {posts.length === 0 ? (
        <EmptyState icon={Warehouse}>{t("proxyNoPost")}</EmptyState>
      ) : (
        <FoundReportForm mode="proxy" categories={data.categories} colours={data.colours} staffPosts={posts} />
      )}
    </div>
  )
}
