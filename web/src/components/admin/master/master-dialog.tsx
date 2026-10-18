"use client"

import { useMutation } from "@tanstack/react-query"
import { Pencil, Plus } from "lucide-react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { type ReactNode, useId, useState } from "react"
import { toast } from "sonner"

import { FormAlert } from "@/components/auth/form-alert"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { fieldErrors, messageFor } from "@/lib/api/problem"

type Props = {
  // edit dialogs name the row they edit, create dialogs carry their own label
  editing?: string
  createLabel?: string
  title: string
  description?: string
  save: () => Promise<unknown>
  // fills the form again each time the dialog opens, so a cancelled edit leaves nothing behind
  reset: () => void
  // a client side check that stops the request, it returns the message to show
  check?: () => string | undefined
  children: (errors: Record<string, string>) => ReactNode
}

// one dialog shape for every master data form: field errors from the api land on their fields, the rest above the form
export function MasterDialog({ editing, createLabel, title, description, save, reset, check, children }: Props) {
  const t = useTranslations("admin.master")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const [local, setLocal] = useState<string>()
  const formId = useId()
  const mutation = useMutation({
    mutationFn: save,
    onSuccess: () => {
      setOpen(false)
      toast.success(editing ? t("saved") : t("created"))
      router.refresh()
    },
  })
  const errors = mutation.error ? fieldErrors(mutation.error) : {}
  const general = local ?? (mutation.error && Object.keys(errors).length === 0 ? messageFor(mutation.error, te) : undefined)
  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        setOpen(v)
        if (v) {
          reset()
          setLocal(undefined)
          mutation.reset()
        }
      }}
    >
      {editing ? (
        <DialogTrigger render={<Button variant="ghost" size="sm" aria-label={t("editTitle", { name: editing })} />}>
          <Pencil aria-hidden />
          {t("edit")}
        </DialogTrigger>
      ) : (
        <DialogTrigger render={<Button />}>
          <Plus aria-hidden />
          {createLabel ?? t("add")}
        </DialogTrigger>
      )}
      <DialogContent className="max-h-[90dvh] overflow-y-auto sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          {description ? <DialogDescription>{description}</DialogDescription> : null}
        </DialogHeader>
        <form
          id={formId}
          className="flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault()
            const problem = check?.()
            setLocal(problem)
            if (!problem) {
              mutation.mutate()
            }
          }}
        >
          <FormAlert>{general}</FormAlert>
          {children(errors)}
        </form>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            {tc("cancel")}
          </Button>
          <Button type="submit" form={formId} disabled={mutation.isPending}>
            {tc("save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function num(v: string): number {
  return Number(v.replace(",", "."))
}
