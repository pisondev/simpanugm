import { FileSearch, HandHeart, PackageSearch } from "lucide-react"
import type { Metadata } from "next"
import Link from "next/link"
import { getTranslations } from "next-intl/server"

import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { PageTransition } from "@/components/motion/primitives"
import { FoundCard, LostCard } from "@/components/report/report-card"
import { ButtonLink } from "@/components/ui/button-link"
import { load, serverApi } from "@/lib/api/server"
import { requireMe } from "@/lib/session"
import { foundOpen, lostOpen } from "@/lib/status"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("nav")
  return { title: t("home") }
}

export default async function HomePage() {
  const [me, api, t] = await Promise.all([requireMe(), serverApi(), getTranslations("member")])
  const [lost, found] = await Promise.all([
    load(api.GET("/lost-reports", { params: { query: { mine: true, status: [...lostOpen], limit: 5 } } })),
    load(api.GET("/found-reports", { params: { query: { mine: true, status: [...foundOpen], limit: 5 } } })),
  ])
  const first = me.full_name.split(/\s+/)[0] ?? me.full_name
  return (
    <PageTransition className="flex flex-col gap-6">
      <PageHeader title={t("homeTitle", { name: first })} lead={t("homeLead")} />
      <div className="grid gap-3 sm:grid-cols-3">
        <ButtonLink href="/lapor/hilang" size="lg" className="justify-start">
          <FileSearch aria-hidden />
          {t("reportLost")}
        </ButtonLink>
        <ButtonLink href="/lapor/temuan" size="lg" variant="outline" className="justify-start">
          <HandHeart aria-hidden />
          {t("reportFound")}
        </ButtonLink>
        <ButtonLink href="/barang-temuan" size="lg" variant="outline" className="justify-start">
          <PackageSearch aria-hidden />
          {t("browseFound")}
        </ButtonLink>
      </div>
      <div className="grid min-w-0 grid-cols-1 gap-8 lg:grid-cols-2">
        <section aria-labelledby="aktif-hilang" className="flex min-w-0 flex-col gap-3">
          <div className="flex items-center justify-between">
            <h2 id="aktif-hilang" className="text-lg font-semibold">
              {t("activeLost")}
            </h2>
            <Link href="/laporan" className="text-sm text-primary underline-offset-4 hover:underline">
              {t("seeAll")}
            </Link>
          </div>
          {lost.items.length === 0 ? (
            <EmptyState icon={FileSearch}>{t("emptyLost")}</EmptyState>
          ) : (
            <ul className="motion-stagger flex flex-col gap-3">
              {lost.items.map((r) => (
                <LostCard key={r.id} report={r} href={`/laporan/hilang/${r.code}`} />
              ))}
            </ul>
          )}
        </section>
        <section aria-labelledby="aktif-temuan" className="flex min-w-0 flex-col gap-3">
          <div className="flex items-center justify-between">
            <h2 id="aktif-temuan" className="text-lg font-semibold">
              {t("activeFound")}
            </h2>
            <Link href="/laporan?tab=temuan" className="text-sm text-primary underline-offset-4 hover:underline">
              {t("seeAll")}
            </Link>
          </div>
          {found.items.length === 0 ? (
            <EmptyState icon={HandHeart}>{t("emptyFound")}</EmptyState>
          ) : (
            <ul className="motion-stagger flex flex-col gap-3">
              {found.items.map((r) => (
                <FoundCard key={r.id} report={r} href={`/laporan/temuan/${r.code}`} />
              ))}
            </ul>
          )}
        </section>
      </div>
    </PageTransition>
  )
}
