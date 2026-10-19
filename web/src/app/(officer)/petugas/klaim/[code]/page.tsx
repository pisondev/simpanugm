import { ArrowLeft } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { ClaimBody } from "@/components/claims/claim-body"
import { ClaimThread } from "@/components/claims/claim-thread"
import { ClaimChecklist, HandoverForm, InternalNote, OfficerClaimActions } from "@/components/claims/officer-panel"
import { StatusBadge } from "@/components/report/status-badge"
import { ButtonLink } from "@/components/ui/button-link"
import { load, serverApi } from "@/lib/api/server"
import { formatDateTime } from "@/lib/format"
import { requireStaff } from "@/lib/session"
import { claimOpen } from "@/lib/status"

export async function generateMetadata({ params }: PageProps<"/petugas/klaim/[code]">): Promise<Metadata> {
  const { code } = await params
  return { title: code }
}

export default async function OfficerClaimDetailPage({ params }: PageProps<"/petugas/klaim/[code]">) {
  await requireStaff()
  const { code } = await params
  const [api, t] = await Promise.all([serverApi(), getTranslations("claims")])
  const claim = await load(api.GET("/claims/{id}", { params: { path: { id: code } } }))

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <ButtonLink href="/petugas/klaim" variant="ghost" className="w-fit">
          <ArrowLeft aria-hidden />
          {t("backToQueue")}
        </ButtonLink>
        <span className="text-sm text-muted-foreground">{t("submittedAt", { at: formatDateTime(claim.created_at) })}</span>
      </div>

      <header className="flex flex-wrap items-center gap-3">
        <h1 className="font-mono text-2xl font-bold tracking-tight">{claim.code}</h1>
        <StatusBadge kind="claim" status={claim.status} />
      </header>

      <div className="grid gap-6 lg:grid-cols-[3fr_2fr]">
        <div className="flex flex-col gap-6">
          <OfficerClaimActions claim={claim} />
          {claim.status === "approved" ? <HandoverForm claim={claim} /> : null}
          <ClaimChecklist claim={claim} />
          <ClaimBody claim={claim} staff={true} />
        </div>
        <div className="flex flex-col gap-6">
          <InternalNote claim={claim} />
          <div className="rounded-xl border bg-card p-4">
            <ClaimThread code={claim.code} canPost={claimOpen.includes(claim.status)} />
          </div>
        </div>
      </div>
    </div>
  )
}
