import { Activity, ArrowRight, ChartNoAxesCombined, MapPin, Network, RotateCcw, Settings2, ShieldCheck, Sparkles, TrendingUp } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import type { ReactNode } from "react"

import { ButtonLink } from "@/components/ui/button-link"
import type { AiClustersResponse, AiMonitoringResponse, AnalyticsSummary } from "@/lib/api/types"
import { cn } from "@/lib/utils"

import type { AdminFeature } from "./admin-feature-label"

type Props = {
  name: string
  uncovered: number
  monitoring: AiMonitoringResponse | null
  clusters: AiClustersResponse | null
  analytics: AnalyticsSummary | null
}

function FeaturePanel({
  feature,
  tone,
  icon,
  ai = true,
  children,
}: {
  feature: AdminFeature
  tone: string
  icon: ReactNode
  ai?: boolean
  children: ReactNode
}) {
  const t = useTranslations("adminDashboard")
  return (
    <section
      id={`admin-${feature}`}
      aria-labelledby={`admin-title-${feature}`}
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
      <h2 id={`admin-title-${feature}`} className="text-[17px] leading-tight font-bold tracking-tight text-[#101f50]">
        {t(`features.${feature}.title`)}
      </h2>
      <p className="mt-1 text-xs font-semibold">{t(`features.${feature}.subtitle`)}</p>
      <p className="mt-2 text-[13px] leading-relaxed text-slate-600">{t(`features.${feature}.description`)}</p>
      <div className="mt-4 flex flex-1 flex-col justify-end gap-3">{children}</div>
    </section>
  )
}

export function AdminDashboard({ name, uncovered, monitoring, clusters, analytics }: Props) {
  const t = useTranslations("adminDashboard")
  const fusion = monitoring?.active_versions.find((version) => version.component === "fusion")
  const ranks = monitoring?.confirmed_ranks
  const run = clusters?.run

  return (
    <div className="admin-workspace mb-7">
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
        <p className="mt-3 max-w-xl text-sm leading-relaxed text-slate-600">
          <strong>{t("greeting", { name: name.split(/\s+/)[0] ?? name })}</strong> {t("lead")}
        </p>
      </header>

      <div className="mt-5 grid min-w-0 gap-5 xl:grid-cols-[225px_minmax(0,1fr)]">
        <aside
          className="admin-companion relative hidden flex-col overflow-hidden rounded-3xl border border-blue-100 bg-[#eef5ff] xl:flex"
          aria-label={t("companion")}
        >
          <div className="relative z-10 px-5 pt-5">
            <span className="inline-flex items-center gap-1.5 rounded-full bg-white px-2.5 py-1 text-[11px] font-bold text-blue-800">
              <Settings2 className="size-3.5" aria-hidden />
              {t("role")}
            </span>
            <p className="mt-4 text-xl leading-tight font-bold tracking-tight text-[#101f50]">{t("companionTitle")}</p>
            <p className="mt-2 text-xs leading-relaxed text-slate-600">{t("companionLead")}</p>
          </div>
          <div className="relative mt-3 min-h-48 flex-1">
            <div className="absolute inset-x-5 inset-y-8 rounded-full bg-blue-200/50 blur-2xl" aria-hidden />
            <img
              src="/illustrations/admin-student.webp"
              width={960}
              height={640}
              alt=""
              aria-hidden
              className="absolute inset-0 size-full object-cover object-[42%_center]"
            />
          </div>
          <div className="relative z-10 mx-3 mb-3 rounded-2xl border border-white bg-white/90 p-3.5">
            <p className="text-xs font-bold text-[#101f50]">{t("coverageTitle")}</p>
            <p className="mt-1.5 text-xs leading-relaxed text-slate-600">{t("coverageHint", { count: uncovered })}</p>
            <Link href="#cakupan-pos" className="mt-2 inline-flex min-h-9 items-center gap-1 text-xs font-bold text-blue-800 hover:underline">
              {t("coverageAction")}
              <ArrowRight className="size-3.5" aria-hidden />
            </Link>
          </div>
        </aside>

        <div className="grid min-w-0 gap-3 md:grid-cols-3">
          <FeaturePanel feature="monitoring" tone="owner-amber" icon={<Activity />}>
            <div className="owner-state">
              {ranks ? (
                <>
                  <p className="font-semibold">{t("confirmedCount", { count: ranks.confirmed_total })}</p>
                  <p className="mt-1">{ranks.confirmed_total > 0 ? t("recall", { value: (ranks.recall_at_1 * 100).toFixed(1) }) : t("noConfirmed")}</p>
                </>
              ) : (
                <p>{t("unavailable")}</p>
              )}
            </div>
            <FeatureAction href="/admin/ai/monitoring" label={t("monitoringAction")} />
          </FeaturePanel>
          <FeaturePanel feature="drift" tone="owner-mint" icon={<TrendingUp />}>
            <div className="owner-state">
              {monitoring ? (
                <>
                  <p className="font-semibold">{t("psi", { value: monitoring.score_psi.toFixed(3) })}</p>
                  <p className="mt-1">{monitoring.psi_warning ? t("driftWarning") : t("noDriftWarning")}</p>
                </>
              ) : (
                <p>{t("unavailable")}</p>
              )}
            </div>
            <FeatureAction href="/admin/ai/monitoring#drift-detection" label={t("driftAction")} />
          </FeaturePanel>
          <FeaturePanel feature="fusion" tone="owner-sky" icon={<RotateCcw />}>
            <p className="owner-state break-words">{monitoring ? (fusion ? t("activeFusion", { name: fusion.name }) : t("noFusion")) : t("unavailable")}</p>
            <FeatureAction href="/admin/ai/monitoring#fusion-control" label={t("fusionAction")} />
          </FeaturePanel>
          <FeaturePanel feature="clusters" tone="owner-violet" icon={<Network />}>
            <p className="owner-state">
              {clusters ? (run ? t("clusterRun", { status: t(`clusterStatus.${run.status}`), count: run.cluster_count }) : t("noClusters")) : t("unavailable")}
            </p>
            <FeatureAction href="/admin/ai/klaster" label={t("clustersAction")} />
          </FeaturePanel>
          <FeaturePanel feature="hotspots" tone="owner-amber" icon={<MapPin />}>
            <p className="owner-state">{t("hotspotsHint")}</p>
            <FeatureAction href="/admin/peta-rawan" label={t("hotspotsAction")} />
          </FeaturePanel>
          <FeaturePanel feature="trends" tone="owner-sky" icon={<ChartNoAxesCombined />} ai={false}>
            <div className="owner-state">
              {analytics ? (
                <>
                  <p className="font-semibold">{t("reportCounts", { lost: analytics.total_lost, found: analytics.total_found })}</p>
                  <p className="mt-1">{t("returnedCount", { count: analytics.total_returned })}</p>
                </>
              ) : (
                <p>{t("unavailable")}</p>
              )}
            </div>
            <FeatureAction href="/admin/tren" label={t("trendsAction")} />
          </FeaturePanel>
        </div>
      </div>
      <div className="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-blue-100 bg-blue-50/70 px-4 py-3 text-xs text-blue-950">
        <p className="flex items-center gap-2">
          <ShieldCheck className="size-4 shrink-0 text-blue-700" aria-hidden />
          {t("metricsNote")}
        </p>
        <Link href="/admin/audit" className="inline-flex min-h-8 items-center gap-1 font-semibold hover:underline">
          {t("auditAction")}
          <ArrowRight className="size-3.5" aria-hidden />
        </Link>
      </div>
    </div>
  )
}

function FeatureAction({ href, label }: { href: string; label: string }) {
  return (
    <ButtonLink href={href} variant="outline" className="w-full justify-between border-current/15 bg-white/80 text-inherit hover:bg-white">
      {label}
      <ArrowRight aria-hidden />
    </ButtonLink>
  )
}
