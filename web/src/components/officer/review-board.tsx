"use client"

import { ExternalLink, Lock, Mail, MapPin, Pin, PinOff, Search, Sparkles, Warehouse } from "lucide-react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import { DismissDialog } from "@/components/matching/dismiss-dialog"
import { RematchButton } from "@/components/matching/match-actions"
import { ScoreBars } from "@/components/matching/score-bars"
import { percent, TierBadge } from "@/components/matching/tier-badge"
import { CodeChip } from "@/components/report/code-chip"
import { PhotoStrip } from "@/components/report/photo-strip"
import { StatusBadge } from "@/components/report/status-badge"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { ButtonLink } from "@/components/ui/button-link"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { api, mediaUrl } from "@/lib/api/client"
import { call, messageFor } from "@/lib/api/problem"
import type { FoundReport, LostReport, MatchCandidate, OfficerBoard } from "@/lib/api/types"
import { formatDateTime, formatRange } from "@/lib/format"

export function ReviewBoardSearch({ initialType = "found", initialId = "" }: { initialType?: "lost" | "found"; initialId?: string }) {
  const t = useTranslations("officer")
  const router = useRouter()
  const [reportType, setReportType] = useState<"lost" | "found">(initialType)
  const [reportId, setReportId] = useState(initialId)

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const trimmed = reportId.trim()
    if (!trimmed) return
    router.push(`/petugas/papan?report_type=${reportType}&report_id=${encodeURIComponent(trimmed)}`)
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-3 sm:flex-row sm:items-center">
      <div className="w-full shrink-0 sm:w-44">
        <NativeSelect value={reportType} onChange={(e) => setReportType(e.target.value as "lost" | "found")} aria-label={t("reportTypeLabel")}>
          <option value="found">{t("boardTypeFound")}</option>
          <option value="lost">{t("boardTypeLost")}</option>
        </NativeSelect>
      </div>
      <div className="relative flex-1">
        <Input
          type="text"
          value={reportId}
          onChange={(e) => setReportId(e.target.value)}
          placeholder={t("boardCodePlaceholder")}
          aria-label={t("boardCodePlaceholder")}
          className="pr-10"
        />
        <Search className="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden />
      </div>
      <Button type="submit" disabled={!reportId.trim()}>
        {t("boardOpenAction")}
      </Button>
    </form>
  )
}

export function CandidateActionButtons({ candidate, isFoundReport }: { candidate: MatchCandidate; isFoundReport: boolean }) {
  const t = useTranslations("officer")
  const te = useTranslations("errors")
  const router = useRouter()
  const [pinned, setPinned] = useState(Boolean(candidate.pinned))
  const [isPinning, setIsPinning] = useState(false)
  const [isInviting, setIsInviting] = useState(false)
  const [invited, setInvited] = useState(false)

  async function handleTogglePin() {
    setIsPinning(true)
    try {
      const nextPinned = !pinned
      const res = await call(
        api.POST("/matches/{id}/pin", {
          params: { path: { id: candidate.id } },
          body: { pinned: nextPinned },
        }),
      )
      setPinned(res.pinned)
      toast.success(res.pinned ? t("boardPinSuccess") : t("boardUnpinSuccess"))
      router.refresh()
    } catch (err) {
      toast.error(messageFor(err, te))
    } finally {
      setIsPinning(false)
    }
  }

  async function handleInviteClaim() {
    setIsInviting(true)
    try {
      await call(
        api.POST("/matches/{id}/invite-claim", {
          params: { path: { id: candidate.id } },
        }),
      )
      setInvited(true)
      toast.success(t("boardInviteClaimSuccess"))
    } catch (err) {
      toast.error(messageFor(err, te))
    } finally {
      setIsInviting(false)
    }
  }

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button variant={pinned ? "default" : "outline"} size="sm" onClick={handleTogglePin} disabled={isPinning}>
        {pinned ? <PinOff className="size-3.5" aria-hidden /> : <Pin className="size-3.5" aria-hidden />}
        {pinned ? t("boardUnpin") : t("boardPin")}
      </Button>

      {isFoundReport && candidate.status === "active" ? (
        <Button variant="secondary" size="sm" onClick={handleInviteClaim} disabled={isInviting || invited}>
          <Mail className="size-3.5" aria-hidden />
          {invited ? t("boardInviteClaimAlready") : t("boardInviteClaim")}
        </Button>
      ) : null}

      {candidate.status === "active" ? <DismissDialog matchId={candidate.id} scope="officer" size="sm" /> : null}
    </div>
  )
}

function SelectedLostReportCard({ report }: { report: LostReport }) {
  const t = useTranslations("officer")
  const tr = useTranslations("report")
  const places = report.places.map((p) => p.location?.name ?? p.zone?.name).filter(Boolean)

  return (
    <Card className="border-border">
      <CardHeader className="space-y-3 pb-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex flex-wrap items-center gap-2">
            <CodeChip code={report.code} />
            <Badge variant="outline">{t("boardTypeLost")}</Badge>
            <StatusBadge kind="lost" status={report.status} />
          </div>
          <div className="flex flex-wrap items-center gap-1.5">
            <RematchButton type="lost" id={report.id} />
            <ButtonLink href="/petugas/duplikat" variant="outline" size="sm">
              {t("boardCheckDuplicates")}
            </ButtonLink>
            <ButtonLink href={`/petugas/laporan/hilang/${report.code}`} variant="ghost" size="sm">
              <ExternalLink className="size-3.5" aria-hidden />
              {t("boardFullReport")}
            </ButtonLink>
          </div>
        </div>
        <CardTitle className="text-xl font-semibold">{report.item_name}</CardTitle>
        <p className="text-sm text-muted-foreground">
          {report.category.name} | {report.category.parent_name}
        </p>
      </CardHeader>
      <CardContent className="space-y-4 text-sm">
        {report.photos && report.photos.length > 0 ? (
          <div className="space-y-1">
            <p className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">{tr("photos")}</p>
            <PhotoStrip photos={report.photos} alt={report.item_name} />
          </div>
        ) : null}

        <div className="space-y-1">
          <p className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">{tr("description")}</p>
          <p className="whitespace-pre-line text-foreground">{report.description}</p>
        </div>

        {report.private_details ? (
          <div className="space-y-1 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3">
            <div className="flex items-center gap-1.5 font-semibold text-amber-700 dark:text-amber-400">
              <Lock className="size-4" aria-hidden />
              <span>{t("boardPrivateDetails")}</span>
            </div>
            <p className="text-foreground">{report.private_details}</p>
          </div>
        ) : null}

        <div className="grid grid-cols-1 gap-3 border-t pt-2 sm:grid-cols-2">
          <div>
            <span className="text-xs text-muted-foreground">{tr("lostWindow")}</span>
            <p className="font-medium text-foreground">{formatRange(report.lost_from, report.lost_until)}</p>
          </div>
          <div>
            <span className="text-xs text-muted-foreground">{tr("places")}</span>
            <p className="font-medium text-foreground">{places.length > 0 ? places.join(", ") : "-"}</p>
          </div>
          {report.colours && report.colours.length > 0 ? (
            <div>
              <span className="text-xs text-muted-foreground">{tr("colours")}</span>
              <p className="font-medium text-foreground">{report.colours.join(", ")}</p>
            </div>
          ) : null}
          {report.brand ? (
            <div>
              <span className="text-xs text-muted-foreground">{tr("brand")}</span>
              <p className="font-medium text-foreground">{report.brand}</p>
            </div>
          ) : null}
        </div>
      </CardContent>
    </Card>
  )
}

function SelectedFoundReportCard({ report }: { report: FoundReport }) {
  const t = useTranslations("officer")
  const tr = useTranslations("report")
  const tm = useTranslations("matching")

  return (
    <Card className="border-border">
      <CardHeader className="space-y-3 pb-3">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div className="flex flex-wrap items-center gap-2">
            <CodeChip code={report.code} />
            <Badge variant="outline">{t("boardTypeFound")}</Badge>
            <StatusBadge kind="custody" status={report.custody_status} />
            <StatusBadge kind="found" status={report.status} />
          </div>
          <div className="flex flex-wrap items-center gap-1.5">
            <RematchButton type="found" id={report.id} />
            <ButtonLink href="/petugas/duplikat" variant="outline" size="sm">
              {t("boardCheckDuplicates")}
            </ButtonLink>
            <ButtonLink href={`/petugas/laporan/temuan/${report.code}`} variant="ghost" size="sm">
              <ExternalLink className="size-3.5" aria-hidden />
              {t("boardFullReport")}
            </ButtonLink>
          </div>
        </div>
        <CardTitle className="text-xl font-semibold">{report.item_name}</CardTitle>
        <p className="text-sm text-muted-foreground">
          {report.category.name} | {report.category.parent_name}
        </p>
      </CardHeader>
      <CardContent className="space-y-4 text-sm">
        {report.photos && report.photos.length > 0 ? (
          <div className="space-y-1">
            <p className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">{tr("photos")}</p>
            <PhotoStrip photos={report.photos} alt={report.item_name} />
          </div>
        ) : null}

        <div className="space-y-1">
          <p className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">{tr("description")}</p>
          <p className="whitespace-pre-line text-foreground">{report.description}</p>
        </div>

        {report.hidden_details ? (
          <div className="space-y-1 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3">
            <div className="flex items-center gap-1.5 font-semibold text-amber-700 dark:text-amber-400">
              <Lock className="size-4" aria-hidden />
              <span>{t("boardHiddenDetails")}</span>
            </div>
            <p className="text-foreground">{report.hidden_details}</p>
          </div>
        ) : null}

        <div className="grid grid-cols-1 gap-3 border-t pt-2 sm:grid-cols-2">
          <div>
            <span className="text-xs text-muted-foreground">{tr("foundAt")}</span>
            <p className="font-medium text-foreground">{formatDateTime(report.found_at)}</p>
          </div>
          <div>
            <span className="text-xs text-muted-foreground">{tr("foundLocation")}</span>
            <p className="font-medium text-foreground">{report.found_location.name}</p>
          </div>
          <div>
            <span className="text-xs text-muted-foreground">{tr("custody")}</span>
            <p className="font-medium text-foreground">{report.holding_post ? report.holding_post.name : tm("withFinder")}</p>
          </div>
          {report.colours && report.colours.length > 0 ? (
            <div>
              <span className="text-xs text-muted-foreground">{tr("colours")}</span>
              <p className="font-medium text-foreground">{report.colours.join(", ")}</p>
            </div>
          ) : null}
          {report.brand ? (
            <div>
              <span className="text-xs text-muted-foreground">{tr("brand")}</span>
              <p className="font-medium text-foreground">{report.brand}</p>
            </div>
          ) : null}
        </div>
      </CardContent>
    </Card>
  )
}

function CandidateCard({ candidate, reportType }: { candidate: MatchCandidate; reportType: "lost" | "found" }) {
  const t = useTranslations("officer")
  const tr = useTranslations("report")
  const tm = useTranslations("matching")
  const tmo = useTranslations("matching.officer")
  const isFoundReport = reportType === "found"

  const code = isFoundReport ? candidate.lost_report.code : candidate.found_report.code
  const itemName = isFoundReport ? candidate.lost_report.item_name : candidate.found_report.item_name
  const category = isFoundReport ? candidate.lost_report.category : candidate.found_report.category
  const counterpartHref = isFoundReport ? `/petugas/laporan/hilang/${candidate.lost_report.code}` : `/petugas/laporan/temuan/${candidate.found_report.code}`

  const foundCandidate = !isFoundReport ? candidate.found_report : null
  const lostCandidate = isFoundReport ? candidate.lost_report : null
  const photo = foundCandidate && foundCandidate.photos.length > 0 ? foundCandidate.photos[0] : null

  return (
    <Card className={`transition-colors ${candidate.pinned ? "border-primary/60 bg-primary-soft/10 shadow-xs" : "border-border bg-card"}`}>
      <CardHeader className="space-y-3 pb-3">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-wrap items-center gap-2">
            <TierBadge tier={candidate.tier} />
            <span className="text-base font-bold text-foreground tabular-nums">{percent(candidate.overall)}</span>
            {candidate.pinned ? (
              <Badge variant="default" className="gap-1">
                <Pin className="size-3 fill-current" aria-hidden />
                {t("boardPinnedBadge")}
              </Badge>
            ) : null}
            {candidate.flags?.includes("name_hit") ? <Badge variant="secondary">{tmo("nameHit")}</Badge> : null}
          </div>

          <CandidateActionButtons candidate={candidate} isFoundReport={isFoundReport} />
        </div>

        <div className="flex items-start justify-between gap-3 pt-1">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <CodeChip code={code} />
              <Link href={counterpartHref} className="font-semibold text-foreground hover:underline">
                {itemName}
              </Link>
            </div>
            <p className="text-xs text-muted-foreground">
              {category.name} | {category.parent_name}
            </p>
          </div>

          <ButtonLink href={counterpartHref} variant="ghost" size="sm">
            <ExternalLink className="size-3.5" aria-hidden />
            {t("boardFullReport")}
          </ButtonLink>
        </div>
      </CardHeader>

      <CardContent className="space-y-4 text-sm">
        <div className="grid grid-cols-1 items-start gap-3 sm:grid-cols-12">
          {photo ? (
            <div className="overflow-hidden rounded-lg bg-muted sm:col-span-4">
              <img src={mediaUrl(photo.media_id, "thumb")} alt={itemName} className="aspect-square w-full object-cover" loading="lazy" />
            </div>
          ) : null}

          <div className={photo ? "space-y-2 sm:col-span-8" : "space-y-2 sm:col-span-12"}>
            {foundCandidate ? <p className="line-clamp-3 text-xs whitespace-pre-line text-foreground sm:text-sm">{foundCandidate.description}</p> : null}

            {foundCandidate && foundCandidate.hidden_details ? (
              <div className="space-y-0.5 rounded-md border border-amber-500/30 bg-amber-500/10 p-2 text-xs">
                <div className="flex items-center gap-1 font-semibold text-amber-700 dark:text-amber-400">
                  <Lock className="size-3" aria-hidden />
                  <span>{t("boardHiddenDetails")}</span>
                </div>
                <p className="text-foreground">{foundCandidate.hidden_details}</p>
              </div>
            ) : null}

            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 pt-1 text-xs text-muted-foreground">
              {lostCandidate ? (
                <>
                  {lostCandidate.owner_name ? (
                    <span>
                      {tmo("owner")}: {lostCandidate.owner_name}
                    </span>
                  ) : null}
                  <span>
                    {tr("lostWindow")}: {formatRange(lostCandidate.lost_from, lostCandidate.lost_until)}
                  </span>
                </>
              ) : null}

              {foundCandidate ? (
                <>
                  <span className="flex items-center gap-1">
                    <MapPin className="size-3" aria-hidden />
                    {foundCandidate.found_location.name}
                  </span>
                  <span>{formatDateTime(foundCandidate.found_at)}</span>
                  <span className="flex items-center gap-1">
                    <Warehouse className="size-3" aria-hidden />
                    {foundCandidate.holding_post ? foundCandidate.holding_post.name : tm("withFinder")}
                  </span>
                </>
              ) : null}
            </div>
          </div>
        </div>

        <div className="border-t pt-3">
          <ScoreBars explanation={candidate.explanation} scores={candidate.scores} />
        </div>
      </CardContent>
    </Card>
  )
}

export function ReviewBoardView({ board, currentType, currentId }: { board: OfficerBoard; currentType: "lost" | "found"; currentId: string }) {
  const t = useTranslations("officer")

  return (
    <div className="space-y-6">
      <ReviewBoardSearch initialType={currentType} initialId={currentId} />

      <div className="grid grid-cols-1 items-start gap-6 lg:grid-cols-12">
        <div className="space-y-4 lg:col-span-5">
          <h2 className="text-base font-semibold tracking-wider text-muted-foreground uppercase">
            {board.report_type === "lost" ? t("boardTypeLost") : t("boardTypeFound")}
          </h2>

          {board.report_type === "lost" && board.lost_report ? <SelectedLostReportCard report={board.lost_report} /> : null}

          {board.report_type === "found" && board.found_report ? <SelectedFoundReportCard report={board.found_report} /> : null}
        </div>

        <div className="space-y-4 lg:col-span-7">
          <div className="flex items-center justify-between">
            <h2 className="text-base font-semibold tracking-wider text-muted-foreground uppercase">
              {t("boardCandidatesCount", { count: board.candidates.length })}
            </h2>
          </div>

          {board.candidates.length === 0 ? (
            <Card className="border-dashed">
              <CardContent className="space-y-2 py-12 text-center text-muted-foreground">
                <Sparkles className="mx-auto size-8 text-muted-foreground/60" aria-hidden />
                <p className="font-medium">{t("boardNoCandidates")}</p>
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-4">
              {board.candidates.map((candidate) => (
                <CandidateCard key={candidate.id} candidate={candidate} reportType={board.report_type} />
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
