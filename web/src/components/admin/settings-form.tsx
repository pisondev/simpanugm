"use client"

import { useMutation } from "@tanstack/react-query"
import { RotateCcw, Save } from "lucide-react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"
import type { SystemSettingItem } from "@/lib/api/types"

interface SettingsState {
  // Lost reports
  lostStaleDays: number
  lostExpiryGraceDays: number
  lostReopenDays: number
  // Found & custody
  foundWithFinderMaxHours: number
  custodyRetentionNormal: number
  custodyRetentionValuable: number
  custodyRetentionIdentity: number
  // Claims
  claimMaxOpen: number
  claimRejectFlagCount: number
  claimRejectFlagWindowDays: number
  claimInfoTimeoutDays: number
  claimApprovedTimeoutDays: number
  // Notifications
  notifyMatchThrottleHours: number
  quietHoursFrom: string
  quietHoursTo: string
  // Auth
  allowedEmailDomains: string
  // Analytics
  analyticsMinCellCount: number
}

function parseInitial(items: SystemSettingItem[]): SettingsState {
  const map = new Map<string, unknown>()
  for (const item of items) {
    map.set(item.key, item.value)
  }

  const quietHours = (map.get("notify.email_quiet_hours") as { from?: string; to?: string }) || {}
  const domains = (map.get("auth.allowed_email_domains") as string[]) || ["mail.ugm.ac.id", "ugm.ac.id"]

  return {
    lostStaleDays: Number(map.get("lost.stale_days")) || 60,
    lostExpiryGraceDays: Number(map.get("lost.expiry_grace_days")) || 14,
    lostReopenDays: Number(map.get("lost.reopen_days")) || 30,
    foundWithFinderMaxHours: Number(map.get("found.with_finder_max_hours")) || 48,
    custodyRetentionNormal: Number(map.get("custody.retention_days.normal")) || 30,
    custodyRetentionValuable: Number(map.get("custody.retention_days.valuable")) || 60,
    custodyRetentionIdentity: Number(map.get("custody.retention_days.identity")) || 30,
    claimMaxOpen: Number(map.get("claim.max_open")) || 3,
    claimRejectFlagCount: Number(map.get("claim.reject_flag_count")) || 3,
    claimRejectFlagWindowDays: Number(map.get("claim.reject_flag_window_days")) || 30,
    claimInfoTimeoutDays: Number(map.get("claim.info_timeout_days")) || 7,
    claimApprovedTimeoutDays: Number(map.get("claim.approved_timeout_days")) || 7,
    notifyMatchThrottleHours: Number(map.get("notify.match_throttle_hours")) || 6,
    quietHoursFrom: quietHours.from || "22:00",
    quietHoursTo: quietHours.to || "06:00",
    allowedEmailDomains: domains.join(", "),
    analyticsMinCellCount: Number(map.get("analytics.min_cell_count")) || 3,
  }
}

export function SettingsForm({ initialItems }: { initialItems: SystemSettingItem[] }) {
  const tc = useTranslations("common")
  const router = useRouter()
  const [state, setState] = useState<SettingsState>(() => parseInitial(initialItems))

  const saveMutation = useMutation({
    mutationFn: () => {
      const domains = state.allowedEmailDomains
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean)

      const payload = [
        { key: "lost.stale_days", value: Number(state.lostStaleDays) },
        { key: "lost.expiry_grace_days", value: Number(state.lostExpiryGraceDays) },
        { key: "lost.reopen_days", value: Number(state.lostReopenDays) },
        { key: "found.with_finder_max_hours", value: Number(state.foundWithFinderMaxHours) },
        { key: "custody.retention_days.normal", value: Number(state.custodyRetentionNormal) },
        { key: "custody.retention_days.valuable", value: Number(state.custodyRetentionValuable) },
        { key: "custody.retention_days.identity", value: Number(state.custodyRetentionIdentity) },
        { key: "claim.max_open", value: Number(state.claimMaxOpen) },
        { key: "claim.reject_flag_count", value: Number(state.claimRejectFlagCount) },
        { key: "claim.reject_flag_window_days", value: Number(state.claimRejectFlagWindowDays) },
        { key: "claim.info_timeout_days", value: Number(state.claimInfoTimeoutDays) },
        { key: "claim.approved_timeout_days", value: Number(state.claimApprovedTimeoutDays) },
        { key: "notify.match_throttle_hours", value: Number(state.notifyMatchThrottleHours) },
        {
          key: "notify.email_quiet_hours",
          value: { from: state.quietHoursFrom, to: state.quietHoursTo },
        },
        { key: "auth.allowed_email_domains", value: domains },
        { key: "analytics.min_cell_count", value: Number(state.analyticsMinCellCount) },
      ]

      return call(api.PATCH("/admin/settings", { body: { settings: payload } }))
    },
    onSuccess: () => {
      toast.success("Pengaturan sistem berhasil disimpan.")
      router.refresh()
    },
    onError: (err: Error) => {
      toast.error(err.message || "Gagal menyimpan pengaturan.")
    },
  })

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        saveMutation.mutate()
      }}
      className="flex flex-col gap-6"
    >
      {/* Lost Reports */}
      <Card>
        <CardHeader>
          <CardTitle>Batas Waktu Laporan Kehilangan</CardTitle>
          <CardDescription>Aturan siklus hidup laporan kehilangan barang (BR-LST).</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-3">
          <div className="flex flex-col gap-2">
            <label htmlFor="lost-stale" className="text-sm font-medium">
              Masa Aktif Laporan (Hari)
            </label>
            <Input
              id="lost-stale"
              type="number"
              min={14}
              max={180}
              value={state.lostStaleDays}
              onChange={(e) => setState((s) => ({ ...s, lostStaleDays: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Rentang: 14 - 180 hari</span>
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="lost-grace" className="text-sm font-medium">
              Tenggang Kedaluwarsa (Hari)
            </label>
            <Input
              id="lost-grace"
              type="number"
              min={3}
              max={30}
              value={state.lostExpiryGraceDays}
              onChange={(e) => setState((s) => ({ ...s, lostExpiryGraceDays: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Rentang: 3 - 30 hari</span>
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="lost-reopen" className="text-sm font-medium">
              Batas Buka Kembali (Hari)
            </label>
            <Input
              id="lost-reopen"
              type="number"
              min={7}
              max={90}
              value={state.lostReopenDays}
              onChange={(e) => setState((s) => ({ ...s, lostReopenDays: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Rentang: 7 - 90 hari</span>
          </div>
        </CardContent>
      </Card>

      {/* Found & Custody */}
      <Card>
        <CardHeader>
          <CardTitle>Barang Temuan & Retensi Penitipan</CardTitle>
          <CardDescription>Batas penyerahan barang dan masa simpan pos (BR-FND & BR-CST).</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div className="flex flex-col gap-2">
            <label htmlFor="found-finder" className="text-sm font-medium">
              Batas di Penemu (Jam)
            </label>
            <Input
              id="found-finder"
              type="number"
              min={12}
              max={72}
              value={state.foundWithFinderMaxHours}
              onChange={(e) => setState((s) => ({ ...s, foundWithFinderMaxHours: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Rentang: 12 - 72 jam</span>
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="custody-normal" className="text-sm font-medium">
              Retensi Barang Biasa (Hari)
            </label>
            <Input
              id="custody-normal"
              type="number"
              min={7}
              max={120}
              value={state.custodyRetentionNormal}
              onChange={(e) => setState((s) => ({ ...s, custodyRetentionNormal: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Rentang: 7 - 120 hari</span>
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="custody-valuable" className="text-sm font-medium">
              Retensi Barang Berharga (Hari)
            </label>
            <Input
              id="custody-valuable"
              type="number"
              min={14}
              max={180}
              value={state.custodyRetentionValuable}
              onChange={(e) => setState((s) => ({ ...s, custodyRetentionValuable: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Rentang: 14 - 180 hari</span>
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="custody-identity" className="text-sm font-medium">
              Retensi Identitas (Hari)
            </label>
            <Input
              id="custody-identity"
              type="number"
              min={7}
              max={90}
              value={state.custodyRetentionIdentity}
              onChange={(e) => setState((s) => ({ ...s, custodyRetentionIdentity: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Rentang: 7 - 90 hari</span>
          </div>
        </CardContent>
      </Card>

      {/* Claims */}
      <Card>
        <CardHeader>
          <CardTitle>Batas & Ambang Klaim</CardTitle>
          <CardDescription>Pencegahan spam klaim dan batas waktu penyelesaian (BR-CLM).</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div className="flex flex-col gap-2">
            <label htmlFor="claim-max" className="text-sm font-medium">
              Maksimal Klaim Terbuka
            </label>
            <Input
              id="claim-max"
              type="number"
              min={1}
              max={10}
              value={state.claimMaxOpen}
              onChange={(e) => setState((s) => ({ ...s, claimMaxOpen: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Rentang: 1 - 10 klaim</span>
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="claim-reject-count" className="text-sm font-medium">
              Ambang Penolakan Pembatasan
            </label>
            <Input
              id="claim-reject-count"
              type="number"
              min={2}
              max={10}
              value={state.claimRejectFlagCount}
              onChange={(e) => setState((s) => ({ ...s, claimRejectFlagCount: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Rentang: 2 - 10 penolakan</span>
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="claim-reject-win" className="text-sm font-medium">
              Jendela Hari Penolakan
            </label>
            <Input
              id="claim-reject-win"
              type="number"
              min={7}
              max={90}
              value={state.claimRejectFlagWindowDays}
              onChange={(e) => setState((s) => ({ ...s, claimRejectFlagWindowDays: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Rentang: 7 - 90 hari</span>
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="claim-info-timeout" className="text-sm font-medium">
              Batas Lengkapi Info (Hari)
            </label>
            <Input
              id="claim-info-timeout"
              type="number"
              min={2}
              max={30}
              value={state.claimInfoTimeoutDays}
              onChange={(e) => setState((s) => ({ ...s, claimInfoTimeoutDays: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Rentang: 2 - 30 hari</span>
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="claim-appr-timeout" className="text-sm font-medium">
              Batas Pengambilan Barang (Hari)
            </label>
            <Input
              id="claim-appr-timeout"
              type="number"
              min={2}
              max={30}
              value={state.claimApprovedTimeoutDays}
              onChange={(e) => setState((s) => ({ ...s, claimApprovedTimeoutDays: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Rentang: 2 - 30 hari</span>
          </div>
        </CardContent>
      </Card>

      {/* Notifications & Quiet Hours */}
      <Card>
        <CardHeader>
          <CardTitle>Notifikasi & Jam Hening</CardTitle>
          <CardDescription>Pengaturan frekuensi email dan jam istirahat malam WIB (BR-NTF-03).</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-3">
          <div className="flex flex-col gap-2">
            <label htmlFor="notify-throttle" className="text-sm font-medium">
              Jeda Match Email (Jam)
            </label>
            <Input
              id="notify-throttle"
              type="number"
              min={1}
              max={24}
              value={state.notifyMatchThrottleHours}
              onChange={(e) => setState((s) => ({ ...s, notifyMatchThrottleHours: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Rentang: 1 - 24 jam</span>
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="quiet-from" className="text-sm font-medium">
              Jam Hening Mulai (WIB)
            </label>
            <Input
              id="quiet-from"
              type="text"
              placeholder="22:00"
              pattern="^([01][0-9]|2[0-3]):[0-5][0-9]$"
              value={state.quietHoursFrom}
              onChange={(e) => setState((s) => ({ ...s, quietHoursFrom: e.target.value }))}
            />
            <span className="text-xs text-muted-foreground">Format: HH:MM</span>
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="quiet-to" className="text-sm font-medium">
              Jam Hening Selesai (WIB)
            </label>
            <Input
              id="quiet-to"
              type="text"
              placeholder="06:00"
              pattern="^([01][0-9]|2[0-3]):[0-5][0-9]$"
              value={state.quietHoursTo}
              onChange={(e) => setState((s) => ({ ...s, quietHoursTo: e.target.value }))}
            />
            <span className="text-xs text-muted-foreground">Format: HH:MM</span>
          </div>
        </CardContent>
      </Card>

      {/* Auth & Security */}
      <Card>
        <CardHeader>
          <CardTitle>Autentikasi & Analitik</CardTitle>
          <CardDescription>Domain email anggota dan ambang batas k-anonymity (BR-ACC-01).</CardDescription>
        </CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">
          <div className="flex flex-col gap-2">
            <label htmlFor="email-domains" className="text-sm font-medium">
              Domain Email Diizinkan
            </label>
            <Input
              id="email-domains"
              type="text"
              value={state.allowedEmailDomains}
              onChange={(e) => setState((s) => ({ ...s, allowedEmailDomains: e.target.value }))}
            />
            <span className="text-xs text-muted-foreground">Pisahkan dengan koma (contoh: mail.ugm.ac.id, ugm.ac.id)</span>
          </div>

          <div className="flex flex-col gap-2">
            <label htmlFor="analytics-k" className="text-sm font-medium">
              Ambang K-Anonymity Analitik
            </label>
            <Input
              id="analytics-k"
              type="number"
              min={3}
              max={20}
              value={state.analyticsMinCellCount}
              onChange={(e) => setState((s) => ({ ...s, analyticsMinCellCount: Number(e.target.value) }))}
            />
            <span className="text-xs text-muted-foreground">Minimal data per sel (k &gt;= 3)</span>
          </div>
        </CardContent>
      </Card>

      <div className="flex items-center justify-end gap-3">
        <Button type="button" variant="outline" onClick={() => setState(parseInitial(initialItems))} disabled={saveMutation.isPending}>
          <RotateCcw className="size-4" aria-hidden />
          {tc("reset")}
        </Button>
        <Button type="submit" disabled={saveMutation.isPending}>
          <Save className="size-4" aria-hidden />
          {saveMutation.isPending ? tc("saving") : tc("save")}
        </Button>
      </div>
    </form>
  )
}
