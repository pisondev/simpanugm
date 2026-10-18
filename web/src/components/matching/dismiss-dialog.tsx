"use client"

import { useMutation } from "@tanstack/react-query"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import {
  AlertDialog,
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

type Props = {
  matchId: string
  // the owner hides the candidate for themselves, an officer dismisses it for everyone (US-21)
  scope: "owner" | "officer"
  // where to go afterwards, the page refreshes in place when absent
  then?: string
  size?: "default" | "sm"
}

export function DismissDialog({ matchId, scope, then, size = "default" }: Props) {
  const t = useTranslations("matching")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const [reason, setReason] = useState("")
  const [error, setError] = useState<string>()
  const owner = scope === "owner"
  const dismiss = useMutation({
    mutationFn: () => call(api.POST("/matches/{id}/dismiss", { params: { path: { id: matchId } }, body: { reason: reason.trim() || undefined } })),
    onSuccess: () => {
      setOpen(false)
      toast.success(owner ? t("notMineDone") : t("officer.dismissDone"))
      if (then) {
        router.push(then)
      }
      router.refresh()
    },
    onError: (err) => setError(messageFor(err, te)),
  })
  const formId = `dismiss-${matchId}`
  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger render={<Button variant="outline" size={size} />}>{owner ? t("notMine") : t("officer.dismiss")}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{owner ? t("notMineTitle") : t("officer.dismissTitle")}</AlertDialogTitle>
          <AlertDialogDescription>{owner ? t("notMineBody") : t("officer.dismissBody")}</AlertDialogDescription>
        </AlertDialogHeader>
        <form
          id={formId}
          className="flex flex-col gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            setError(undefined)
            dismiss.mutate()
          }}
        >
          <label htmlFor={`${formId}-reason`} className="text-sm font-medium">
            {t("reason")}
          </label>
          <Textarea
            id={`${formId}-reason`}
            rows={2}
            maxLength={300}
            placeholder={t("reasonPlaceholder")}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            aria-describedby={error ? `${formId}-error` : undefined}
          />
          <p id={`${formId}-error`} aria-live="polite" className="text-sm text-destructive">
            {error ?? ""}
          </p>
        </form>
        <AlertDialogFooter>
          <AlertDialogCancel>{tc("cancel")}</AlertDialogCancel>
          <Button type="submit" form={formId} variant="destructive" disabled={dismiss.isPending}>
            {owner ? t("notMineConfirm") : t("officer.dismissConfirm")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
