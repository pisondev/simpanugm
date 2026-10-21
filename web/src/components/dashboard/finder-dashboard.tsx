"use client"

import { useQuery } from "@tanstack/react-query"
import {
  ArrowRight,
  Bell,
  ChevronRight,
  ClipboardPlus,
  Files,
  HandHeart,
  History,
  ImagePlus,
  LoaderCircle,
  MapPin,
  Search,
  ShieldCheck,
  Sparkles,
  Warehouse,
} from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import { useState, useSyncExternalStore, type ReactNode } from "react"

import { StatusBadge } from "@/components/report/status-badge"
import { Button } from "@/components/ui/button"
import { ButtonLink } from "@/components/ui/button-link"
import { NativeSelect } from "@/components/ui/native-select"
import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"
import type { FoundReportSummary } from "@/lib/api/types"
import { cn } from "@/lib/utils"

type Feature = "reporting" | "description" | "suggestions" | "duplicates" | "handover" | "history"
const subscribeHydration = () => () => undefined
const clientHydrated = () => true
const serverHydrated = () => false

function FeaturePanel({ feature, icon, tone, ai, children }: { feature: Feature; icon: ReactNode; tone: string; ai?: boolean; children: ReactNode }) {
  const t = useTranslations("finderDashboard")
  return (
    <section
      id={`finder-${feature}`}
      aria-labelledby={`finder-title-${feature}`}
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
      <h2 id={`finder-title-${feature}`} className="text-[17px] leading-tight font-bold tracking-tight text-[#101f50]">
        {t(`features.${feature}.title`)}
      </h2>
      <p className="mt-1 text-xs font-semibold">{t(`features.${feature}.subtitle`)}</p>
      <p className="mt-2 text-[13px] leading-relaxed text-slate-600">{t(`features.${feature}.description`)}</p>
      <div className="mt-4 flex flex-1 flex-col justify-end gap-3">{children}</div>
    </section>
  )
}

export function FinderDashboard({ name, reports }: { name: string; reports: FoundReportSummary[] }) {
  const t = useTranslations("finderDashboard")
  const [selectedCode, setSelectedCode] = useState("")
  const hydrated = useSyncExternalStore(subscribeHydration, clientHydrated, serverHydrated)
  const report = reports.find((r) => r.code === selectedCode) ?? reports.find((r) => r.custody_status === "with_finder") ?? reports[0]
  const hint = useQuery({
    queryKey: ["finder-dashboard", "hint", report?.code],
    queryFn: () => call(api.GET("/found-reports/{id}/match-hint", { params: { path: { id: report?.code ?? "" } } })),
    enabled: Boolean(report),
    retry: false,
    refetchInterval: 30_000,
    refetchOnWindowFocus: true,
  })
  const detail = useQuery({
    queryKey: ["finder-dashboard", "report", report?.code],
    queryFn: () => call(api.GET("/found-reports/{id}", { params: { path: { id: report?.code ?? "" } } })),
    enabled: Boolean(report),
    retry: false,
    refetchInterval: 30_000,
    refetchOnWindowFocus: true,
  })
  const details = hydrated ? detail.data : undefined
  const items = hydrated ? (hint.data?.items ?? []) : []
  const reportPath = report ? `/laporan/temuan/${report.code}` : undefined
  const post = details?.holding_post?.name ?? details?.intended_post?.name ?? report?.holding_post_name
  const first = name.split(/\s+/)[0] ?? name
  const custody = details?.custody_status ?? report?.custody_status

  function suggestions() {
    if (!report) return <p className="owner-state">{t("noReport")}</p>
    if (!hydrated || hint.isPending)
      return (
        <p role="status" className="owner-state flex items-center gap-2">
          <LoaderCircle className="size-4 animate-spin" aria-hidden />
          {t("loading")}
        </p>
      )
    if (hint.isError)
      return (
        <div className="owner-state" role="status">
          <p>{t("unavailable")}</p>
          <Button size="sm" variant="ghost" disabled={hint.isFetching} onClick={() => void hint.refetch()}>
            {t("retry")}
          </Button>
        </div>
      )
    if (!items.length) return <p className="owner-state">{t("noHints")}</p>
    return (
      <ul className="flex flex-col gap-2">
        {items.slice(0, 2).map((item) => (
          <li key={`${item.item_name}-${item.category_name}`} className="rounded-lg border border-blue-200/70 bg-white/80 p-2.5">
            <p className="text-xs font-semibold text-slate-900">{item.item_name}</p>
            <p className="mt-1 text-[11px] text-slate-600">{item.category_name}</p>
          </li>
        ))}
      </ul>
    )
  }

  return (
    <div className="finder-workspace">
      <header className="owner-hero relative isolate overflow-hidden rounded-3xl border border-blue-100 bg-[#f1f7ff] px-5 py-6 sm:px-8 sm:py-7">
        <img
          src="/illustrations/campus-entrance.webp"
          width={1280}
          height={720}
          alt=""
          aria-hidden
          className="pointer-events-none absolute inset-y-0 right-0 -z-20 h-full w-[58%] object-cover object-center"
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
          className="finder-companion relative hidden flex-col overflow-hidden rounded-3xl border border-blue-100 bg-[#eef5ff] xl:flex"
          aria-label={t("companion")}
        >
          <div className="relative z-10 px-5 pt-5">
            <span className="inline-flex items-center gap-1.5 rounded-full bg-white px-2.5 py-1 text-[11px] font-bold text-blue-800">
              <HandHeart className="size-3.5" aria-hidden />
              {t("finder")}
            </span>
            <p className="mt-4 text-xl leading-tight font-bold tracking-tight text-[#101f50]">{t("companionTitle")}</p>
            <p className="mt-2 text-xs leading-relaxed text-slate-600">{t("companionLead")}</p>
          </div>
          <div className="relative mt-3 min-h-48 flex-1">
            <div className="absolute inset-x-5 inset-y-8 rounded-full bg-blue-200/50 blur-2xl" aria-hidden />
            <img
              src="/illustrations/finder-student.webp"
              width={960}
              height={640}
              alt=""
              aria-hidden
              className="absolute inset-0 size-full object-cover object-[47%_center]"
            />
          </div>
          <div className="relative z-10 mx-3 mb-3 rounded-2xl border border-white bg-white/90 p-3.5">
            <p className="flex items-center gap-2 text-xs font-bold text-[#101f50]">
              <Search className="size-4 text-blue-700" aria-hidden />
              {t("ownerTitle")}
            </p>
            <p className="mt-1.5 text-xs leading-relaxed text-slate-600">{t("ownerHint")}</p>
            <Link href="/beranda?mode=pemilik" className="mt-2 inline-flex min-h-9 items-center gap-1 text-xs font-bold text-blue-800 hover:underline">
              {t("ownerAction")}
              <ArrowRight className="size-3.5" aria-hidden />
            </Link>
          </div>
        </aside>

        <div className="min-w-0">
          <div className="mb-4 flex flex-col gap-2 rounded-xl border bg-white p-3 sm:flex-row sm:items-center sm:gap-4">
            <label htmlFor="finder-report" className="shrink-0 text-xs font-bold text-slate-600">
              {t("selectedReport")}
            </label>
            {reports.length ? (
              <div className="min-w-0 flex-1">
                <NativeSelect
                  id="finder-report"
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
            <Link href="/laporan?tab=temuan" className="inline-flex min-h-9 shrink-0 items-center gap-1 text-xs font-semibold text-primary hover:underline">
              {t("allReports")}
              <ChevronRight className="size-3.5" aria-hidden />
            </Link>
          </div>

          <div className="grid min-w-0 gap-3 md:grid-cols-3">
            <FeaturePanel feature="reporting" icon={<ClipboardPlus />} tone="owner-amber">
              <div className="flex flex-wrap gap-1.5">
                {(["photo", "place", "time"] as const).map((field) => (
                  <span key={field} className="rounded-md border border-amber-200/70 bg-white/70 px-2 py-1 text-[10px] font-semibold text-amber-900">
                    {t(`fields.${field}`)}
                  </span>
                ))}
              </div>
              <ButtonLink href="/lapor/temuan" className="w-full justify-between bg-[#101f50] text-white hover:bg-[#1a3278]">
                {t("reportAction")}
                <ArrowRight aria-hidden />
              </ButtonLink>
            </FeaturePanel>
            <FeaturePanel feature="description" icon={<Sparkles />} tone="owner-mint" ai>
              <p className="owner-state flex items-start gap-2">
                <ImagePlus className="mt-0.5 size-4 shrink-0 text-emerald-700" aria-hidden />
                {t("descriptionHint")}
              </p>
              <ButtonLink
                href="/lapor/temuan#ai-assisted-description"
                variant="outline"
                className="w-full justify-between border-emerald-200 bg-white/80 text-emerald-900 hover:bg-white"
              >
                {t("descriptionAction")}
                <ArrowRight aria-hidden />
              </ButtonLink>
            </FeaturePanel>
            <FeaturePanel feature="suggestions" icon={<Search />} tone="owner-sky" ai>
              {suggestions()}
              <FeatureAction href={reportPath ? `${reportPath}#saran-pemilik` : undefined} label={t("suggestionAction")} />
            </FeaturePanel>
            <FeaturePanel feature="duplicates" icon={<Files />} tone="owner-violet" ai>
              <p className="owner-state">{t("duplicateHint")}</p>
              <ButtonLink
                href="/lapor/temuan?fitur=duplikat"
                variant="outline"
                className="w-full justify-between border-violet-200 bg-white/80 text-violet-900 hover:bg-white"
              >
                {t("duplicateAction")}
                <ArrowRight aria-hidden />
              </ButtonLink>
            </FeaturePanel>
            <FeaturePanel feature="handover" icon={<ShieldCheck />} tone="owner-sky">
              {report ? (
                <div className="flex flex-col gap-2 rounded-xl border border-blue-200/70 bg-white/80 p-2.5">
                  <StatusBadge kind="custody" status={custody ?? report.custody_status} />
                  <p className="text-xs font-semibold text-slate-900">{report.code}</p>
                  {post ? (
                    <p className="flex items-start gap-1 text-xs text-slate-600">
                      <MapPin className="mt-0.5 size-3 shrink-0" aria-hidden />
                      {post}
                    </p>
                  ) : null}
                  <p className="text-xs text-slate-600">
                    {custody === "with_finder" ? t("withFinder") : custody === "in_custody" ? t("inCustody") : t("custodyRecorded")}
                  </p>
                </div>
              ) : (
                <p className="owner-state">{t("noReport")}</p>
              )}
              <FeatureAction href={reportPath ? `${reportPath}#penyerahan` : undefined} label={t("handoverAction")} />
            </FeaturePanel>
            <FeaturePanel feature="history" icon={<Bell />} tone="owner-violet">
              <div className="flex flex-wrap gap-2">
                <Link
                  href="/notifikasi"
                  className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-violet-200 bg-white/80 px-2.5 text-xs font-semibold text-violet-900 hover:bg-white"
                >
                  <Bell className="size-3.5" aria-hidden />
                  {t("notifications")}
                </Link>
                {reportPath ? (
                  <Link
                    href={`${reportPath}#riwayat`}
                    className="inline-flex min-h-9 items-center gap-1.5 rounded-lg border border-violet-200 bg-white/80 px-2.5 text-xs font-semibold text-violet-900 hover:bg-white"
                  >
                    <History className="size-3.5" aria-hidden />
                    {t("timeline")}
                  </Link>
                ) : null}
              </div>
              <ButtonLink
                href="/laporan?tab=temuan"
                variant="outline"
                className="w-full justify-between border-violet-200 bg-white/80 text-violet-900 hover:bg-white"
              >
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
          {t("privacy")}
        </p>
        <Link href="/bantuan" className="inline-flex min-h-8 items-center gap-1 font-semibold hover:underline">
          <Warehouse className="size-3.5" aria-hidden />
          {t("postsAction")}
          <ArrowRight className="size-3.5" aria-hidden />
        </Link>
      </div>
    </div>
  )
}

function FeatureAction({ href, label }: { href?: string; label: string }) {
  const t = useTranslations("finderDashboard")
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
      <p className="mt-1.5 text-center text-[10px] text-slate-600">{t("createFirst")}</p>
    </div>
  )
}
