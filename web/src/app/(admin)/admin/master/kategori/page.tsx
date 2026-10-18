import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { CategoryDialog } from "@/components/admin/master/category-dialog"
import { CompatDialog } from "@/components/admin/master/compat-dialog"
import { ActiveBadge, MasterNav } from "@/components/admin/master/master-nav"
import { ToggleActive } from "@/components/admin/master/toggle-active"
import { PageHeader } from "@/components/layout/page-header"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { load, serverApi } from "@/lib/api/server"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("admin.master")
  return { title: `${t("title")} | ${t("tabs.categories")}` }
}

export default async function CategoriesPage() {
  const api = await serverApi()
  const [cats, pairs, t, ts] = await Promise.all([
    load(api.GET("/admin/categories")),
    load(api.GET("/admin/category-compatibility")),
    getTranslations("admin.master"),
    getTranslations("status.sensitivity"),
  ])
  const parents = cats.items.filter((c) => !c.parent_id)
  const leaves = cats.items.filter((c) => c.parent_id)
  return (
    <>
      <PageHeader
        title={t("title")}
        lead={t("lead")}
        actions={
          <div className="flex flex-wrap gap-2">
            <CategoryDialog parents={parents} leaf={false} />
            <CategoryDialog parents={parents} leaf />
          </div>
        }
      />
      <MasterNav current="categories" />
      <div className="rounded-xl border bg-card">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("colName")}</TableHead>
              <TableHead>{t("colSlug")}</TableHead>
              <TableHead>{t("categories.sensitivity")}</TableHead>
              <TableHead>{t("categories.questions")}</TableHead>
              <TableHead>{t("colStatus")}</TableHead>
              <TableHead>
                <span className="sr-only">{t("colActions")}</span>
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {parents.flatMap((p) => [
              <TableRow key={p.id} className="bg-muted/40">
                <TableCell className="font-semibold">{p.name}</TableCell>
                <TableCell className="text-muted-foreground">{p.slug}</TableCell>
                <TableCell />
                <TableCell />
                <TableCell>
                  <ActiveBadge active={p.active} edited={Boolean(p.edited_at)} />
                </TableCell>
                <TableCell className="text-right whitespace-nowrap">
                  <CategoryDialog category={p} parents={parents} leaf={false} />
                  <ToggleActive kind="category" id={p.id} name={p.name} active={p.active} />
                </TableCell>
              </TableRow>,
              ...leaves
                .filter((l) => l.parent_id === p.id)
                .map((l) => (
                  <TableRow key={l.id}>
                    <TableCell className="pl-6">{l.name}</TableCell>
                    <TableCell className="text-muted-foreground">{l.slug}</TableCell>
                    <TableCell>{l.sensitivity ? ts(l.sensitivity) : null}</TableCell>
                    <TableCell>{t("categories.questionCount", { count: l.verification_questions.length })}</TableCell>
                    <TableCell>
                      <ActiveBadge active={l.active} edited={Boolean(l.edited_at)} />
                    </TableCell>
                    <TableCell className="text-right whitespace-nowrap">
                      <CategoryDialog category={l} parents={parents} leaf />
                      <ToggleActive kind="category" id={l.id} name={l.name} active={l.active} />
                    </TableCell>
                  </TableRow>
                )),
            ])}
          </TableBody>
        </Table>
      </div>

      <section aria-labelledby="kompatibilitas" className="mt-8 flex flex-col gap-3">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 id="kompatibilitas" className="font-semibold">
              {t("categories.compatTitle")}
            </h2>
            <p className="text-sm text-muted-foreground">{t("categories.compatLead")}</p>
          </div>
          <CompatDialog leaves={leaves.filter((l) => l.active)} />
        </div>
        {pairs.items.length === 0 ? (
          <p className="text-muted-foreground">{t("categories.compatEmpty")}</p>
        ) : (
          <div className="rounded-xl border bg-card">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("categories.compatA")}</TableHead>
                  <TableHead>{t("categories.compatB")}</TableHead>
                  <TableHead className="text-right">{t("categories.factor")}</TableHead>
                  <TableHead>{t("colStatus")}</TableHead>
                  <TableHead>
                    <span className="sr-only">{t("colActions")}</span>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {pairs.items.map((p) => (
                  <TableRow key={`${p.category_a}-${p.category_b}`}>
                    <TableCell>{p.name_a}</TableCell>
                    <TableCell>{p.name_b}</TableCell>
                    <TableCell className="text-right tabular-nums">{p.factor.toFixed(2)}</TableCell>
                    <TableCell>
                      <ActiveBadge active={p.active} />
                    </TableCell>
                    <TableCell className="text-right">
                      <CompatDialog leaves={leaves} pair={p} />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </section>
    </>
  )
}
