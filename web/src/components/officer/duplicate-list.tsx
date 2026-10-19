"use client"

import { Check, CopyCheck, GitMerge, X } from "lucide-react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import { EmptyState } from "@/components/layout/empty-state"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { api, mediaUrl } from "@/lib/api/client"
import type { OfficerDuplicateItem } from "@/lib/api/types"
import { formatDateTime } from "@/lib/format"

interface Props {
  initialItems: OfficerDuplicateItem[]
  currentStatus: string
}

export function DuplicateList({ initialItems, currentStatus }: Props) {
  const t = useTranslations("officer")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()

  const [items, setItems] = useState<OfficerDuplicateItem[]>(initialItems)
  const [survivorSelection, setSurvivorSelection] = useState<Record<string, string>>({})
  const [busyId, setBusyId] = useState<string | null>(null)

  // Manual merge form state
  const [showManual, setShowManual] = useState(false)
  const [manualType, setManualType] = useState<"found" | "lost">("found")
  const [manualSurvivor, setManualSurvivor] = useState("")
  const [manualDuplicate, setManualDuplicate] = useState("")
  const [manualSubmitting, setManualSubmitting] = useState(false)

  const handleConfirm = async (item: OfficerDuplicateItem) => {
    const survivorId = survivorSelection[item.id] ?? item.report_a.id
    setBusyId(item.id)
    try {
      const res = await api.POST("/officer/duplicates/{id}/confirm", {
        params: { path: { id: item.id } },
        body: { survivor_id: survivorId },
      })
      if (res.error) {
        toast.error(te("generic"))
        return
      }
      toast.success(t("mergeSuccess"))
      setItems((prev) => prev.filter((it) => it.id !== item.id))
      router.refresh()
    } catch {
      toast.error(te("generic"))
    } finally {
      setBusyId(null)
    }
  }

  const handleReject = async (item: OfficerDuplicateItem) => {
    setBusyId(item.id)
    try {
      const res = await api.POST("/officer/duplicates/{id}/reject", {
        params: { path: { id: item.id } },
      })
      if (res.error) {
        toast.error(te("generic"))
        return
      }
      toast.success(t("rejectSuccess"))
      setItems((prev) => prev.filter((it) => it.id !== item.id))
      router.refresh()
    } catch {
      toast.error(te("generic"))
    } finally {
      setBusyId(null)
    }
  }

  const handleManualMerge = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!manualSurvivor.trim() || !manualDuplicate.trim()) return
    setManualSubmitting(true)
    try {
      const res = await api.POST("/officer/duplicates/merge", {
        body: {
          report_type: manualType,
          survivor_id: manualSurvivor.trim(),
          duplicate_id: manualDuplicate.trim(),
        },
      })
      if (res.error) {
        toast.error(te("generic"))
        return
      }
      toast.success(t("manualMergeSuccess"))
      setManualSurvivor("")
      setManualDuplicate("")
      setShowManual(false)
      router.refresh()
    } catch {
      toast.error(te("generic"))
    } finally {
      setManualSubmitting(false)
    }
  }

  const tabs = [
    { key: "suggested", label: t("tabSuggested") },
    { key: "confirmed", label: t("tabConfirmed") },
    { key: "rejected", label: t("tabRejected") },
  ]

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-1 rounded-lg border bg-muted/40 p-1 text-sm">
          {tabs.map((tab) => (
            <Link
              key={tab.key}
              href={`/petugas/duplikat?status=${tab.key}`}
              className={`rounded-md px-3 py-1 font-medium transition-colors ${
                currentStatus === tab.key ? "bg-background text-foreground shadow-xs" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {tab.label}
            </Link>
          ))}
        </div>

        <Button type="button" variant="outline" size="sm" onClick={() => setShowManual((prev) => !prev)} className="gap-2">
          <GitMerge className="size-4" aria-hidden />
          {t("manualMergeTitle")}
        </Button>
      </div>

      {showManual ? (
        <form onSubmit={handleManualMerge} className="flex flex-col gap-4 rounded-xl border bg-card p-5 shadow-xs">
          <div>
            <h3 className="font-semibold">{t("manualMergeTitle")}</h3>
            <p className="text-sm text-muted-foreground">{t("manualMergeLead")}</p>
          </div>

          <div className="grid gap-3 sm:grid-cols-3">
            <div className="flex flex-col gap-1.5">
              <label htmlFor="manual-type" className="text-xs font-medium">
                {t("reportTypeLabel")}
              </label>
              <NativeSelect id="manual-type" value={manualType} onChange={(e) => setManualType(e.target.value as "found" | "lost")}>
                <option value="found">{t("typeFound")}</option>
                <option value="lost">{t("typeLost")}</option>
              </NativeSelect>
            </div>

            <div className="flex flex-col gap-1.5">
              <label htmlFor="manual-survivor" className="text-xs font-medium">
                {t("survivorCode")} (UUID)
              </label>
              <Input
                id="manual-survivor"
                placeholder="ID laporan yang bertahan"
                value={manualSurvivor}
                onChange={(e) => setManualSurvivor(e.target.value)}
                required
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <label htmlFor="manual-duplicate" className="text-xs font-medium">
                {t("duplicateCode")} (UUID)
              </label>
              <Input
                id="manual-duplicate"
                placeholder="ID laporan yang ditutup"
                value={manualDuplicate}
                onChange={(e) => setManualDuplicate(e.target.value)}
                required
              />
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="ghost" size="sm" onClick={() => setShowManual(false)}>
              {tc("cancel")}
            </Button>
            <Button type="submit" size="sm" disabled={manualSubmitting}>
              {manualSubmitting ? tc("submitting") : t("manualMergeSubmit")}
            </Button>
          </div>
        </form>
      ) : null}

      {items.length === 0 ? (
        <EmptyState icon={CopyCheck}>{t("emptyDuplicates")}</EmptyState>
      ) : (
        <div className="grid gap-4">
          {items.map((item) => {
            const selectedSurvivor = survivorSelection[item.id] ?? item.report_a.id
            const isBusy = busyId === item.id

            return (
              <div key={item.id} className="flex flex-col gap-4 rounded-xl border bg-card p-5 shadow-xs">
                <div className="flex flex-wrap items-center justify-between gap-2 border-b pb-3">
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="font-mono text-xs">
                      {item.report_type === "found" ? t("typeFound") : t("typeLost")}
                    </Badge>
                    <Badge variant="secondary" className="text-xs font-medium">
                      {t("scorePercent", { score: Math.round(item.score * 100) })}
                    </Badge>
                  </div>
                  <span className="text-xs text-muted-foreground">{formatDateTime(item.created_at)}</span>
                </div>

                <div className="grid gap-4 md:grid-cols-2">
                  {/* Report A */}
                  <div
                    className={`rounded-lg border p-4 transition-colors ${
                      selectedSurvivor === item.report_a.id ? "border-primary/50 bg-primary-soft/20" : "bg-muted/10"
                    }`}
                  >
                    <div className="mb-2 flex items-center justify-between">
                      <span className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">{t("reportA")}</span>
                      <Link href={`/petugas/laporan/${item.report_a.code}`} target="_blank" className="font-mono text-xs text-primary underline">
                        {item.report_a.code}
                      </Link>
                    </div>

                    <div className="flex gap-3">
                      {item.report_a.thumb_media_id ? (
                        <img
                          src={mediaUrl(item.report_a.thumb_media_id, "thumb")}
                          alt={item.report_a.item_name}
                          className="size-16 shrink-0 rounded-md border object-cover"
                        />
                      ) : (
                        <div className="flex size-16 shrink-0 items-center justify-center rounded-md border bg-muted text-xs text-muted-foreground">Foto</div>
                      )}
                      <div className="min-w-0 flex-1">
                        <p className="truncate font-medium">{item.report_a.item_name}</p>
                        <p className="mt-0.5 text-xs text-muted-foreground">{item.report_a.category_name}</p>
                        <p className="mt-0.5 text-xs text-muted-foreground">
                          {item.report_a.location_name} &bull; {formatDateTime(item.report_a.occurred_at)}
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Report B */}
                  <div
                    className={`rounded-lg border p-4 transition-colors ${
                      selectedSurvivor === item.report_b.id ? "border-primary/50 bg-primary-soft/20" : "bg-muted/10"
                    }`}
                  >
                    <div className="mb-2 flex items-center justify-between">
                      <span className="text-xs font-semibold tracking-wider text-muted-foreground uppercase">{t("reportB")}</span>
                      <Link href={`/petugas/laporan/${item.report_b.code}`} target="_blank" className="font-mono text-xs text-primary underline">
                        {item.report_b.code}
                      </Link>
                    </div>

                    <div className="flex gap-3">
                      {item.report_b.thumb_media_id ? (
                        <img
                          src={mediaUrl(item.report_b.thumb_media_id, "thumb")}
                          alt={item.report_b.item_name}
                          className="size-16 shrink-0 rounded-md border object-cover"
                        />
                      ) : (
                        <div className="flex size-16 shrink-0 items-center justify-center rounded-md border bg-muted text-xs text-muted-foreground">Foto</div>
                      )}
                      <div className="min-w-0 flex-1">
                        <p className="truncate font-medium">{item.report_b.item_name}</p>
                        <p className="mt-0.5 text-xs text-muted-foreground">{item.report_b.category_name}</p>
                        <p className="mt-0.5 text-xs text-muted-foreground">
                          {item.report_b.location_name} &bull; {formatDateTime(item.report_b.occurred_at)}
                        </p>
                      </div>
                    </div>
                  </div>
                </div>

                {item.status === "suggested" ? (
                  <div className="flex flex-wrap items-center justify-between gap-3 border-t pt-3">
                    <div className="flex flex-col gap-1">
                      <span className="text-xs font-medium text-muted-foreground">{t("survivorPrompt")}</span>
                      <div className="flex gap-4">
                        <label className="flex cursor-pointer items-center gap-1.5 text-sm">
                          <input
                            type="radio"
                            name={`survivor-${item.id}`}
                            value={item.report_a.id}
                            checked={selectedSurvivor === item.report_a.id}
                            onChange={() => setSurvivorSelection((prev) => ({ ...prev, [item.id]: item.report_a.id }))}
                            className="size-4 accent-primary"
                          />
                          <span>{t("survivorA", { code: item.report_a.code })}</span>
                        </label>
                        <label className="flex cursor-pointer items-center gap-1.5 text-sm">
                          <input
                            type="radio"
                            name={`survivor-${item.id}`}
                            value={item.report_b.id}
                            checked={selectedSurvivor === item.report_b.id}
                            onChange={() => setSurvivorSelection((prev) => ({ ...prev, [item.id]: item.report_b.id }))}
                            className="size-4 accent-primary"
                          />
                          <span>{t("survivorB", { code: item.report_b.code })}</span>
                        </label>
                      </div>
                    </div>

                    <div className="flex items-center gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={isBusy}
                        onClick={() => handleReject(item)}
                        className="gap-1.5 text-muted-foreground"
                      >
                        <X className="size-4" aria-hidden />
                        {t("rejectDuplicate")}
                      </Button>
                      <Button type="button" size="sm" disabled={isBusy} onClick={() => handleConfirm(item)} className="gap-1.5">
                        <Check className="size-4" aria-hidden />
                        {t("confirmMerge")}
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="flex items-center justify-between border-t pt-3 text-xs text-muted-foreground">
                    <Badge variant={item.status === "confirmed" ? "default" : "secondary"}>
                      {item.status === "confirmed" ? t("tabConfirmed") : t("tabRejected")}
                    </Badge>
                    {item.reviewed_at ? <span>{formatDateTime(item.reviewed_at)}</span> : null}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
