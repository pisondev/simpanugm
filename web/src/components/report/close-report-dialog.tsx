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

const lostReasons = ["found_by_owner", "no_longer_needed"] as const
const foundReasons = ["returned_directly", "reported_in_error"] as const

type Props = { kind: "lost" | "found"; id: string }

// FR-LOST-10 and FR-FOUND-09: closing asks for a reason, a found report also needs a note
export function CloseReportDialog({ kind, id }: Props) {
  const t = useTranslations("report.close")
  const ts = useTranslations("status.closeReason")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const reasons = kind === "lost" ? lostReasons : foundReasons
  const [reason, setReason] = useState<string>(reasons[0])
  const [note, setNote] = useState("")
  const [error, setError] = useState<string>()
  const close = useMutation({
    mutationFn: async () => {
      if (kind === "lost") {
        await call(
          api.POST("/lost-reports/{id}/close", {
            params: { path: { id } },
            body: { reason: reason as (typeof lostReasons)[number], note: note.trim() || undefined },
          }),
        )
      } else {
        await call(
          api.POST("/found-reports/{id}/close", { params: { path: { id } }, body: { reason: reason as (typeof foundReasons)[number], note: note.trim() } }),
        )
      }
    },
    onSuccess: () => {
      setOpen(false)
      toast.success(t("done"))
      router.refresh()
    },
    onError: (err) => setError(messageFor(err, te)),
  })
  const noteRequired = kind === "found"
  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger render={<Button variant="outline" />}>{kind === "lost" ? t("lostAction") : t("foundAction")}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{kind === "lost" ? t("lostTitle") : t("foundTitle")}</AlertDialogTitle>
          <AlertDialogDescription>{kind === "lost" ? t("lostBody") : t("foundBody")}</AlertDialogDescription>
        </AlertDialogHeader>
        <form
          id={`close-${id}`}
          className="flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault()
            if (noteRequired && note.trim().length < 5) {
              setError(t("noteMin"))
              return
            }
            setError(undefined)
            close.mutate()
          }}
        >
          <fieldset className="flex flex-col gap-2">
            <legend className="mb-2 text-sm font-medium">{t("reason")}</legend>
            {reasons.map((r) => (
              <label key={r} className="flex min-h-11 items-center gap-3">
                <input type="radio" name="reason" value={r} checked={reason === r} onChange={() => setReason(r)} className="size-4 accent-primary" />
                {ts(r)}
              </label>
            ))}
          </fieldset>
          <div className="flex flex-col gap-2">
            <label htmlFor={`close-note-${id}`} className="text-sm font-medium">
              {t("note")}
              {noteRequired ? null : <span className="ml-1 font-normal text-muted-foreground">({tc("optional")})</span>}
            </label>
            <Textarea
              id={`close-note-${id}`}
              rows={3}
              maxLength={500}
              placeholder={t("notePlaceholder")}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              aria-describedby={error ? `close-error-${id}` : undefined}
            />
          </div>
          <p id={`close-error-${id}`} aria-live="polite" className="text-sm text-destructive">
            {error ?? ""}
          </p>
        </form>
        <AlertDialogFooter>
          <AlertDialogCancel>{tc("cancel")}</AlertDialogCancel>
          <Button type="submit" form={`close-${id}`} variant="destructive" disabled={close.isPending}>
            {t("confirm")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
