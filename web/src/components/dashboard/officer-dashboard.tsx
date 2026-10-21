import { ArrowRight, ClipboardCheck, Crown, Files, Handshake, MapPin, PackageOpen, Search, ShieldCheck, Sparkles } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import type { ReactNode } from "react"

import { StatusBadge } from "@/components/report/status-badge"
import { Button } from "@/components/ui/button"
import { ButtonLink } from "@/components/ui/button-link"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import type { ClaimSummary, OfficerQueueItem, OfficerSummary, Post } from "@/lib/api/types"
import { cn } from "@/lib/utils"

type Feature = "priority" | "board" | "checklist" | "search" | "duplicates" | "handover"
type Props = {
  name: string
  posts: Post[]
  scope: "posts" | "campus"
  queue: OfficerQueueItem[]
  total: number
  summary: OfficerSummary
  approved: ClaimSummary[]
  selectedCode?: string
  typeFilter?: string
}

function FeaturePanel({ feature, tone, icon, ai = true, children }: { feature: Feature; tone: string; icon: ReactNode; ai?: boolean; children: ReactNode }) {
  const t = useTranslations("officerDashboard")
  return (
    <section
      id={`officer-${feature}`}
      aria-labelledby={`officer-title-${feature}`}
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
      <h2 id={`officer-title-${feature}`} className="text-[17px] leading-tight font-bold tracking-tight text-[#101f50]">
        {t(`features.${feature}.title`)}
      </h2>
      <p className="mt-1 text-xs font-semibold">{t(`features.${feature}.subtitle`)}</p>
      <p className="mt-2 text-[13px] leading-relaxed text-slate-600">{t(`features.${feature}.description`)}</p>
      <div className="mt-4 flex flex-1 flex-col justify-end gap-3">{children}</div>
    </section>
  )
}

export function OfficerDashboard({ name, posts, scope, queue, total, summary, approved, selectedCode, typeFilter }: Props) {
  const t = useTranslations("officerDashboard")
  const to = useTranslations("officer")
  const selected =
    queue.find((item) => item.found_code === selectedCode && item.claim_code) ??
    queue.find((item) => item.found_code === selectedCode) ??
    queue.find((item) => item.claim_code) ??
    queue[0]
  const cases = queue.filter((item, index) => queue.findIndex((other) => other.found_code === item.found_code) === index)
  const first = queue[0]
  const ready = approved.find((claim) => claim.custody_status === "in_custody")
  const board = selected ? `/petugas/papan?report_type=found&report_id=${encodeURIComponent(selected.found_id)}` : "/petugas/papan"
  const checklist = selected?.claim_code ? `/petugas/klaim/${encodeURIComponent(selected.claim_code)}#verification-checklist` : undefined
  const handover = ready ? `/petugas/klaim/${encodeURIComponent(ready.code)}#handover-to-owner` : "/petugas/klaim?status=approved"

  return (
    <div className="officer-workspace mb-7">
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

      <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
        <nav aria-label={t("scopeLabel")} className="inline-flex rounded-full border bg-white p-1 text-xs font-semibold">
          {(["posts", "campus"] as const).map((value) => (
            <Link
              key={value}
              href={value === "posts" ? "/petugas" : "/petugas?scope=campus"}
              aria-current={scope === value ? "page" : undefined}
              className={cn(
                "inline-flex min-h-9 items-center rounded-full px-4 text-muted-foreground hover:text-primary",
                scope === value && "bg-blue-50 text-primary",
              )}
            >
              {value === "posts" ? to("scopePosts") : to("scopeCampus")}
            </Link>
          ))}
        </nav>
        <p className="flex min-w-0 items-center gap-1.5 text-xs text-slate-600">
          <MapPin className="size-3.5 shrink-0" aria-hidden />
          {scope === "campus" ? t("campusOverview") : posts.map((post) => post.name).join(", ")}
        </p>
      </div>

      <div className="mt-4 grid min-w-0 gap-5 xl:grid-cols-[225px_minmax(0,1fr)]">
        <aside
          className="officer-companion relative hidden flex-col overflow-hidden rounded-3xl border border-blue-100 bg-[#eef5ff] xl:flex"
          aria-label={t("companion")}
        >
          <div className="relative z-10 px-5 pt-5">
            <span className="inline-flex items-center gap-1.5 rounded-full bg-white px-2.5 py-1 text-[11px] font-bold text-blue-800">
              <ShieldCheck className="size-3.5" aria-hidden />
              {t("role")}
            </span>
            <p className="mt-4 text-xl leading-tight font-bold tracking-tight text-[#101f50]">{t("companionTitle")}</p>
            <p className="mt-2 text-xs leading-relaxed text-slate-600">{t("companionLead")}</p>
          </div>
          <div className="relative mt-3 min-h-48 flex-1">
            <div className="absolute inset-x-5 inset-y-8 rounded-full bg-blue-200/50 blur-2xl" aria-hidden />
            <img
              src="/illustrations/officer-student.webp"
              width={960}
              height={640}
              alt=""
              aria-hidden
              className="absolute inset-0 size-full object-cover object-[46%_center]"
            />
          </div>
          <div className="relative z-10 mx-3 mb-3 rounded-2xl border border-white bg-white/90 p-3.5">
            <p className="text-xs font-bold text-[#101f50]">{t("receiptTitle")}</p>
            <p className="mt-1.5 text-xs leading-relaxed text-slate-600">{t("receiptHint", { count: summary.with_finder })}</p>
            <Link href="/petugas/terima" className="mt-2 inline-flex min-h-9 items-center gap-1 text-xs font-bold text-blue-800 hover:underline">
              <PackageOpen className="size-3.5" aria-hidden />
              {t("receiveAction")}
              <ArrowRight className="size-3.5" aria-hidden />
            </Link>
          </div>
        </aside>

        <div className="min-w-0">
          <form method="get" className="mb-4 flex flex-col gap-2 rounded-xl border bg-white p-3 sm:flex-row sm:items-center sm:gap-3">
            <input type="hidden" name="scope" value={scope} />
            {typeFilter ? <input type="hidden" name="type" value={typeFilter} /> : null}
            <label htmlFor="officer-case" className="shrink-0 text-xs font-bold text-slate-600">
              {t("selectedCase")}
            </label>
            {cases.length ? (
              <>
                <div className="min-w-0 flex-1">
                  <NativeSelect id="officer-case" name="kasus" defaultValue={selected?.found_code} className="h-10 border-0 bg-slate-50 text-sm font-medium">
                    {cases.map((item) => (
                      <option key={item.found_code} value={item.found_code}>
                        {item.title} ({item.found_code})
                      </option>
                    ))}
                  </NativeSelect>
                </div>
                <Button type="submit" variant="outline" size="sm">
                  {t("selectAction")}
                </Button>
              </>
            ) : (
              <p className="text-sm text-slate-500">{t("noCases")}</p>
            )}
          </form>

          <div className="grid min-w-0 gap-3 md:grid-cols-3">
            <FeaturePanel feature="priority" tone="owner-amber" icon={<Crown />}>
              <div className="owner-state">
                <p className="font-semibold">{t("queueCount", { count: total })}</p>
                {first ? (
                  <>
                    <p className="mt-1 text-xs font-medium text-slate-900">{first.title}</p>
                    <p className="mt-1 text-xs">{to("scoreLabel", { score: Math.round(first.priority_score * 100) })}</p>
                  </>
                ) : (
                  <p className="mt-1">{t("noCases")}</p>
                )}
              </div>
              <FeatureAction href="#antrean" label={t("priorityAction")} />
            </FeaturePanel>
            <FeaturePanel feature="board" tone="owner-mint" icon={<Search />}>
              <p className="owner-state">{selected ? `${selected.found_code} | ${selected.title}` : t("boardHint")}</p>
              <FeatureAction href={board} label={t("boardAction")} />
            </FeaturePanel>
            <FeaturePanel feature="checklist" tone="owner-sky" icon={<ClipboardCheck />}>
              <p className="owner-state">{selected?.claim_code ? t("claimSelected", { code: selected.claim_code }) : t("checklistHint")}</p>
              {checklist ? (
                <FeatureAction href={checklist} label={t("checklistAction")} />
              ) : (
                <>
                  <Button disabled variant="outline" className="w-full bg-white/60 text-slate-600 disabled:opacity-100">
                    {t("checklistAction")}
                  </Button>
                  <Link href="/petugas/klaim" className="text-center text-xs font-semibold hover:underline">
                    {t("claimsAction")}
                  </Link>
                </>
              )}
            </FeaturePanel>
            <FeaturePanel feature="search" tone="owner-violet" icon={<Search />}>
              <form action="/petugas/cari" method="get" className="flex flex-col gap-3">
                <input type="hidden" name="semantic" value="true" />
                <label htmlFor="officer-semantic" className="text-xs font-semibold">
                  {t("searchLabel")}
                </label>
                <Input id="officer-semantic" name="q" placeholder={t("searchPlaceholder")} maxLength={200} required className="bg-white/90" />
                <Button type="submit" variant="outline" className="w-full justify-between border-violet-200 bg-white/80 text-violet-900 hover:bg-white">
                  {t("searchAction")}
                  <ArrowRight aria-hidden />
                </Button>
              </form>
            </FeaturePanel>
            <FeaturePanel feature="duplicates" tone="owner-amber" icon={<Files />}>
              <p className="owner-state">{t("duplicatesHint")}</p>
              <FeatureAction href="/petugas/duplikat" label={t("duplicatesAction")} />
            </FeaturePanel>
            <FeaturePanel feature="handover" tone="owner-sky" icon={<Handshake />} ai={false}>
              {ready ? (
                <div className="rounded-xl border border-blue-200/70 bg-white/80 p-2.5">
                  <StatusBadge kind="claim" status={ready.status} />
                  <p className="mt-2 text-xs font-semibold text-slate-900">{ready.found_item_name}</p>
                  <p className="mt-1 text-xs text-slate-600">{ready.code}</p>
                </div>
              ) : (
                <p className="owner-state">{t("noApproved")}</p>
              )}
              <FeatureAction href={handover} label={ready ? t("handoverAction") : t("approvedAction")} />
            </FeaturePanel>
          </div>
        </div>
      </div>
      <div className="mt-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-blue-100 bg-blue-50/70 px-4 py-3 text-xs text-blue-950">
        <p className="flex items-center gap-2">
          <ShieldCheck className="size-4 shrink-0 text-blue-700" aria-hidden />
          {t("humanDecision")}
        </p>
        <Link href="/petugas/klaim" className="inline-flex min-h-8 items-center gap-1 font-semibold hover:underline">
          {t("claimsAction")}
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
