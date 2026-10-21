"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { ClipboardCheck, Handshake, KeyRound, ShieldCheck, UserCheck } from "lucide-react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { OfficerFeatureLabel } from "@/components/dashboard/officer-feature-label"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api/client"
import { call, isApiError, messageFor } from "@/lib/api/problem"
import type { Claim, ClaimCheck, RejectionReason } from "@/lib/api/types"
import { cn } from "@/lib/utils"

import { ActionDialog } from "./action-dialog"

const rejectionReasons: RejectionReason[] = [
  "details_mismatch",
  "insufficient_evidence",
  "item_already_returned",
  "duplicate_claim",
  "suspected_fraud",
  "other",
]
const idTypes = ["ktm", "ktp", "sim", "passport", "other"] as const
const results = ["pending", "pass", "fail", "not_applicable"] as const

function useChecks(code: string, enabled: boolean) {
  return useQuery({
    queryKey: ["claim-checks", code],
    queryFn: () => call(api.GET("/claims/{id}/checks", { params: { path: { id: code } } })),
    enabled,
  })
}

// the next steps K2, K3, K6, K10, K13, K15 and the code regeneration, each only when the api lists the action
export function OfficerClaimActions({ claim }: { claim: Claim }) {
  const t = useTranslations("claims.officer")
  const tr = useTranslations("claims.reason")
  const te = useTranslations("errors")
  const router = useRouter()
  const can = (a: string) => claim.actions.includes(a)
  const checks = useChecks(claim.code, can("approve"))
  const id = { path: { id: claim.code } }
  const simple = useMutation({
    mutationFn: async (kind: "pick_up" | "approve" | "regenerate") => {
      switch (kind) {
        case "pick_up":
          return call(api.POST("/claims/{id}/pick-up", { params: { path: id.path } }))
        case "approve":
          return call(api.POST("/claims/{id}/approve", { params: { path: id.path, header: { "If-Match": `"${claim.version}"` } }, body: {} }))
        default:
          return call(api.POST("/claims/{id}/handover-code", { params: { path: id.path } }))
      }
    },
    onSuccess: (_data, kind) => {
      toast.success(t(`done.${kind}`))
      router.refresh()
    },
    onError: (err) => toast(messageFor(err, te)),
  })
  const noPost = !can("approve") && claim.status === "under_review"
  return (
    <section aria-labelledby="c-langkah" className="flex flex-col gap-3 rounded-xl border bg-card p-4">
      <h2 id="c-langkah" className="font-semibold">
        {t("actionsTitle")}
      </h2>
      {noPost ? (
        <Alert>
          <ShieldCheck aria-hidden />
          <AlertTitle>{t("otherPostTitle")}</AlertTitle>
          <AlertDescription>{t("otherPostBody")}</AlertDescription>
        </Alert>
      ) : null}
      <div className="flex flex-wrap gap-2">
        {can("pick_up") ? (
          <Button onClick={() => simple.mutate("pick_up")} disabled={simple.isPending}>
            <UserCheck aria-hidden />
            {t("pickUp")}
          </Button>
        ) : null}
        {can("request_info") ? (
          <ActionDialog
            trigger={t("requestInfo")}
            title={t("requestInfoTitle")}
            body={t("requestInfoBody")}
            label={t("requestInfoLabel")}
            minLength={3}
            maxLength={2000}
            confirm={t("requestInfoConfirm")}
            done={t("done.request_info")}
            run={({ text }) => call(api.POST("/claims/{id}/request-info", { params: { path: id.path }, body: { message: text } }))}
          />
        ) : null}
        {can("approve") ? (
          <div className="flex flex-col items-start gap-1">
            <Button onClick={() => simple.mutate("approve")} disabled={simple.isPending || !checks.data?.complete}>
              <ShieldCheck aria-hidden />
              {t("approve")}
            </Button>
            {!checks.data?.complete ? <p className="text-xs text-muted-foreground">{t("approveNeedsChecks")}</p> : null}
          </div>
        ) : null}
        {can("reject") ? (
          <ActionDialog
            destructive
            trigger={t("reject")}
            title={t("rejectTitle")}
            body={t("rejectBody")}
            reasons={rejectionReasons.map((r) => ({ value: r, label: tr(r) }))}
            reasonLabel={t("rejectReason")}
            label={t("rejectNote")}
            confirm={t("rejectConfirm")}
            done={t("done.reject")}
            run={({ text, reason }) =>
              call(api.POST("/claims/{id}/reject", { params: { path: id.path }, body: { reason: reason as RejectionReason, note: text || undefined } }))
            }
          />
        ) : null}
        {can("revoke") ? (
          <ActionDialog
            destructive
            trigger={t("revoke")}
            title={t("revokeTitle")}
            body={t("revokeBody")}
            label={t("reasonLabel")}
            minLength={3}
            confirm={t("revokeConfirm")}
            done={t("done.revoke")}
            run={({ text }) => call(api.POST("/claims/{id}/revoke", { params: { path: id.path }, body: { reason: text } }))}
          />
        ) : null}
        {can("regenerate_code") ? (
          <Button variant="outline" onClick={() => simple.mutate("regenerate")} disabled={simple.isPending}>
            <KeyRound aria-hidden />
            {t("regenerate")}
          </Button>
        ) : null}
        {can("reopen") ? (
          <ActionDialog
            trigger={t("reopen")}
            title={t("reopenTitle")}
            body={t("reopenBody")}
            label={t("reasonLabel")}
            minLength={3}
            confirm={t("reopenConfirm")}
            done={t("done.reopen")}
            run={({ text }) => call(api.POST("/claims/{id}/reopen", { params: { path: id.path }, body: { reason: text } }))}
          />
        ) : null}
      </div>
      {claim.handover_locked ? <p className="text-sm text-destructive">{t("locked")}</p> : null}
    </section>
  )
}

// AI-08: the checklist an officer works through before approving. Suggestions are hints, the result is the officer's
export function ClaimChecklist({ claim }: { claim: Claim }) {
  const t = useTranslations("claims.checks")
  const te = useTranslations("errors")
  const client = useQueryClient()
  const editable = claim.actions.includes("approve") || claim.actions.includes("request_info")
  const query = useChecks(claim.code, true)
  const [draft, setDraft] = useState<Record<string, { result: string; note: string }>>({})
  const items = query.data?.items ?? []
  const value = (c: ClaimCheck) => draft[c.item_key] ?? { result: c.result, note: c.note ?? "" }
  const save = useMutation({
    mutationFn: () =>
      call(
        api.PUT("/claims/{id}/checks", {
          params: { path: { id: claim.code } },
          body: {
            items: items
              .filter((c) => !c.info_only)
              .map((c) => ({ item_key: c.item_key, result: value(c).result as (typeof results)[number], note: value(c).note.trim() || undefined })),
          },
        }),
      ),
    onSuccess: (data) => {
      client.setQueryData(["claim-checks", claim.code], data)
      setDraft({})
      toast.success(t("saved"))
    },
    onError: (err) => toast(messageFor(err, te)),
  })
  if (query.isPending) {
    return <p className="text-sm text-muted-foreground">{t("loading")}</p>
  }
  return (
    <section id="verification-checklist" aria-labelledby="c-checks" className="flex scroll-mt-24 flex-col gap-3 rounded-xl border bg-card p-4">
      <OfficerFeatureLabel feature="checklist" />
      <div className="flex items-center gap-2">
        <ClipboardCheck className="size-5 text-primary" aria-hidden />
        <h2 id="c-checks" className="font-semibold">
          {t("title")}
        </h2>
      </div>
      <p className="text-sm text-muted-foreground">{t("lead")}</p>
      <ul className="flex flex-col gap-3">
        {items.map((c) => {
          const v = value(c)
          return (
            <li key={c.item_key} className="flex flex-col gap-2 rounded-lg border p-3">
              <p className="font-medium">
                {c.label}
                {c.required ? null : <span className="ml-1 font-normal text-muted-foreground">({t("optional")})</span>}
              </p>
              {c.suggestion.text ? (
                <p className={cn("text-sm", c.suggestion.level === "warn" ? "text-warning-text" : "text-muted-foreground")}>{c.suggestion.text}</p>
              ) : null}
              {c.info_only ? null : (
                <div className="grid gap-2 sm:grid-cols-[12rem_1fr]">
                  <NativeSelect
                    aria-label={t("resultFor", { label: c.label })}
                    value={v.result}
                    disabled={!editable}
                    onChange={(e) => setDraft((prev) => ({ ...prev, [c.item_key]: { ...v, result: e.target.value } }))}
                  >
                    {results.map((r) => (
                      <option key={r} value={r}>
                        {t(`result.${r}`)}
                      </option>
                    ))}
                  </NativeSelect>
                  <Input
                    aria-label={t("noteFor", { label: c.label })}
                    placeholder={t("notePlaceholder")}
                    maxLength={500}
                    value={v.note}
                    disabled={!editable}
                    onChange={(e) => setDraft((prev) => ({ ...prev, [c.item_key]: { ...v, note: e.target.value } }))}
                  />
                </div>
              )}
            </li>
          )
        })}
      </ul>
      <div className="flex flex-wrap items-center gap-3">
        {editable ? (
          <Button onClick={() => save.mutate()} disabled={save.isPending}>
            {t("save")}
          </Button>
        ) : null}
        <p className={cn("text-sm", query.data?.complete ? "text-success-text" : "text-muted-foreground")}>
          {query.data?.complete ? t("complete") : t("incomplete")}
        </p>
      </div>
    </section>
  )
}

// FR-CUST-04, BR-HND-02 to BR-HND-04: the code from the claimant, the ID type and last four, and the name check
export function HandoverForm({ claim }: { claim: Claim }) {
  const t = useTranslations("claims.handover")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const walkIn = !claim.claimant
  const [code, setCode] = useState("")
  const [idType, setIdType] = useState<(typeof idTypes)[number]>("ktm")
  const [last4, setLast4] = useState("")
  const [named, setNamed] = useState(false)
  const [error, setError] = useState<string>()
  const handover = useMutation({
    mutationFn: () =>
      call(
        api.POST("/found-reports/{id}/handover", {
          params: { path: { id: claim.found_report.code } },
          body: { claim_id: claim.id, code: walkIn ? undefined : code, recipient_id_type: idType, recipient_id_last4: last4.trim(), name_confirmed: named },
        }),
      ),
    onSuccess: () => {
      toast.success(t("done"))
      router.refresh()
    },
    onError: (err) => {
      setError(messageFor(err, te))
      if (isApiError(err, 422, "handover_locked")) router.refresh()
    },
  })
  return (
    <section
      id="handover-to-owner"
      aria-labelledby="c-serah"
      className="flex scroll-mt-24 flex-col gap-3 rounded-xl border border-primary/30 bg-primary-soft/50 p-4"
    >
      <OfficerFeatureLabel feature="handover" />
      <div className="flex items-center gap-2">
        <Handshake className="size-5 text-primary" aria-hidden />
        <h2 id="c-serah" className="font-semibold">
          {t("title")}
        </h2>
      </div>
      <p className="text-sm text-muted-foreground">{walkIn ? t("walkInLead") : t("lead")}</p>
      <form
        className="grid gap-3 sm:grid-cols-2"
        onSubmit={(e) => {
          e.preventDefault()
          setError(undefined)
          handover.mutate()
        }}
      >
        {walkIn ? null : (
          <div className="flex flex-col gap-2 sm:col-span-2">
            <label htmlFor="h-code" className="text-sm font-medium">
              {t("code")}
            </label>
            <Input
              id="h-code"
              inputMode="numeric"
              autoComplete="off"
              pattern="[0-9]{6}"
              maxLength={6}
              required
              value={code}
              onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
              className="font-mono tracking-[0.2em]"
            />
          </div>
        )}
        <div className="flex flex-col gap-2">
          <label htmlFor="h-type" className="text-sm font-medium">
            {t("idType")}
          </label>
          <NativeSelect id="h-type" value={idType} onChange={(e) => setIdType(e.target.value as (typeof idTypes)[number])}>
            {idTypes.map((i) => (
              <option key={i} value={i}>
                {t(`idTypes.${i}`)}
              </option>
            ))}
          </NativeSelect>
        </div>
        <div className="flex flex-col gap-2">
          <label htmlFor="h-last4" className="text-sm font-medium">
            {t("last4")}
          </label>
          <Input
            id="h-last4"
            inputMode="numeric"
            autoComplete="off"
            pattern="[0-9]{4}"
            maxLength={4}
            required
            value={last4}
            onChange={(e) => setLast4(e.target.value.replace(/\D/g, ""))}
            className="font-mono tracking-[0.2em]"
          />
        </div>
        <label className="flex items-start gap-3 sm:col-span-2">
          <Checkbox checked={named} onCheckedChange={(v) => setNamed(v === true)} className="mt-0.5" />
          <span>{walkIn ? t("nameWalkIn", { name: claim.visitor_name ?? "" }) : t("name", { name: claim.claimant?.full_name ?? "" })}</span>
        </label>
        <p aria-live="polite" className="text-sm text-destructive sm:col-span-2">
          {error ?? ""}
        </p>
        <div className="sm:col-span-2">
          <Button type="submit" disabled={handover.isPending || !named || last4.length !== 4 || (!walkIn && code.length !== 6)}>
            {handover.isPending ? tc("saving") : t("confirm")}
          </Button>
        </div>
      </form>
    </section>
  )
}

export function InternalNote({ claim }: { claim: Claim }) {
  const t = useTranslations("claims.officer")
  const te = useTranslations("errors")
  const router = useRouter()
  const [note, setNote] = useState(claim.internal_note ?? "")
  const save = useMutation({
    mutationFn: () => call(api.PATCH("/claims/{id}/internal-note", { params: { path: { id: claim.code } }, body: { note: note.trim() } })),
    onSuccess: () => {
      toast.success(t("noteSaved"))
      router.refresh()
    },
    onError: (err) => toast(messageFor(err, te)),
  })
  return (
    <section aria-labelledby="c-catatan" className="flex flex-col gap-2 rounded-xl border bg-card p-4">
      <label htmlFor="c-note" id="c-catatan" className="font-semibold">
        {t("noteTitle")}
      </label>
      <p className="text-sm text-muted-foreground">{t("noteLead")}</p>
      <Textarea id="c-note" rows={3} maxLength={1000} value={note} onChange={(e) => setNote(e.target.value)} />
      <Button variant="outline" className="w-fit" onClick={() => save.mutate()} disabled={save.isPending || note.trim() === (claim.internal_note ?? "")}>
        {t("noteSave")}
      </Button>
    </section>
  )
}
