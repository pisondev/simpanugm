"use client"

import { useMutation } from "@tanstack/react-query"
import { ArchiveX } from "lucide-react"
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
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api/client"
import { call, messageFor } from "@/lib/api/problem"
import type { FoundReport } from "@/lib/api/types"

const dispositionKinds = ["donated", "destroyed", "transferred_to_unit", "returned_to_finder"] as const

type Props = { report: FoundReport }

// US-33, FR-CUST-06: record disposition for unclaimed items
export function CustodyDispositionDialog({ report }: Props) {
  const t = useTranslations("officer.disposition")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const isIdentity = report.category.sensitivity === "identity"
  const [kind, setKind] = useState<string>(isIdentity ? "transferred_to_unit" : "donated")
  const [recipient, setRecipient] = useState("")
  const [note, setNote] = useState("")
  const [error, setError] = useState<string>()

  const dispose = useMutation({
    mutationFn: () =>
      call(
        api.POST("/found-reports/{id}/dispose", {
          params: { path: { id: report.code } },
          body: {
            kind: kind as (typeof dispositionKinds)[number],
            note: note.trim(),
            recipient: recipient.trim() || undefined,
          },
        }),
      ),
    onSuccess: () => {
      setOpen(false)
      toast.success(t("done"))
      router.refresh()
    },
    onError: (err) => setError(messageFor(err, te)),
  })

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger render={<Button variant="outline" />}>
        <ArchiveX aria-hidden />
        {t("trigger")}
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("title")}</AlertDialogTitle>
          <AlertDialogDescription>{t("lead")}</AlertDialogDescription>
        </AlertDialogHeader>
        <form
          className="flex flex-col gap-3 py-2"
          onSubmit={(e) => {
            e.preventDefault()
            setError(undefined)
            dispose.mutate()
          }}
        >
          <div className="flex flex-col gap-2">
            <label htmlFor="disp-kind" className="text-sm font-medium">
              {t("kindLabel")}
            </label>
            <NativeSelect id="disp-kind" value={kind} disabled={isIdentity} onChange={(e) => setKind(e.target.value)}>
              {isIdentity ? (
                <option value="transferred_to_unit">{t("kinds.transferred_to_unit")}</option>
              ) : (
                dispositionKinds.map((k) => (
                  <option key={k} value={k}>
                    {t(`kinds.${k}`)}
                  </option>
                ))
              )}
            </NativeSelect>
          </div>
          <div className="flex flex-col gap-2">
            <label htmlFor="disp-recipient" className="text-sm font-medium">
              {t("recipientLabel")}
            </label>
            <Input
              id="disp-recipient"
              placeholder={t("recipientPlaceholder")}
              maxLength={100}
              value={recipient}
              onChange={(e) => setRecipient(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-2">
            <label htmlFor="disp-note" className="text-sm font-medium">
              {t("noteLabel")}
            </label>
            <Textarea
              id="disp-note"
              rows={3}
              placeholder={t("notePlaceholder")}
              minLength={5}
              maxLength={500}
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </div>
          {error ? (
            <p className="text-sm text-destructive" aria-live="polite">
              {error}
            </p>
          ) : null}
          <AlertDialogFooter className="mt-2">
            <AlertDialogCancel>{tc("cancel")}</AlertDialogCancel>
            <Button type="submit" disabled={dispose.isPending || note.trim().length < 5}>
              {dispose.isPending ? tc("saving") : t("submit")}
            </Button>
          </AlertDialogFooter>
        </form>
      </AlertDialogContent>
    </AlertDialog>
  )
}
