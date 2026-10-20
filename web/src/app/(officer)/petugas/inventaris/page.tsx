import { ImageOff, Warehouse } from "lucide-react"
import type { Metadata } from "next"
import Link from "next/link"
import { getTranslations } from "next-intl/server"

import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { CustodyManagementActions } from "@/components/officer/custody-actions"
import { DueLabel } from "@/components/officer/due-label"
import { StatusBadge } from "@/components/report/status-badge"
import { Button } from "@/components/ui/button"
import { NativeSelect } from "@/components/ui/native-select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { mediaUrl } from "@/lib/api/client"
import { load, serverApi } from "@/lib/api/server"
import { pick, pickOne } from "@/lib/query"
import { requestTime } from "@/lib/clock"
import { requireStaff } from "@/lib/session"
import { staffPosts } from "@/lib/staff"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("officer")
  return { title: t("inventoryTitle") }
}

const sorts = ["retention_due_at", "-received_at"] as const

// US-32: what the post holds, sorted by due date by default
export default async function InventoryPage({ searchParams }: PageProps<"/petugas/inventaris">) {
  const now = await requestTime()
  const me = await requireStaff()
  const [posts, params, t, ts, tc] = await Promise.all([
    staffPosts(me),
    searchParams,
    getTranslations("officer"),
    getTranslations("status"),
    getTranslations("common"),
  ])
  if (posts.length === 0) {
    return (
      <>
        <PageHeader title={t("inventoryTitle")} />
        <EmptyState icon={Warehouse}>{t("noPosts")}</EmptyState>
      </>
    )
  }
  const requested = pick(params.post)
  const post = posts.find((p) => p.id === requested) ?? posts[0]
  const sort = pickOne(params.sort, sorts) ?? "retention_due_at"
  if (!post) {
    return null
  }
  const api = await serverApi()
  const [inventory, activePosts] = await Promise.all([
    load(api.GET("/posts/{id}/inventory", { params: { path: { id: post.id }, query: { sort } } })),
    load(api.GET("/posts")).then((result) => result.items),
  ])
  return (
    <>
      <PageHeader title={t("inventoryTitle")} lead={t("inventoryLead", { post: inventory.post.name })} />
      <form method="get" className="mb-5 flex flex-col gap-3 rounded-xl border bg-card p-4 sm:flex-row sm:items-end">
        {posts.length > 1 ? (
          <div className="flex flex-1 flex-col gap-2">
            <label htmlFor="inv-post" className="text-sm font-medium">
              {t("choosePost")}
            </label>
            <NativeSelect id="inv-post" name="post" defaultValue={post.id}>
              {posts.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </NativeSelect>
          </div>
        ) : (
          <input type="hidden" name="post" value={post.id} />
        )}
        <div className="flex flex-1 flex-col gap-2">
          <label htmlFor="inv-sort" className="text-sm font-medium">
            {t("sort")}
          </label>
          <NativeSelect id="inv-sort" name="sort" defaultValue={sort}>
            <option value="retention_due_at">{t("sortDue")}</option>
            <option value="-received_at">{t("sortReceived")}</option>
          </NativeSelect>
        </div>
        <Button type="submit" variant="outline">
          {tc("apply")}
        </Button>
      </form>
      {inventory.items.length === 0 ? (
        <EmptyState icon={Warehouse}>{t("inventoryEmpty")}</EmptyState>
      ) : (
        <div role="region" aria-label={t("inventoryTableLabel")} className="overflow-hidden rounded-xl border bg-card">
          <div className="divide-y md:hidden">
            {inventory.items.map((item) => (
              <article key={item.id} className="flex flex-col gap-3 p-4">
                <div className="flex min-w-0 items-center gap-3">
                  {item.thumb_media_id ? (
                    <img src={mediaUrl(item.thumb_media_id, "thumb")} alt="" className="size-12 shrink-0 rounded-md object-cover" loading="lazy" />
                  ) : (
                    <span className="flex size-12 shrink-0 items-center justify-center rounded-md bg-muted">
                      <ImageOff className="size-4 text-muted-foreground" aria-hidden />
                    </span>
                  )}
                  <Link href={`/petugas/laporan/temuan/${item.code}`} className="flex min-w-0 flex-col hover:underline">
                    <span className="font-medium">{item.item_name}</span>
                    <span className="font-mono text-xs break-words text-muted-foreground">
                      {item.code} | {item.category_name}
                    </span>
                  </Link>
                </div>
                <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
                  <div>
                    <dt className="text-muted-foreground">{t("colSlot")}</dt>
                    <dd className="font-mono break-words">{item.storage_slot}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">{t("colAge")}</dt>
                    <dd>{t("ageDays", { count: item.age_days })}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">{t("colDue")}</dt>
                    <dd>
                      <DueLabel iso={item.retention_due_at} now={now} />
                    </dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">{t("colSensitivity")}</dt>
                    <dd>{ts(`sensitivity.${item.sensitivity}`)}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">{t("colClaim")}</dt>
                    <dd>{ts(`claimState.${item.claim_state}`)}</dd>
                  </div>
                  <div>
                    <dt className="text-muted-foreground">{t("colStatus")}</dt>
                    <dd>
                      <StatusBadge kind="found" status={item.status} />
                    </dd>
                  </div>
                </dl>
                <CustodyManagementActions
                  report={{ id: item.id, code: item.code, category: { sensitivity: item.sensitivity } }}
                  posts={activePosts}
                  currentPostId={post.id}
                  canDispose={item.status === "unclaimed" && item.claim_state === "none"}
                />
              </article>
            ))}
          </div>
          <div tabIndex={0} className="hidden overflow-x-auto focus-visible:ring-2 focus-visible:ring-ring focus-visible:outline-none md:block">
            <Table className="min-w-[900px]">
              <TableHeader>
                <TableRow>
                  <TableHead>{t("colItem")}</TableHead>
                  <TableHead>{t("colSlot")}</TableHead>
                  <TableHead>{t("colAge")}</TableHead>
                  <TableHead>{t("colDue")}</TableHead>
                  <TableHead>{t("colSensitivity")}</TableHead>
                  <TableHead>{t("colClaim")}</TableHead>
                  <TableHead>{t("colStatus")}</TableHead>
                  <TableHead>{t("colActions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {inventory.items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>
                      <Link href={`/petugas/laporan/temuan/${item.code}`} className="flex items-center gap-3 hover:underline">
                        {item.thumb_media_id ? (
                          <img src={mediaUrl(item.thumb_media_id, "thumb")} alt="" className="size-10 rounded-md object-cover" loading="lazy" />
                        ) : (
                          <span className="flex size-10 items-center justify-center rounded-md bg-muted">
                            <ImageOff className="size-4 text-muted-foreground" aria-hidden />
                          </span>
                        )}
                        <span className="flex flex-col">
                          <span className="font-medium">{item.item_name}</span>
                          <span className="font-mono text-xs text-muted-foreground">
                            {item.code} | {item.category_name}
                          </span>
                        </span>
                      </Link>
                    </TableCell>
                    <TableCell className="font-mono">{item.storage_slot}</TableCell>
                    <TableCell>{t("ageDays", { count: item.age_days })}</TableCell>
                    <TableCell>
                      <DueLabel iso={item.retention_due_at} now={now} />
                    </TableCell>
                    <TableCell>{ts(`sensitivity.${item.sensitivity}`)}</TableCell>
                    <TableCell>{ts(`claimState.${item.claim_state}`)}</TableCell>
                    <TableCell>
                      <StatusBadge kind="found" status={item.status} />
                    </TableCell>
                    <TableCell>
                      <CustodyManagementActions
                        report={{ id: item.id, code: item.code, category: { sensitivity: item.sensitivity } }}
                        posts={activePosts}
                        currentPostId={post.id}
                        canDispose={item.status === "unclaimed" && item.claim_state === "none"}
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </div>
      )}
    </>
  )
}
