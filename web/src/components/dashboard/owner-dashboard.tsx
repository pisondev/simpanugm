"use client"

import { useQuery } from "@tanstack/react-query"
import {
  ArrowRight,
  Bell,
  ChartNoAxesColumnIncreasing,
  Check,
  ChevronRight,
  ClipboardCheck,
  FilePlus2,
  HandHeart,
  History,
  ImagePlus,
  LoaderCircle,
  MapPin,
  Search,
  ShieldCheck,
  Sparkles,
} from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import { useState, useSyncExternalStore, type ReactNode } from "react"

import { useUnreadCount } from "@/components/notifications/notification-bell"
import { StatusBadge } from "@/components/report/status-badge"
import { Button } from "@/components/ui/button"
import { ButtonLink } from "@/components/ui/button-link"
import { NativeSelect } from "@/components/ui/native-select"
import { api, mediaUrl } from "@/lib/api/client"
import { call } from "@/lib/api/problem"
import type { ClaimSummary, LostReportSummary, MatchCandidate } from "@/lib/api/types"
import { formatDate } from "@/lib/format"
import { cn } from "@/lib/utils"

type Props = { name: string; reports: LostReportSummary[]; hasMoreReports: boolean; claims: ClaimSummary[] }
type Feature = "reporting" | "assistant" | "candidates" | "explanation" | "claim" | "history"

const subscribeHydration = () => () => undefined
const clientHydrated = () => true
const serverHydrated = () => false

function FeaturePanel({ feature, icon, tone, ai, children }: { feature: Feature; icon: ReactNode; tone: string; ai?: boolean; children: ReactNode }) {
  const t = useTranslations("ownerDashboard")
  return (
    <section
      id={`fitur-${feature}`}
      aria-labelledby={`judul-${feature}`}
      className={cn("owner-feature flex min-w-0 scroll-mt-24 flex-col rounded-2xl border p-4", tone)}
    >
      <div className="mb-2 flex items-start justify-between gap-2">
        <span className="owner-feature-icon flex size-9 items-center justify-center rounded-xl [&_svg]:size-5" aria-hidden>
          {icon}
        </span>
        {ai ? (
          <span className="inline-flex items-center gap-1 rounded-full border border-current/15 bg-white/80 px-2 py-1 text-[10px] font-bold tracking-wide">
            <Sparkles className="size-3" aria-hidden />
            AI Feature
          </span>
        ) : null}
      </div>
      <h2 id={`judul-${feature}`} className="text-[17px] leading-tight font-bold tracking-tight text-[#101f50]">
        {t(`features.${feature}.title`)}
      </h2>
      <p className="mt-1 text-xs font-semibold">{t(`features.${feature}.subtitle`)}</p>
      <p className="mt-2 text-[13px] leading-relaxed text-slate-600">{t(`features.${feature}.description`)}</p>
      <div className="mt-4 flex flex-1 flex-col justify-end gap-3">{children}</div>
    </section>
  )
}

export function OwnerDashboard({ name, reports, hasMoreReports, claims }: Props) {
  const t = useTranslations("ownerDashboard")
  const tm = useTranslations("matching")
  const [selectedCode, setSelectedCode] = useState("")
  // Cached queries can be ready before hydration on a return visit.
  const hydrated = useSyncExternalStore(subscribeHydration, clientHydrated, serverHydrated)
  const report =
    reports.find((r) => r.code === selectedCode) ??
    reports.find((r) => r.status === "potential_match" && r.candidate_count > 0) ??
    reports.find((r) => r.candidate_count > 0) ??
    reports[0]
  const matches = useQuery({
    queryKey: ["owner-dashboard", "matches", report?.code],
    queryFn: () => call(api.GET("/lost-reports/{id}/matches", { params: { path: { id: report?.code ?? "" } } })),
    enabled: Boolean(report),
    refetchInterval: 30_000,
    refetchOnWindowFocus: true,
    retry: false,
  })
  const unread = useUnreadCount()
  const candidate = hydrated ? matches.data?.items[0] : undefined
  const reportPath = report ? `/laporan/hilang/${report.code}` : undefined
  const candidatePath = candidate && reportPath ? `${reportPath}/kandidat/${candidate.id}` : undefined
  const relatedClaim = candidate ? claims.find((c) => c.found_code === candidate.found_report.code && c.lost_code === report?.code) : undefined
  const first = name.split(/\s+/)[0] ?? name
  const ready = claims.find((c) => c.status === "approved")
  const needsInfo = claims.find((c) => c.status === "info_requested")

  function candidateContent() {
    if (!report) return <p className="owner-state">{t("noReport")}</p>
    if (!hydrated || matches.isPending)
      return (
        <p className="owner-state flex items-center gap-2" role="status">
          <LoaderCircle className="size-4 animate-spin" aria-hidden />
          {t("loading")}
        </p>
      )
    if (matches.isError)
      return (
        <div className="owner-state">
          <p>{t("unavailable")}</p>
          <Button size="sm" variant="ghost" disabled={matches.isFetching} onClick={() => void matches.refetch()}>
            {t("retry")}
          </Button>
        </div>
      )
    if (!candidate || !candidatePath) return <p className="owner-state">{t("noCandidates")}</p>
    return <CandidatePreview candidate={candidate} href={candidatePath} />
  }

  return (
    <div className="owner-workspace">
      <header className="owner-hero relative isolate overflow-hidden rounded-3xl border border-blue-100 bg-[#f1f7ff] px-5 py-6 sm:px-8 sm:py-7">
        <img
          src="/illustrations/campus-entrance.webp"
          width={1280}
          height={720}
          alt=""
          aria-hidden
          className="owner-campus pointer-events-none absolute inset-y-0 right-0 -z-20 h-full w-[58%] object-cover object-center"
        />
        <div className="owner-hero-fade pointer-events-none absolute inset-0 -z-10" />
        <p className="mb-3 inline-flex items-center gap-2 rounded-full border border-blue-200 bg-white/90 px-3 py-1 text-xs font-bold text-blue-800">
          <span className="size-2 rounded-full bg-amber-400" aria-hidden />
          {t("eyebrow")}
        </p>
        <h1 className="max-w-2xl text-[clamp(1.8rem,3.5vw,3.1rem)] leading-[1.1] font-extrabold tracking-tight text-[#101f50]">
          {t("headingStart")} <span className="text-[#0759e8]">{t("headingAccent")}</span>
        </h1>
        <div className="mt-3 max-w-xl text-sm leading-relaxed text-slate-600">
          <h2 className="inline font-semibold">{t("greeting", { name: first })}</h2> <p className="inline">{t("lead")}</p>
        </div>
      </header>

      <div className="mt-5 grid min-w-0 gap-5 xl:grid-cols-[225px_minmax(0,1fr)]">
        <aside
          className="owner-companion relative hidden flex-col overflow-hidden rounded-3xl border border-blue-100 bg-[#eef5ff] xl:flex"
          aria-label={t("companion")}
        >
          <div className="relative z-10 px-5 pt-5">
            <span className="inline-flex items-center gap-1.5 rounded-full bg-white px-2.5 py-1 text-[11px] font-bold text-blue-800">
              <ShieldCheck className="size-3.5" aria-hidden />
              {t("owner")}
            </span>
            <p className="mt-4 text-xl leading-tight font-bold tracking-tight text-[#101f50]">{t("companionTitle")}</p>
            <p className="mt-2 text-xs leading-relaxed text-slate-600">{t("companionLead")}</p>
          </div>
          <div className="relative mt-3 min-h-48 flex-1">
            <div className="absolute inset-x-5 inset-y-8 rounded-full bg-blue-200/50 blur-2xl" aria-hidden />
            <img
              src="/illustrations/owner-student.webp"
              width={960}
              height={640}
              alt=""
              aria-hidden
              className="absolute inset-0 size-full object-cover object-[47%_center]"
            />
          </div>
          <div className="relative z-10 mx-3 mb-3 rounded-2xl border border-white bg-white/90 p-3.5">
            <p className="flex items-center gap-2 text-xs font-bold text-[#101f50]">
              <HandHeart className="size-4 text-blue-700" aria-hidden />
              {t("finderTitle")}
            </p>
            <p className="mt-1.5 text-xs leading-relaxed text-slate-600">{t("finderHint")}</p>
            <Link href="/lapor/temuan" className="mt-2 inline-flex min-h-9 items-center gap-1 text-xs font-bold text-blue-800 hover:underline">
              {t("finderAction")}
              <ArrowRight className="size-3.5" aria-hidden />
            </Link>
          </div>
        </aside>

        <div className="min-w-0">
          <div className="mb-4 flex flex-col gap-2 rounded-xl border bg-white p-3 sm:flex-row sm:items-center sm:gap-4">
            <label htmlFor="owner-report" className="shrink-0 text-xs font-bold text-slate-600">
              {t("selectedReport")}
            </label>
            {reports.length ? (
              <div className="min-w-0 flex-1">
                <NativeSelect
                  id="owner-report"
                  value={report?.code ?? ""}
                  onChange={(event) => setSelectedCode(event.target.value)}
                  className="h-10 border-0 bg-slate-50 text-sm font-medium"
                >
                  {reports.map((r) => (
                    <option key={r.code} value={r.code}>
                      {r.item_name} ({r.code})
                    </option>
                  ))}
                </NativeSelect>
              </div>
            ) : (
              <p className="flex-1 text-sm text-slate-500">{t("noReportShort")}</p>
            )}
            <Link href="/laporan" className="inline-flex min-h-9 shrink-0 items-center gap-1 text-xs font-semibold text-primary hover:underline">
              {t("allReports")}
              {hasMoreReports ? "+" : ""}
              <ChevronRight className="size-3.5" aria-hidden />
            </Link>
          </div>

          <div className="grid min-w-0 gap-3 md:grid-cols-3">
            <FeaturePanel feature="reporting" icon={<FilePlus2 />} tone="owner-amber">
              <div className="flex flex-wrap gap-1.5" aria-label={t("reportFields")}>
                {(["category", "place", "time", "photo"] as const).map((field) => (
                  <span key={field} className="rounded-md border border-amber-200/70 bg-white/70 px-2 py-1 text-[10px] font-semibold text-amber-900">
                    {t(`fields.${field}`)}
                  </span>
                ))}
              </div>
              <ButtonLink href="/lapor/hilang" className="w-full justify-between bg-[#101f50] text-white hover:bg-[#1a3278]">
                {t("reportAction")}
                <ArrowRight aria-hidden />
              </ButtonLink>
            </FeaturePanel>

            <FeaturePanel feature="assistant" icon={<Sparkles />} tone="owner-mint" ai>
              <div className="owner-state flex items-center gap-2">
                <ImagePlus className="size-5 shrink-0 text-emerald-700" aria-hidden />
                <span>{t("assistantHint")}</span>
              </div>
              <ButtonLink
                href="/lapor/hilang#smart-report-assistant"
                variant="outline"
                className="w-full justify-between border-emerald-200 bg-white/80 text-emerald-900 hover:bg-white"
              >
                {t("assistantAction")}
                <ArrowRight aria-hidden />
              </ButtonLink>
            </FeaturePanel>

            <FeaturePanel feature="candidates" icon={<Search />} tone="owner-sky" ai>
              {candidateContent()}
              <FeatureAction href={reportPath ? `${reportPath}#kandidat` : undefined} label={t("candidateAction")} disabledLabel={t("createFirst")} />
            </FeaturePanel>

            <FeaturePanel feature="explanation" icon={<ChartNoAxesColumnIncreasing />} tone="owner-violet" ai>
              {candidate ? (
                <ul className="grid grid-cols-2 gap-2" aria-label={t("signals")}>
                  {(["visual", "text", "location", "time"] as const).map((signal) => {
                    const line = candidate.explanation[signal]
                    const label = signal === "visual" && candidate.explanation.visual_mode === "cross_modal" ? t("crossModal") : t(`signalsShort.${signal}`)
                    return (
                      <li key={signal} className="rounded-lg border border-violet-200/70 bg-white/70 px-2 py-1.5">
                        <span className="block text-[10px] text-slate-600">{label}</span>
                        <span className="text-xs font-semibold text-violet-900">
                          {line.missing ? tm("level.missing") : line.level ? tm(`level.${line.level}`) : tm("level.missing")}
                        </span>
                      </li>
                    )
                  })}
                </ul>
              ) : (
                <p className="owner-state">{t("explanationHint")}</p>
              )}
              <FeatureAction href={candidatePath ? `${candidatePath}#alasan` : undefined} label={t("explanationAction")} disabledLabel={t("needCandidate")} />
            </FeaturePanel>

            <FeaturePanel feature="claim" icon={<ClipboardCheck />} tone="owner-mint">
              <p className="owner-state flex items-start gap-2">
                <ShieldCheck className="mt-0.5 size-4 shrink-0 text-emerald-700" aria-hidden />
                <span>{relatedClaim ? t("existingClaim", { code: relatedClaim.code }) : t("claimHint")}</span>
              </p>
              <div className="flex flex-col gap-2">
                <FeatureAction
                  href={relatedClaim ? `/klaim/${relatedClaim.code}` : candidate ? `/klaim/baru?pasangan=${candidate.id}` : undefined}
                  label={relatedClaim ? t("viewClaim") : t("claimAction")}
                  disabledLabel={t("needCandidate")}
                />
                <Link href="/klaim" className="text-center text-xs font-semibold underline-offset-4 hover:underline">
                  {t("myClaims")}
                </Link>
              </div>
            </FeaturePanel>

            <FeaturePanel feature="history" icon={<Bell />} tone="owner-sky">
              <div className="flex flex-wrap gap-2">
                <Link
                  href="/notifikasi"
                  className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-blue-200 bg-white/80 px-2.5 text-xs font-semibold text-blue-900 hover:bg-white"
                >
                  <Bell className="size-3.5" aria-hidden />
                  {!hydrated
                    ? t("notificationAction")
                    : unread.isError
                      ? t("viewNotifications")
                      : unread.data
                        ? t("unread", { count: unread.data.count })
                        : t("notificationAction")}
                </Link>
                {reportPath ? (
                  <Link
                    href={`${reportPath}#riwayat`}
                    className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-blue-200 bg-white/80 px-2.5 text-xs font-semibold text-blue-900 hover:bg-white"
                  >
                    <History className="size-3.5" aria-hidden />
                    {t("timelineAction")}
                  </Link>
                ) : null}
              </div>
              <ButtonLink href="/laporan" variant="outline" className="w-full justify-between border-blue-200 bg-white/80 text-blue-900 hover:bg-white">
                {t("historyAction")}
                <ArrowRight aria-hidden />
              </ButtonLink>
            </FeaturePanel>
          </div>
        </div>
      </div>

      <div className="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-blue-100 bg-blue-50/70 px-4 py-3 text-xs text-blue-950">
        <p className="flex items-center gap-2">
          <ShieldCheck className="size-4 shrink-0 text-blue-700" aria-hidden />
          {t("humanDecision")}
        </p>
        <Link href="/barang-temuan" className="inline-flex min-h-8 items-center gap-1 font-semibold hover:underline">
          {t("browse")}
          <ArrowRight className="size-3.5" aria-hidden />
        </Link>
      </div>

      {ready || needsInfo ? (
        <section aria-label={t("nextActions")} className="mt-5 grid gap-3 sm:grid-cols-2">
          {[ready, needsInfo]
            .filter((c): c is ClaimSummary => Boolean(c))
            .map((claim) => (
              <Link
                key={claim.id}
                href={`/klaim/${claim.code}`}
                className="flex min-w-0 items-center gap-3 rounded-xl border bg-white p-4 transition-colors hover:border-blue-300"
              >
                <span className="flex size-9 shrink-0 items-center justify-center rounded-full bg-blue-50 text-blue-700">
                  <Check className="size-4" aria-hidden />
                </span>
                <div className="min-w-0 flex-1">
                  <p className="text-xs font-semibold text-primary">{claim.status === "approved" ? t("handoverReady") : t("replyNeeded")}</p>
                  <p className="mt-1 truncate text-sm font-semibold">{claim.found_item_name}</p>
                </div>
                <ChevronRight className="size-4 shrink-0" aria-hidden />
              </Link>
            ))}
        </section>
      ) : null}

      {claims.length ? (
        <section aria-labelledby="owner-claims-title" className="mt-6">
          <div className="mb-3 flex items-center justify-between gap-3">
            <h2 id="owner-claims-title" className="font-semibold text-[#101f50]">
              {t("recentClaims")}
            </h2>
            <Link href="/klaim" className="text-xs font-semibold text-primary hover:underline">
              {t("myClaims")}
            </Link>
          </div>
          <ul className="divide-y rounded-xl border bg-white">
            {claims.slice(0, 3).map((claim) => (
              <li key={claim.id}>
                <Link href={`/klaim/${claim.code}`} className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 hover:bg-slate-50">
                  <div>
                    <p className="text-sm font-medium">{claim.found_item_name}</p>
                    <p className="mt-1 text-xs text-slate-500">
                      {claim.code} | {formatDate(claim.updated_at)}
                    </p>
                  </div>
                  <StatusBadge kind="claim" status={claim.status} />
                </Link>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  )
}

function FeatureAction({ href, label, disabledLabel }: { href?: string; label: string; disabledLabel: string }) {
  return href ? (
    <ButtonLink href={href} variant="outline" className="w-full justify-between border-current/15 bg-white/80 text-inherit hover:bg-white">
      {label}
      <ArrowRight aria-hidden />
    </ButtonLink>
  ) : (
    <div>
      <Button disabled variant="outline" className="w-full bg-white/60 text-slate-600 disabled:opacity-100">
        {label}
      </Button>
      <p className="mt-1.5 text-center text-[10px] text-slate-600">{disabledLabel}</p>
    </div>
  )
}

function CandidatePreview({ candidate, href }: { candidate: MatchCandidate; href: string }) {
  const photo = candidate.found_report.photos[0]
  const t = useTranslations("ownerDashboard")
  return (
    <Link href={href} className="flex min-w-0 items-center gap-2.5 rounded-xl border border-blue-200/70 bg-white/85 p-2 hover:bg-white">
      <span className="flex size-11 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-blue-50">
        {photo ? (
          <img
            src={mediaUrl(photo.media_id, photo.variant === "thumb_blur" ? "thumb_blur" : "thumb")}
            width={44}
            height={44}
            alt=""
            className="size-full object-cover"
          />
        ) : (
          <Search className="size-5" aria-hidden />
        )}
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-xs font-semibold text-slate-900">{candidate.found_report.item_name}</span>
        <span className="mt-1 flex items-center gap-1 text-[10px] text-slate-600">
          <MapPin className="size-3 shrink-0" aria-hidden />
          <span className="truncate">{candidate.found_report.found_location.name}</span>
        </span>
      </span>
      <span className="text-sm font-bold text-blue-800" aria-label={t("matchScore")}>
        {Math.round(candidate.overall * 100)}%
      </span>
    </Link>
  )
}
