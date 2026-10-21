import { FileSearch, HandHeart } from "lucide-react"
import type { Metadata } from "next"
import Link from "next/link"
import { getTranslations } from "next-intl/server"
import { cache } from "react"

import { DashboardSwitcher } from "@/components/dashboard/dashboard-switcher"
import { FinderDashboard } from "@/components/dashboard/finder-dashboard"
import { OwnerDashboard } from "@/components/dashboard/owner-dashboard"
import { EmptyState } from "@/components/layout/empty-state"
import { PageTransition } from "@/components/motion/primitives"
import { FoundCard, LostCard } from "@/components/report/report-card"
import { ButtonLink } from "@/components/ui/button-link"
import { load, serverApi } from "@/lib/api/server"
import { requireMe } from "@/lib/session"
import { foundOpen, lostOpen } from "@/lib/status"

const dashboardData = cache(async () => {
  const [me, api, t] = await Promise.all([requireMe(), serverApi(), getTranslations("member")])
  const [lost, found, claims] = await Promise.all([
    load(api.GET("/lost-reports", { params: { query: { mine: true, status: [...lostOpen], limit: 100 } } })),
    load(api.GET("/found-reports", { params: { query: { mine: true, status: [...foundOpen], limit: 100 } } })),
    load(api.GET("/claims", { params: { query: { limit: 5 } } })),
  ])
  return { me, t, lost, found, claims }
})

async function dashboardMode(params: PageProps<"/beranda">["searchParams"]) {
  const [query, data] = await Promise.all([params, dashboardData()])
  return query.mode === "penemu" || (query.mode !== "pemilik" && data.lost.items.length === 0 && data.found.items.length > 0) ? "penemu" : "pemilik"
}

export async function generateMetadata({ searchParams }: PageProps<"/beranda">): Promise<Metadata> {
  const mode = await dashboardMode(searchParams)
  const t = await getTranslations(mode === "penemu" ? "finderDashboard" : "ownerDashboard")
  return { title: t("pageTitle") }
}

export default async function HomePage({ searchParams }: PageProps<"/beranda">) {
  const [{ me, t, lost, found, claims }, mode] = await Promise.all([dashboardData(), dashboardMode(searchParams)])
  return (
    <PageTransition className="flex flex-col gap-8">
      <DashboardSwitcher mode={mode} />
      {mode === "penemu" ? (
        <FinderDashboard name={me.full_name} reports={found.items} />
      ) : (
        <OwnerDashboard name={me.full_name} reports={lost.items} hasMoreReports={Boolean(lost.next_cursor)} claims={claims.items} />
      )}
      <div className="grid min-w-0 gap-6 border-t pt-7 lg:grid-cols-2">
        <section aria-labelledby="aktif-hilang" className="flex min-w-0 flex-col gap-3">
          <div className="flex items-center justify-between gap-3">
            <h2 id="aktif-hilang" className="text-lg font-semibold">
              {t("activeLost")}
            </h2>
            <Link href="/laporan" className="text-sm font-medium text-primary underline-offset-4 hover:underline">
              {t("seeAll")}
            </Link>
          </div>
          {lost.items.length === 0 ? (
            <EmptyState icon={FileSearch}>{t("emptyLost")}</EmptyState>
          ) : (
            <ul className="flex flex-col gap-3">
              {lost.items.slice(0, 3).map((r) => (
                <LostCard key={r.id} report={r} href={`/laporan/hilang/${r.code}`} />
              ))}
            </ul>
          )}
        </section>
        <section aria-labelledby="aktif-temuan" className="flex min-w-0 flex-col gap-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <h2 id="aktif-temuan" className="text-lg font-semibold">
              {t("activeFound")}
            </h2>
            <ButtonLink href="/lapor/temuan" variant="outline" size="sm">
              <HandHeart aria-hidden />
              {t("reportFound")}
            </ButtonLink>
          </div>
          {found.items.length === 0 ? (
            <EmptyState icon={HandHeart}>{t("emptyFound")}</EmptyState>
          ) : (
            <ul className="flex flex-col gap-3">
              {found.items.slice(0, 3).map((r) => (
                <FoundCard key={r.id} report={r} href={`/laporan/temuan/${r.code}`} />
              ))}
            </ul>
          )}
          <Link href="/laporan?tab=temuan" className="w-fit text-sm font-medium text-primary underline-offset-4 hover:underline">
            {t("seeAll")}
          </Link>
        </section>
      </div>
    </PageTransition>
  )
}
