import { ShieldQuestion } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { ClaimCard } from "@/components/claims/claim-card"
import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { Pager } from "@/components/layout/pager"
import { Button } from "@/components/ui/button"
import { ButtonLink } from "@/components/ui/button-link"
import { NativeSelect } from "@/components/ui/native-select"
import type { ClaimStatus } from "@/lib/api/types"
import { load, serverApi } from "@/lib/api/server"
import { pick, pickOne } from "@/lib/query"
import { requireStaff } from "@/lib/session"
import { staffPosts } from "@/lib/staff"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("claims")
  return { title: t("queueTitle") }
}

const claimStatuses: ClaimStatus[] = ["submitted", "under_review", "info_requested", "on_hold", "approved", "handed_over", "rejected", "withdrawn", "expired"]

export default async function OfficerClaimsPage({ searchParams }: PageProps<"/petugas/klaim">) {
  const me = await requireStaff()
  const [posts, params, t, ts, tc] = await Promise.all([
    staffPosts(me),
    searchParams,
    getTranslations("claims"),
    getTranslations("status.claim"),
    getTranslations("common"),
  ])

  const status = pickOne(params.status, claimStatuses)
  const postId = pick(params.post_id)
  const cursor = pick(params.cursor)

  const api = await serverApi()
  const query = {
    cursor,
    limit: 20,
    status: status ? [status] : undefined,
    post_id: postId,
  }
  const list = await load(api.GET("/claims", { params: { query } }))

  return (
    <div className="max-w-5xl">
      <PageHeader title={t("queueTitle")} lead={t("queueLead")} />

      <form method="get" className="mb-5 flex flex-col gap-3 rounded-xl border bg-card p-4 sm:flex-row sm:items-end">
        <div className="flex flex-1 flex-col gap-2">
          <label htmlFor="filter-status" className="text-sm font-medium">
            {t("filterStatus")}
          </label>
          <NativeSelect id="filter-status" name="status" defaultValue={status ?? ""}>
            <option value="">{t("allStatuses")}</option>
            {claimStatuses.map((s) => (
              <option key={s} value={s}>
                {ts(s)}
              </option>
            ))}
          </NativeSelect>
        </div>

        {posts.length > 0 ? (
          <div className="flex flex-1 flex-col gap-2">
            <label htmlFor="filter-post" className="text-sm font-medium">
              {t("filterPost")}
            </label>
            <NativeSelect id="filter-post" name="post_id" defaultValue={postId ?? ""}>
              <option value="">{t("allPosts")}</option>
              {posts.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </NativeSelect>
          </div>
        ) : null}

        <div className="flex gap-2">
          <Button type="submit">{tc("apply")}</Button>
          <ButtonLink href="/petugas/klaim" variant="ghost">
            {tc("reset")}
          </ButtonLink>
        </div>
      </form>

      {list.items.length === 0 ? (
        <EmptyState icon={ShieldQuestion}>{t("emptyQueue")}</EmptyState>
      ) : (
        <ul className="flex flex-col gap-3">
          {list.items.map((c) => (
            <ClaimCard key={c.id} claim={c} href={`/petugas/klaim/${c.code}`} staff={true} />
          ))}
        </ul>
      )}

      <Pager path="/petugas/klaim" filter={{ status: status ?? undefined, post_id: postId ?? undefined }} cursor={cursor} next={list.next_cursor} />
    </div>
  )
}
