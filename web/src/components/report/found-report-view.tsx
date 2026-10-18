import { Hand, Lock, MapPin, Phone, UserRound, Warehouse } from "lucide-react"
import { useTranslations } from "next-intl"
import type { ReactNode } from "react"

import type { FoundReport, HistoryEntry, Post } from "@/lib/api/types"
import { formatDate, formatDateTime } from "@/lib/format"
import { formatRanges, rangesFor, weekdays } from "@/lib/hours"

import { CodeChip } from "./code-chip"
import { Detail, DetailList } from "./detail-list"
import { PhotoStrip } from "./photo-strip"
import { OpenNow } from "./post-hours"
import { StatusBadge } from "./status-badge"
import { Timeline } from "./timeline"

type Props = { report: FoundReport; history: HistoryEntry[]; post?: Post; actions?: ReactNode; staff?: boolean; matching?: ReactNode }

function PostCard({ post }: { post: Post }) {
  const t = useTranslations("report")
  const th = useTranslations("help")
  return (
    <div className="mt-3 flex flex-col gap-2 rounded-lg border bg-card p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <span className="font-medium">{post.name}</span>
        <OpenNow hours={post.opening_hours} />
      </div>
      <p className="flex items-center gap-2 text-sm text-muted-foreground">
        <MapPin className="size-4" aria-hidden />
        {t("handInAddress", { location: `${post.location_name}, ${post.zone_name}` })}
      </p>
      {post.phone ? (
        <a href={`tel:${post.phone}`} className="flex min-h-11 items-center gap-2 text-sm text-primary">
          <Phone className="size-4" aria-hidden />
          {t("handInPhone", { phone: post.phone })}
        </a>
      ) : null}
      <details className="text-sm">
        <summary className="min-h-11 cursor-pointer py-2 font-medium">{t("handInHours")}</summary>
        <table className="w-full">
          <tbody>
            {weekdays.map((d) => {
              const ranges = rangesFor(post.opening_hours, d)
              return (
                <tr key={d} className="border-t">
                  <th scope="row" className="py-1 pr-3 text-left font-normal text-muted-foreground">
                    {th(`weekdays.${d}`)}
                  </th>
                  <td className="py-1 text-right">{ranges.length > 0 ? formatRanges(ranges) : th("closed")}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </details>
    </div>
  )
}

export function FoundReportView({ report, history, post, actions, staff = false, matching }: Props) {
  const t = useTranslations("report")
  const tc = useTranslations("common")
  const ts = useTranslations("status")
  const tp = useTranslations("profile")
  const withFinder = report.custody_status === "with_finder"
  return (
    <article className="flex flex-col gap-6">
      <header className="flex flex-col gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <CodeChip code={report.code} />
          <StatusBadge kind="found" status={report.status} />
          <StatusBadge kind="custody" status={report.custody_status} />
          {report.proxy ? <span className="rounded-full border px-2.5 py-0.5 text-xs text-muted-foreground">{t("proxyBadge")}</span> : null}
        </div>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight md:text-2xl">{report.item_name}</h1>
            <p className="text-muted-foreground">
              {report.category.name} | {report.category.parent_name} | {ts(`sensitivity.${report.category.sensitivity}`)}
            </p>
          </div>
          {actions ? <div className="flex shrink-0 flex-wrap gap-2">{actions}</div> : null}
        </div>
      </header>

      {!staff && withFinder && !report.custody_keep ? (
        <section className="rounded-xl border border-warning/30 bg-warning/5 p-4">
          <h2 className="flex items-center gap-2 font-semibold">
            <Hand className="size-5 text-warning" aria-hidden />
            {t("handInTitle")}
          </h2>
          <p className="mt-1">{t("handInBody", { code: report.code, post: report.intended_post?.name ?? "" })}</p>
          {post ? <PostCard post={post} /> : null}
        </section>
      ) : null}
      {!staff && withFinder && report.custody_keep ? (
        <section className="rounded-xl border border-warning/30 bg-warning/5 p-4">
          <h2 className="flex items-center gap-2 font-semibold">
            <Hand className="size-5 text-warning" aria-hidden />
            {t("keepTitle")}
          </h2>
          <p className="mt-1">{t("keepBody")}</p>
        </section>
      ) : null}
      {!staff && report.custody_status === "in_custody" ? (
        <section className="rounded-xl border border-primary/30 bg-primary-soft/50 p-4">
          <h2 className="flex items-center gap-2 font-semibold">
            <Warehouse className="size-5 text-primary" aria-hidden />
            {t("inCustodyTitle")}
          </h2>
          <p className="mt-1">{t("inCustodyBody", { post: report.holding_post?.name ?? "" })}</p>
        </section>
      ) : null}

      {matching}

      <div className="grid gap-6 lg:grid-cols-[3fr_2fr]">
        <div className="flex flex-col gap-6">
          <PhotoStrip photos={report.photos} alt={tc("photoAlt", { kind: tc("photoKindFound"), name: report.item_name })} />
          <section className="rounded-xl border bg-card p-4">
            <DetailList>
              <Detail label={t("description")}>{report.description}</Detail>
              <Detail label={t("colours")}>{report.colours.length > 0 ? report.colours.join(", ") : tc("emptyValue")}</Detail>
              <Detail label={t("brand")}>{report.brand ?? tc("emptyValue")}</Detail>
              <Detail label={t("foundAt")}>{formatDateTime(report.found_at)}</Detail>
              <Detail label={t("foundLocation")}>
                {report.found_location.name} | {report.found_location.zone_name}
              </Detail>
              {report.location_note ? <Detail label={t("locationNote")}>{report.location_note}</Detail> : null}
              {report.intended_post && withFinder ? <Detail label={t("intendedPost")}>{report.intended_post.name}</Detail> : null}
              {report.holding_post ? <Detail label={t("holdingPost")}>{report.holding_post.name}</Detail> : null}
              {staff && report.storage_slot ? <Detail label={t("storageSlot")}>{report.storage_slot}</Detail> : null}
              {report.received_at ? <Detail label={t("receivedAt")}>{formatDateTime(report.received_at)}</Detail> : null}
              {staff && report.retention_due_at ? <Detail label={t("retentionDue")}>{formatDate(report.retention_due_at)}</Detail> : null}
              {report.status === "closed" && report.close_reason ? (
                <Detail label={t("closeReason")}>
                  {ts.has(`closeReason.${report.close_reason}` as "closeReason.expired")
                    ? ts(`closeReason.${report.close_reason}` as "closeReason.expired")
                    : report.close_reason}
                </Detail>
              ) : null}
            </DetailList>
          </section>
          {report.hidden_details ? (
            <section aria-labelledby="tersembunyi" className="rounded-xl border border-warning/30 bg-warning/5 p-4">
              <h2 id="tersembunyi" className="flex items-center gap-2 font-semibold">
                <Lock className="size-4 text-warning" aria-hidden />
                {t("hiddenDetails")}
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">{t("privateNote")}</p>
              <p className="mt-3 whitespace-pre-line">{report.hidden_details}</p>
            </section>
          ) : null}
          {staff && (report.finder || report.proxy_finder_name) ? (
            <section aria-labelledby="penemu" className="rounded-xl border bg-card p-4">
              <h2 id="penemu" className="mb-3 flex items-center gap-2 font-semibold">
                <UserRound className="size-4" aria-hidden />
                {t("finder")}
              </h2>
              <DetailList>
                <Detail label={tp("name")}>{report.finder?.full_name ?? report.proxy_finder_name}</Detail>
                {report.finder?.email ? <Detail label={tp("email")}>{report.finder.email}</Detail> : null}
                {report.finder?.phone ? <Detail label={tp("phone")}>{report.finder.phone}</Detail> : null}
                {report.finder?.nim_nip ? <Detail label={tp("nimNip")}>{report.finder.nim_nip}</Detail> : null}
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
