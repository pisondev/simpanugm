import { getTranslations } from "next-intl/server"

import { Detail, DetailList } from "@/components/report/detail-list"
import { PhotoStrip } from "@/components/report/photo-strip"
import type { Claim } from "@/lib/api/types"
import { formatDateTime } from "@/lib/format"

// what the claim is about, shown to the claimant and to officers alike. Officer only fields arrive only for officers
export async function ClaimBody({ claim, staff }: { claim: Claim; staff: boolean }) {
  const [t, tc, tr] = await Promise.all([getTranslations("claims"), getTranslations("common"), getTranslations("report")])
  const f = claim.found_report
  const l = claim.lost_report
  const blurred = f.photos.some((p) => p.variant === "thumb_blur")
  return (
    <div className="flex flex-col gap-6">
      <section aria-labelledby="c-barang" className="flex flex-col gap-3">
        <h2 id="c-barang" className="font-semibold">
          {t("foundTitle")}
        </h2>
        <PhotoStrip
          photos={f.photos.map((p, i) => ({ media_id: p.media_id, position: i }))}
          alt={tc("photoAlt", { kind: tc("photoKindFound"), name: f.item_name })}
          only={blurred && !staff ? "thumb_blur" : undefined}
        />
        <div className="rounded-xl border bg-card p-4">
          <DetailList>
            <Detail label={t("code")}>{f.code}</Detail>
            <Detail label={tr("itemName")}>{f.item_name}</Detail>
            <Detail label={tr("category")}>{`${f.category.name} | ${f.category.parent_name}`}</Detail>
            <Detail label={tr("colours")}>{f.colours.length > 0 ? f.colours.join(", ") : tc("emptyValue")}</Detail>
            <Detail label={tr("brand")}>{f.brand ?? tc("emptyValue")}</Detail>
            <Detail label={tr("foundAt")}>{formatDateTime(f.found_at)}</Detail>
            <Detail label={tr("foundLocation")}>{f.found_location_name}</Detail>
            <Detail label={tr("custody")}>{f.holding_post ? t("atPost", { post: f.holding_post.name }) : t("withFinder")}</Detail>
            {staff && f.storage_slot ? <Detail label={t("slot")}>{f.storage_slot}</Detail> : null}
            {staff && f.hidden_details ? <Detail label={t("hiddenDetails")}>{f.hidden_details}</Detail> : null}
          </DetailList>
        </div>
      </section>

      <section aria-labelledby="c-hilang" className="flex flex-col gap-3">
        <h2 id="c-hilang" className="font-semibold">
          {t("lostTitle")}
        </h2>
        <div className="rounded-xl border bg-card p-4">
          <DetailList>
            <Detail label={t("code")}>{l.code}</Detail>
            <Detail label={tr("itemName")}>{l.item_name}</Detail>
            <Detail label={tr("colours")}>{l.colours.length > 0 ? l.colours.join(", ") : tc("emptyValue")}</Detail>
            <Detail label={tr("brand")}>{l.brand ?? tc("emptyValue")}</Detail>
            <Detail label={tr("lostWindow")}>{`${formatDateTime(l.lost_from)} - ${formatDateTime(l.lost_until)}`}</Detail>
            {staff && l.private_details ? <Detail label={t("privateDetails")}>{l.private_details}</Detail> : null}
          </DetailList>
        </div>
      </section>

      <section aria-labelledby="c-klaim" className="flex flex-col gap-3">
        <h2 id="c-klaim" className="font-semibold">
          {t("claimTitle")}
        </h2>
        <div className="rounded-xl border bg-card p-4">
          <DetailList>
            {staff && claim.claimant ? (
              <Detail label={t("claimant")}>{[claim.claimant.full_name, claim.claimant.email, claim.claimant.phone].filter(Boolean).join(" | ")}</Detail>
            ) : null}
            {staff && claim.visitor_name ? <Detail label={t("visitor")}>{claim.visitor_name}</Detail> : null}
            <Detail label={t("statement")}>{claim.statement}</Detail>
            {claim.answers.map((a) => (
              <Detail key={a.key} label={a.question}>
                {a.answer || tc("emptyValue")}
              </Detail>
            ))}
            {claim.preferred_visit ? <Detail label={t("preferredVisit")}>{claim.preferred_visit}</Detail> : null}
          </DetailList>
          {claim.evidence.length > 0 ? (
            <div className="mt-4">
              <p className="mb-2 text-sm text-muted-foreground">{t("evidence")}</p>
              <PhotoStrip photos={claim.evidence} alt={t("evidenceAlt")} />
            </div>
          ) : null}
        </div>
      </section>

      {claim.decision_reason || claim.decision_note ? (
        <section aria-labelledby="c-keputusan" className="rounded-xl border bg-card p-4">
          <h2 id="c-keputusan" className="mb-2 font-semibold">
            {t("decisionTitle")}
          </h2>
          <DetailList>
            {claim.decision_reason ? <Detail label={t("decisionReason")}>{t(`reason.${claim.decision_reason}`)}</Detail> : null}
            {claim.decision_note ? <Detail label={t("decisionNote")}>{claim.decision_note}</Detail> : null}
          </DetailList>
        </section>
      ) : null}
    </div>
  )
}
