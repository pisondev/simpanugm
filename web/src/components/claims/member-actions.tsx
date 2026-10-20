"use client"

import { useMutation } from "@tanstack/react-query"
import { Clock, KeyRound, Undo2 } from "lucide-react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api/client"
import { call, messageFor } from "@/lib/api/problem"
import type { Claim } from "@/lib/api/types"
import { formatDateTime } from "@/lib/format"

// US-26 and US-27: what the claimant can do while the claim is open
export function MemberClaimActions({ claim }: { claim: Claim }) {
  const can = (a: string) => claim.actions.includes(a)
  return (
    <div className="flex flex-col gap-4">
      {claim.status === "approved" ? <HandoverCodePanel claim={claim} /> : null}
      {claim.status === "on_hold" ? <OnHoldNote /> : null}
      {can("respond") ? <RespondForm code={claim.code} /> : null}
      {can("withdraw") ? <WithdrawDialog code={claim.code} /> : null}
    </div>
  )
}

function OnHoldNote() {
  const t = useTranslations("claims.member")
  return (
    <Alert>
      <Clock aria-hidden />
      <AlertTitle>{t("onHoldTitle")}</AlertTitle>
      <AlertDescription>{t("onHoldBody")}</AlertDescription>
    </Alert>
  )
}

// FR-CUST-04: the code is shown only to the claimant and only while the claim is approved
function HandoverCodePanel({ claim }: { claim: Claim }) {
  const t = useTranslations("claims.member")
  const post = claim.found_report.holding_post?.name ?? ""
  return (
    <section aria-labelledby="kode-serah" className="ai-panel flex flex-col gap-3 p-4">
      <div className="flex items-center gap-2">
        <KeyRound className="size-5 text-primary" aria-hidden />
        <h2 id="kode-serah" className="font-semibold">
          {t("codeTitle")}
        </h2>
      </div>
      {claim.handover_locked ? (
        <p>{t("codeLocked")}</p>
      ) : claim.handover_code ? (
        <>
          <p aria-label={t("codeAria", { code: claim.handover_code.split("").join(" ") })} className="text-4xl font-semibold tracking-[0.3em] tabular-nums">
            {claim.handover_code}
          </p>
          <p>{t("codeBody", { post })}</p>
        </>
      ) : (
        <p>{t("codeMissing")}</p>
      )}
      {claim.approved_expires_at ? <p className="text-sm text-muted-foreground">{t("codeUntil", { at: formatDateTime(claim.approved_expires_at) })}</p> : null}
    </section>
  )
}

function RespondForm({ code }: { code: string }) {
  const t = useTranslations("claims.member")
  const te = useTranslations("errors")
  const router = useRouter()
  const [message, setMessage] = useState("")
  const respond = useMutation({
    mutationFn: () => call(api.POST("/claims/{id}/respond", { params: { path: { id: code } }, body: { message: message.trim() } })),
    onSuccess: () => {
      setMessage("")
      toast.success(t("respondDone"))
      router.refresh()
    },
    onError: (err) => toast(messageFor(err, te)),
  })
  return (
    <form
      className="surface flex flex-col gap-2 border-warning/40 bg-warning/10 p-4"
      onSubmit={(e) => {
        e.preventDefault()
        if (message.trim()) {
          respond.mutate()
        }
      }}
    >
      <label htmlFor="claim-respond" className="font-semibold">
        {t("respondTitle")}
      </label>
      <p className="text-sm text-muted-foreground">{t("respondLead")}</p>
      <Textarea id="claim-respond" rows={3} maxLength={2000} value={message} onChange={(e) => setMessage(e.target.value)} />
      <Button type="submit" className="w-fit" disabled={respond.isPending || !message.trim()}>
        {t("respondSend")}
      </Button>
    </form>
  )
}

function WithdrawDialog({ code }: { code: string }) {
  const t = useTranslations("claims.member")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const withdraw = useMutation({
    mutationFn: () => call(api.POST("/claims/{id}/withdraw", { params: { path: { id: code } } })),
    onSuccess: () => {
      toast.success(t("withdrawDone"))
      router.refresh()
    },
    onError: (err) => toast(messageFor(err, te)),
  })
  return (
    <AlertDialog>
      <AlertDialogTrigger render={<Button variant="outline" className="w-fit" />}>
        <Undo2 aria-hidden />
        {t("withdraw")}
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("withdrawTitle")}</AlertDialogTitle>
          <AlertDialogDescription>{t("withdrawBody")}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{tc("cancel")}</AlertDialogCancel>
          <AlertDialogAction variant="destructive" onClick={() => withdraw.mutate()}>
            {t("withdrawConfirm")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
