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
import { api } from "@/lib/api/client"
import { call, messageFor } from "@/lib/api/problem"

export type MasterKind = "category" | "zone" | "location" | "post"

function setActive(kind: MasterKind, id: string, active: boolean): Promise<unknown> {
  const params = { path: { id } }
  switch (kind) {
    case "category":
      return call(api.PATCH("/admin/categories/{id}", { params, body: { active } }))
    case "zone":
      return call(api.PATCH("/admin/zones/{id}", { params, body: { active } }))
    case "location":
      return call(api.PATCH("/admin/locations/{id}", { params, body: { active } }))
    case "post":
      return call(api.PATCH("/admin/posts/{id}", { params, body: { active } }))
  }
}

// FR-MD-07: switching a row off hides it from pickers and keeps it on the records that use it
export function ToggleActive({ kind, id, name, active }: { kind: MasterKind; id: string; name: string; active: boolean }) {
  const t = useTranslations("admin.master")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const toggle = useMutation({
    mutationFn: () => setActive(kind, id, !active),
    onSuccess: () => {
      setOpen(false)
      toast.success(t("statusChanged"))
      router.refresh()
    },
    onError: (err) => toast(messageFor(err, te)),
  })
  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger render={<Button variant="ghost" size="sm" />}>{active ? t("deactivate") : t("activate")}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{active ? t("deactivateTitle", { name }) : t("activateTitle", { name })}</AlertDialogTitle>
          <AlertDialogDescription>{active ? t("deactivateBody", { name }) : t("activateBody", { name })}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{tc("cancel")}</AlertDialogCancel>
          <Button variant={active ? "destructive" : "default"} onClick={() => toggle.mutate()} disabled={toggle.isPending}>
            {active ? t("confirmDeactivate") : t("confirmActivate")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
