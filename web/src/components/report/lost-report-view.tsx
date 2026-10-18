import { LoaderCircle, Lock, Sparkles, UserRound } from "lucide-react"
import { useTranslations } from "next-intl"
import type { ReactNode } from "react"

import type { HistoryEntry, LostReport } from "@/lib/api/types"
import { formatDateTime, formatRange } from "@/lib/format"

import { CodeChip } from "./code-chip"
import { Detail, DetailList } from "./detail-list"
import { PhotoStrip } from "./photo-strip"
import { StatusBadge } from "./status-badge"
import { Timeline } from "./timeline"

// matching normally answers within seconds, after two minutes the calm empty text replaces the spinner
const searchingWindowMs = 2 * 60_000

type Props = { report: LostReport; history: HistoryEntry[]; actions?: ReactNode; staff?: boolean; now: number; candidates?: ReactNode }

// information-architecture 5: status and the next action, candidates, then the fields, then the timeline
export function LostReportView({ report, history, actions, staff = false, now, candidates }: Props) {
  const t = useTranslations("report")
  const tc = useTranslations("common")
  const ts = useTranslations("status")
  const tp = useTranslations("profile")
  const open = report.status !== "closed" && report.status !== "returned"
  const searching = report.candidate_count === 0 && open && now - new Date(report.created_at).getTime() < searchingWindowMs
  const places = report.places.map((p) => p.location?.name ?? p.zone?.name).filter(Boolean)
  return (
    <article className="flex flex-col gap-6">
      <header className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <CodeChip code={report.code} />
          <StatusBadge kind="lost" status={report.status} />
          {report.proxy ? <span className="rounded-full border px-2.5 py-0.5 text-xs text-muted-foreground">{t("proxyBadge")}</span> : null}
        </div>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight md:text-2xl">{report.item_name}</h1>
            <p className="text-muted-foreground">
              {report.category.name} | {report.category.parent_name}
            </p>
          </div>
          {actions ? <div className="flex shrink-0 flex-wrap gap-2">{actions}</div> : null}
        </div>
      </header>

      {open ? (
        <section className="flex items-start gap-3 rounded-xl border border-info/30 bg-primary-soft/50 p-4" aria-live="polite">
          {searching ? (
            <LoaderCircle className="mt-0.5 size-5 shrink-0 animate-spin text-info" aria-hidden />
          ) : (
            <Sparkles className="mt-0.5 size-5 shrink-0 text-info" aria-hidden />
          )}
          <p>{report.candidate_count > 0 ? t("candidateCount", { count: report.candidate_count }) : searching ? t("searching") : t("noCandidates")}</p>
        </section>
      ) : null}

      {candidates}

      {report.status === "closed" && report.close_reason ? (
        <p className="rounded-xl border bg-card p-4">
          {t("closeReason")}:{" "}
          {ts.has(`closeReason.${report.close_reason}` as "closeReason.expired")
            ? ts(`closeReason.${report.close_reason}` as "closeReason.expired")
            : report.close_reason}
          {report.closed_at ? ` | ${formatDateTime(report.closed_at)}` : ""}
        </p>
      ) : null}

      <div className="grid gap-6 lg:grid-cols-[3fr_2fr]">
        <div className="flex flex-col gap-6">
          <section aria-labelledby="foto" className="flex flex-col gap-3">
            <h2 id="foto" className="sr-only">
              {t("photos")}
            </h2>
            <PhotoStrip photos={report.photos} alt={tc("photoAlt", { kind: tc("photoKindLost"), name: report.item_name })} />
          </section>
          <section className="rounded-xl border bg-card p-4">
            <DetailList>
              <Detail label={t("description")}>{report.description}</Detail>
              <Detail label={t("colours")}>{report.colours.length > 0 ? report.colours.join(", ") : tc("emptyValue")}</Detail>
              <Detail label={t("brand")}>{report.brand ?? tc("emptyValue")}</Detail>
              <Detail label={t("lostWindow")}>{formatRange(report.lost_from, report.lost_until)}</Detail>
              <Detail label={t("places")}>{report.location_unknown || places.length === 0 ? t("placeUnknown") : places.join(", ")}</Detail>
              {report.location_note ? <Detail label={t("locationNote")}>{report.location_note}</Detail> : null}
              <Detail label={t("createdAt")}>{formatDateTime(report.created_at)}</Detail>
            </DetailList>
          </section>
          {report.private_details ? (
            <section aria-labelledby="ciri" className="rounded-xl border border-warning/30 bg-warning/5 p-4">
              <h2 id="ciri" className="flex items-center gap-2 font-semibold">
                <Lock className="size-4 text-warning" aria-hidden />
                {t("privateDetails")}
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">{t("privateNote")}</p>
              <p className="mt-3 whitespace-pre-line">{report.private_details}</p>
            </section>
          ) : null}
          {staff && report.reporter ? (
            <section aria-labelledby="pelapor" className="rounded-xl border bg-card p-4">
              <h2 id="pelapor" className="mb-3 flex items-center gap-2 font-semibold">
                <UserRound className="size-4" aria-hidden />
                {t("reporter")}
              </h2>
              <DetailList>
                <Detail label={t("reporter")}>{report.reporter.full_name}</Detail>
                {report.reporter.email ? <Detail label={tp("email")}>{report.reporter.email}</Detail> : null}
                {report.reporter.phone ? <Detail label={tp("phone")}>{report.reporter.phone}</Detail> : null}
                {report.reporter.nim_nip ? <Detail label={tp("nimNip")}>{report.reporter.nim_nip}</Detail> : null}
              </DetailList>
            </section>
          ) : null}
        </div>
        <section aria-labelledby="riwayat" className="flex flex-col gap-4 rounded-xl border bg-card p-4">
          <h2 id="riwayat" className="font-semibold">
            {t("timeline")}
          </h2>
          <Timeline items={history} />
        </section>
      </div>
    </article>
  )
}
