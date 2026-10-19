"use client"

import { useMutation } from "@tanstack/react-query"
import { CheckCircle2 } from "lucide-react"
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

type Props = { code: string }

// US-11, US-31, FR-LOST-14: owner confirms receipt after handover
export function ConfirmReceiptDialog({ code }: Props) {
  const t = useTranslations("report.confirmReceipt")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const confirm = useMutation({
    mutationFn: () => call(api.POST("/lost-reports/{id}/confirm-receipt", { params: { path: { id: code } } })),
    onSuccess: () => {
      setOpen(false)
      toast.success(t("done"))
      router.refresh()
    },
    onError: (err) => toast(messageFor(err, te)),
  })

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger render={<Button />}>
        <CheckCircle2 aria-hidden />
        {t("trigger")}
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("title")}</AlertDialogTitle>
          <AlertDialogDescription>{t("body")}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{tc("cancel")}</AlertDialogCancel>
          <Button onClick={() => confirm.mutate()} disabled={confirm.isPending}>
            {confirm.isPending ? tc("saving") : t("confirm")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
