import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { ActiveBadge, MasterNav } from "@/components/admin/master/master-nav"
import { PostDialog } from "@/components/admin/master/post-dialog"
import { ToggleActive } from "@/components/admin/master/toggle-active"
import { PageHeader } from "@/components/layout/page-header"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { load, serverApi } from "@/lib/api/server"
import { formatRanges, rangesFor, weekdays } from "@/lib/hours"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("admin.master")
  return { title: `${t("title")} | ${t("tabs.posts")}` }
}

export default async function PostsPage() {
  const api = await serverApi()
  const [posts, locations, t, th, tc] = await Promise.all([
    load(api.GET("/admin/posts")),
    load(api.GET("/admin/locations")),
    getTranslations("admin.master"),
    getTranslations("help"),
    getTranslations("common"),
  ])
  return (
    <>
      <PageHeader title={t("title")} lead={t("lead")} actions={<PostDialog locations={locations.items} />} />
      <MasterNav current="posts" />
      <div
        role="region"
        aria-label="Daftar pos"
        tabIndex={0}
        className="overflow-x-auto rounded-xl border bg-card focus-visible:outline-2 focus-visible:outline-ring"
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("colName")}</TableHead>
              <TableHead>{t("posts.location")}</TableHead>
              <TableHead>{t("posts.phone")}</TableHead>
              <TableHead>{t("posts.hours")}</TableHead>
              <TableHead>{t("colStatus")}</TableHead>
              <TableHead>
                <span className="sr-only">{t("colActions")}</span>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {posts.items.map((p) => (
              <TableRow key={p.id}>
                <TableCell className="max-w-64 font-medium whitespace-normal">{p.name}</TableCell>
                <TableCell className="max-w-56 whitespace-normal">
                  {p.location_name} | {p.zone_name}
                </TableCell>
                <TableCell>{p.phone ?? tc("emptyValue")}</TableCell>
                <TableCell className="text-sm whitespace-normal">
                  <ul>
                    {weekdays.map((d) => {
                      const ranges = rangesFor(p.opening_hours, d)
                      return (
                        <li key={d}>
                          <span className="text-muted-foreground">{th(`weekdays.${d}`)}:</span> {ranges.length > 0 ? formatRanges(ranges) : th("closed")}
                        </li>
                      )
                    })}
                  </ul>
                </TableCell>
                <TableCell>
                  <ActiveBadge active={p.active} edited={Boolean(p.edited_at)} />
                </TableCell>
                <TableCell className="text-right whitespace-nowrap">
                  <PostDialog post={p} locations={locations.items} />
                  <ToggleActive kind="post" id={p.id} name={p.name} active={p.active} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </>
  )
}
