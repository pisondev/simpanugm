"use client"

import { useMutation } from "@tanstack/react-query"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { type ReactNode, useId, useState } from "react"
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
import { NativeSelect } from "@/components/ui/native-select"
import { Textarea } from "@/components/ui/textarea"
import { messageFor } from "@/lib/api/problem"

export type Reason = { value: string; label: string }

type Props = {
  trigger: ReactNode
  title: string
  body: string
  // the free text field, absent when the dialog only confirms
  label?: string
  minLength?: number
  maxLength?: number
  // a choice that comes with the text, for example the rejection reason
  reasons?: Reason[]
  reasonLabel?: string
  confirm: string
  destructive?: boolean
  done: string
  run: (values: { text: string; reason: string }) => Promise<unknown>
}

// one dialog for the officer actions that need a reason: ask for information, reject, revoke, reopen
export function ActionDialog({ trigger, title, body, label, minLength = 0, maxLength = 500, reasons, reasonLabel, confirm, destructive, done, run }: Props) {
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const id = useId()
  const [open, setOpen] = useState(false)
  const [text, setText] = useState("")
  const [reason, setReason] = useState(reasons?.[0]?.value ?? "")
  const [error, setError] = useState<string>()
  const mutation = useMutation({
    mutationFn: () => run({ text: text.trim(), reason }),
    onSuccess: () => {
      setOpen(false)
      setText("")
      toast.success(done)
      router.refresh()
    },
    onError: (err) => setError(messageFor(err, te)),
  })
  const short = label !== undefined && text.trim().length < minLength
  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger render={<Button variant={destructive ? "outline" : "default"} />}>{trigger}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{body}</AlertDialogDescription>
        </AlertDialogHeader>
        <form
          id={id}
          className="flex flex-col gap-3"
          onSubmit={(e) => {
            e.preventDefault()
            setError(undefined)
            mutation.mutate()
          }}
        >
          {reasons ? (
            <div className="flex flex-col gap-2">
              <label htmlFor={`${id}-reason`} className="text-sm font-medium">
                {reasonLabel}
              </label>
              <NativeSelect id={`${id}-reason`} value={reason} onChange={(e) => setReason(e.target.value)}>
                {reasons.map((r) => (
                  <option key={r.value} value={r.value}>
                    {r.label}
                  </option>
                ))}
              </NativeSelect>
            </div>
          ) : null}
          {label !== undefined ? (
            <div className="flex flex-col gap-2">
              <label htmlFor={`${id}-text`} className="text-sm font-medium">
                {label}
              </label>
              <Textarea
                id={`${id}-text`}
                rows={3}
                maxLength={maxLength}
                value={text}
                onChange={(e) => setText(e.target.value)}
                aria-describedby={error ? `${id}-error` : undefined}
              />
            </div>
          ) : null}
          <p id={`${id}-error`} aria-live="polite" className="text-sm text-destructive">
            {error ?? ""}
          </p>
        </form>
        <AlertDialogFooter>
          <AlertDialogCancel>{tc("cancel")}</AlertDialogCancel>
          <Button type="submit" form={id} variant={destructive ? "destructive" : "default"} disabled={mutation.isPending || short}>
            {confirm}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
