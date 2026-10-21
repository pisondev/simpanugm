import type { Metadata } from "next"
import { redirect } from "next/navigation"
import { getTranslations } from "next-intl/server"

import { ClaimForm, type LostOption, type Question } from "@/components/claims/claim-form"
import { OwnerFeatureLabel } from "@/components/dashboard/owner-feature-label"
import { PageHeader } from "@/components/layout/page-header"
import { PageTransition } from "@/components/motion/primitives"
import { load, serverApi } from "@/lib/api/server"
import { pick } from "@/lib/query"
import { requireMe } from "@/lib/session"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("claims")
  return { title: t("newTitle") }
}

export default async function NewClaimPage({ searchParams }: PageProps<"/klaim/baru">) {
  await requireMe()
  const params = await searchParams
  const matchIdParam = pick(params.pasangan)
  const foundIdParam = pick(params.temuan)
  const categoryIdParam = pick(params.kategori)
  const nameParam = pick(params.nama)
  const lostCodeParam = pick(params.hilang)

  if (!matchIdParam && !foundIdParam) {
    redirect("/barang-temuan")
  }

  const [api, t] = await Promise.all([serverApi(), getTranslations("claims")])

  let foundId = foundIdParam ?? ""
  let foundName = nameParam ?? ""
  let categoryId = categoryIdParam ?? ""
  let matchId: string | undefined
  let presetLostId: string | undefined

  if (matchIdParam) {
    const match = await load(api.GET("/matches/{id}", { params: { path: { id: matchIdParam } } }))
    foundId = match.found_report.id
    foundName = match.found_report.item_name
    categoryId = match.found_report.category.id
    matchId = match.id
    presetLostId = match.lost_report.id
  } else if (foundId && (!foundName || !categoryId)) {
    try {
      const foundReport = await load(api.GET("/found-reports/{id}", { params: { path: { id: foundId } } }))
      foundName = foundReport.item_name
      categoryId = foundReport.category.id
    } catch {
      // fallback to params
    }
  }

  const [catsRes, lostReportsRes, coloursRes] = await Promise.all([
    load(api.GET("/categories")),
    load(api.GET("/lost-reports", { params: { query: { mine: true } } })),
    load(api.GET("/colours")),
  ])

  const allLeaves = catsRes.items.flatMap((p) => (p.children && p.children.length > 0 ? p.children : [p]))
  const category = allLeaves.find((c) => c.id === categoryId)
  const questions: Question[] = (category?.verification_questions ?? []).map((q) => ({
    key: q.key,
    question: q.question,
    required: q.required,
    check_hint: q.check_hint ?? undefined,
  }))

  const claimable = ["reported", "potential_match", "under_verification"]
  const lostOptions: LostOption[] = lostReportsRes.items
    .filter((r) => claimable.includes(r.status))
    .map((r) => ({
      id: r.id,
      code: r.code,
      label: `${r.code} - ${r.item_name}`,
    }))

  if (lostCodeParam && !presetLostId) {
    const foundLost = lostReportsRes.items.find((r) => r.code === lostCodeParam || r.id === lostCodeParam)
    if (foundLost) {
      presetLostId = foundLost.id
    }
  }

  return (
    <PageTransition className="mx-auto max-w-2xl">
      <OwnerFeatureLabel feature="claim" />
      <PageHeader title={t("newTitle")} />
      <section className="surface p-4 sm:p-5">
        <ClaimForm
          foundId={foundId}
          foundName={foundName}
          questions={questions}
          lostOptions={lostOptions}
          presetLostId={presetLostId}
          matchId={matchId}
          categories={catsRes.items}
          colours={coloursRes.items}
          defaultCategoryId={categoryId}
        />
      </section>
    </PageTransition>
  )
}
