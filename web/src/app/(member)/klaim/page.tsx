import { Inbox, PackageSearch } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { ClaimCard } from "@/components/claims/claim-card"
import { EmptyState } from "@/components/layout/empty-state"
import { PageHeader } from "@/components/layout/page-header"
import { PageTransition } from "@/components/motion/primitives"
import { Pager } from "@/components/layout/pager"
import { ButtonLink } from "@/components/ui/button-link"
import { load, serverApi } from "@/lib/api/server"
import { pick } from "@/lib/query"
import { requireMe } from "@/lib/session"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("claims")
  return { title: t("myTitle") }
}

export default async function MemberClaimsPage({ searchParams }: PageProps<"/klaim">) {
  await requireMe()
  const params = await searchParams
  const cursor = pick(params.cursor)
  const [api, t] = await Promise.all([serverApi(), getTranslations("claims")])
  const query = { cursor, limit: 20 }
  const list = await load(api.GET("/claims", { params: { query } }))

  return (
    <PageTransition className="mx-auto max-w-5xl">
      <PageHeader title={t("myTitle")} lead={t("myLead")} />
      {list.items.length === 0 ? (
        <EmptyState
          icon={Inbox}
          action={
            <ButtonLink href="/barang-temuan" variant="outline">
              <PackageSearch aria-hidden />
              {t("foundTitle")}
            </ButtonLink>
          }
        >
          {t("empty")}
        </EmptyState>
      ) : (
        <ul className="flex flex-col gap-3">
          {list.items.map((c) => (
            <ClaimCard key={c.id} claim={c} href={`/klaim/${c.code}`} staff={false} />
          ))}
        </ul>
      )}
      <Pager path="/klaim" filter={{}} cursor={cursor} next={list.next_cursor} />
    </PageTransition>
  )
}
