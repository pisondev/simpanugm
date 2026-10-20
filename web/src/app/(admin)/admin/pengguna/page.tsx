import { Users } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { EditUserDialog } from "@/components/admin/edit-user-dialog"
import { InviteDialog } from "@/components/admin/invite-dialog"
import { UserModerationMenu } from "@/components/admin/user-moderation-dialog"
import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { Pager } from "@/components/layout/pager"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { load, serverApi } from "@/lib/api/server"
import type { Role } from "@/lib/api/types"
import { formatRelative } from "@/lib/format"
import { pick, pickOne } from "@/lib/query"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("admin")
  return { title: t("usersTitle") }
}

const roles: Role[] = ["member", "officer", "admin"]

export default async function UsersPage({ searchParams }: PageProps<"/admin/pengguna">) {
  const params = await searchParams
  const filter = { q: pick(params.q)?.slice(0, 100), role: pickOne(params.role, roles) }
  const cursor = pick(params.cursor)
  const api = await serverApi()
  const [users, posts, t, ts, tc] = await Promise.all([
    load(api.GET("/admin/users", { params: { query: { ...filter, cursor, limit: 50 } } })),
    load(api.GET("/posts")),
    getTranslations("admin"),
    getTranslations("status"),
    getTranslations("common"),
  ])
  const postName = new Map(posts.items.map((p) => [p.id, p.name]))
  return (
    <>
      <PageHeader title={t("usersTitle")} lead={t("usersLead")} actions={<InviteDialog posts={posts.items} />} />
      <form method="get" role="search" className="mb-5 flex flex-col gap-3 rounded-xl border bg-card p-4 sm:flex-row sm:items-end">
        <div className="flex flex-1 flex-col gap-2">
          <label htmlFor="u-q" className="text-sm font-medium">
            {tc("search")}
          </label>
          <Input id="u-q" name="q" type="search" defaultValue={filter.q} placeholder={t("searchPlaceholder")} />
        </div>
        <div className="flex flex-col gap-2 sm:w-48">
          <label htmlFor="u-role" className="text-sm font-medium">
            {t("role")}
          </label>
          <NativeSelect id="u-role" name="role" defaultValue={filter.role ?? ""}>
            <option value="">{tc("all")}</option>
            {roles.map((r) => (
              <option key={r} value={r}>
                {ts(`role.${r}`)}
              </option>
            ))}
          </NativeSelect>
        </div>
        <Button type="submit" variant="outline">
          {tc("apply")}
        </Button>
      </form>
      {users.items.length === 0 ? (
        <EmptyState icon={Users}>{t("empty")}</EmptyState>
      ) : (
        <div
          role="region"
          aria-label="Daftar pengguna"
          tabIndex={0}
          className="overflow-x-auto rounded-xl border bg-card focus-visible:outline-2 focus-visible:outline-ring"
        >
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("colName")}</TableHead>
                <TableHead>{t("colRole")}</TableHead>
                <TableHead>{t("colStatus")}</TableHead>
                <TableHead>{t("colPosts")}</TableHead>
                <TableHead>{t("colLastLogin")}</TableHead>
                <TableHead>
                  <span className="sr-only">{t("edit")}</span>
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {users.items.map((u) => (
                <TableRow key={u.id}>
                  <TableCell>
                    <span className="flex flex-col">
                      <span className="font-medium">{u.full_name}</span>
                      <span className="text-xs text-muted-foreground">{u.email}</span>
                    </span>
                  </TableCell>
                  <TableCell>{ts(`role.${u.role}`)}</TableCell>
                  <TableCell>{ts(`account.${u.status}`)}</TableCell>
                  <TableCell className="max-w-64 whitespace-normal">{u.post_ids.map((id) => postName.get(id) ?? id).join(", ") || tc("none")}</TableCell>
                  <TableCell>{u.last_login_at ? formatRelative(u.last_login_at) : t("never")}</TableCell>
                  <TableCell className="text-right">
                    <div className="flex items-center justify-end gap-1">
                      <EditUserDialog user={u} posts={posts.items} />
                      <UserModerationMenu user={u} />
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
      <Pager path="/admin/pengguna" filter={filter} cursor={cursor} next={users.next_cursor} />
    </>
  )
}
