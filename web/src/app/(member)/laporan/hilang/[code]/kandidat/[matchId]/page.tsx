import { ArrowLeft, Info, Warehouse } from "lucide-react"
import type { Metadata } from "next"
import { notFound } from "next/navigation"
import { getTranslations } from "next-intl/server"

import { DismissDialog } from "@/components/matching/dismiss-dialog"
import { MaybeMineButton } from "@/components/matching/match-actions"
import { ScoreBars } from "@/components/matching/score-bars"
import { percent, TierBadge } from "@/components/matching/tier-badge"
import { CodeChip } from "@/components/report/code-chip"
import { Detail, DetailList } from "@/components/report/detail-list"
import { PhotoStrip } from "@/components/report/photo-strip"
import { ButtonLink } from "@/components/ui/button-link"
import { load, serverApi } from "@/lib/api/server"
import { formatDateTime } from "@/lib/format"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("matching")
  return { title: t("title") }
}

// US-20 and US-21: one candidate with its reasons, and the owner's two answers
export default async function CandidatePage({ params }: PageProps<"/laporan/hilang/[code]/kandidat/[matchId]">) {
  const { code, matchId } = await params
  const [api, t, tr, tc] = await Promise.all([serverApi(), getTranslations("matching"), getTranslations("report"), getTranslations("common")])
  const match = await load(api.GET("/matches/{id}", { params: { path: { id: matchId } } }))
  if (match.lost_report.code !== code) {
    notFound()
  }
  const f = match.found_report
  const blurred = f.photos.some((p) => p.variant === "thumb_blur")
  const back = `/laporan/hilang/${code}#kandidat`
  return (
    <article className="flex flex-col gap-6">
      <ButtonLink href={back} variant="ghost" className="w-fit">
        <ArrowLeft aria-hidden />
        {t("back")}
      </ButtonLink>
      <header className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <TierBadge tier={match.tier} />
          <span className="font-semibold tabular-nums">
            {t("overall")} {percent(match.overall)}
          </span>
          <CodeChip code={f.code} />
        </div>
        <h1 className="text-xl font-semibold tracking-tight md:text-2xl">{f.item_name}</h1>
        <p className="text-muted-foreground">
          {f.category.name} | {f.category.parent_name}
        </p>
      </header>

      <div className="grid gap-6 lg:grid-cols-[3fr_2fr]">
        <div className="flex flex-col gap-6">
          <section aria-labelledby="barang" className="flex flex-col gap-3">
            <h2 id="barang" className="sr-only">
              {t("itemTitle")}
            </h2>
            <PhotoStrip
              photos={f.photos.map((p, i) => ({ media_id: p.media_id, position: i }))}
              alt={tc("photoAlt", { kind: tc("photoKindFound"), name: f.item_name })}
              only={blurred ? "thumb_blur" : undefined}
            />
            {blurred ? <p className="text-sm text-muted-foreground">{t("blurredNote")}</p> : null}
          </section>
          <section className="rounded-xl border bg-card p-4">
            <DetailList>
              <Detail label={tr("description")}>{f.description}</Detail>
              <Detail label={tr("colours")}>{f.colours.length > 0 ? f.colours.join(", ") : tc("emptyValue")}</Detail>
              <Detail label={tr("brand")}>{f.brand ?? tc("emptyValue")}</Detail>
              <Detail label={tr("foundAt")}>{formatDateTime(f.found_at)}</Detail>
              <Detail label={tr("foundLocation")}>{`${f.found_location.name}, ${f.found_location.zone_name}`}</Detail>
              <Detail label={tr("custody")}>{f.holding_post ? t("holdingPost", { post: f.holding_post.name }) : t("withFinder")}</Detail>
            </DetailList>
          </section>
        </div>
        <div className="flex flex-col gap-6">
          <section aria-labelledby="alasan" className="flex flex-col gap-4 rounded-xl border bg-card p-4">
            <h2 id="alasan" className="font-semibold">
              {t("whyTitle")}
            </h2>
            <ScoreBars explanation={match.explanation} />
          </section>
          <section aria-labelledby="langkah" className="flex items-start gap-3 rounded-xl border border-info/30 bg-primary-soft/50 p-4">
            {f.holding_post ? (
              <Warehouse className="mt-0.5 size-5 shrink-0 text-info" aria-hidden />
            ) : (
              <Info className="mt-0.5 size-5 shrink-0 text-info" aria-hidden />
            )}
            <div>
              <h2 id="langkah" className="font-semibold">
                {t("nextStepTitle")}
              </h2>
              <p className="mt-1">{f.holding_post ? t("nextStepInCustody", { post: f.holding_post.name, code }) : t("nextStepWithFinder")}</p>
            </div>
          </section>
          <div className="flex flex-wrap gap-2">
            <ButtonLink href={`/klaim/baru?pasangan=${match.id}`}>{t("claimAction")}</ButtonLink>
            <MaybeMineButton matchId={match.id} />
            <DismissDialog matchId={match.id} scope="owner" then={`/laporan/hilang/${code}`} />
          </div>
        </div>
      </div>
    </article>
  )
}
