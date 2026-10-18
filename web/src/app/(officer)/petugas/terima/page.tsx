import { Warehouse } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { ReceivePanel } from "@/components/officer/receive-panel"
import { codePattern } from "@/lib/paths"
import { pick } from "@/lib/query"
import { requireStaff } from "@/lib/session"
import { staffPosts } from "@/lib/staff"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("officer")
  return { title: t("receiveTitle") }
}

export default async function ReceivePage({ searchParams }: PageProps<"/petugas/terima">) {
  const me = await requireStaff()
  const [posts, t, params] = await Promise.all([staffPosts(me), getTranslations("officer"), searchParams])
  const code = pick(params.code, codePattern)
  return (
    <div className="max-w-5xl">
      <PageHeader title={t("receiveTitle")} lead={t("receiveLead")} />
      {posts.length === 0 ? <EmptyState icon={Warehouse}>{t("noPosts")}</EmptyState> : <ReceivePanel posts={posts} initialCode={code} />}
    </div>
  )
}
