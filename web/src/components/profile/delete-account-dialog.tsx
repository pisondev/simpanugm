"use client"

import { useMutation } from "@tanstack/react-query"
import { Trash2 } from "lucide-react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { api } from "@/lib/api/client"

export function DeleteAccountDialog() {
  const t = useTranslations("profile")
  const tc = useTranslations("common")
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const [errorMsg, setErrorMsg] = useState<string | null>(null)

  const del = useMutation({
    mutationFn: async () => {
      setErrorMsg(null)
      const res = await api.POST("/me/deletion")
      if (res.error) {
        const p = res.error as { code?: string; detail?: string }
        if (p.code === "approved_claim_exists" || res.response.status === 409 || res.response.status === 422) {
          throw new Error("Akun tidak dapat dihapus karena memiliki riwayat klaim yang telah disetujui (BR-RET-05).")
        }
        throw new Error(p.detail || "Gagal menghapus akun. Coba lagi nanti.")
      }
      return res.data
    },
    onSuccess: () => {
      toast.success(t("deleteAccountSuccess"))
      setOpen(false)
      router.push("/masuk")
      router.refresh()
    },
    onError: (err: Error) => {
      setErrorMsg(err.message)
    },
  })

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="destructive" />}>
        <Trash2 className="size-4" aria-hidden />
        {t("deleteAccountBtn")}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("deleteAccountConfirmTitle")}</DialogTitle>
          <DialogDescription>{t("deleteAccountConfirmBody")}</DialogDescription>
        </DialogHeader>
        {errorMsg ? (
          <div className="rounded-lg border border-destructive/20 bg-destructive/10 p-3 text-sm text-destructive" role="alert">
            {errorMsg}
          </div>
        ) : null}
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)} disabled={del.isPending}>
            {tc("cancel")}
          </Button>
          <Button variant="destructive" onClick={() => del.mutate()} disabled={del.isPending}>
            {del.isPending ? tc("submitting") : t("deleteAccountBtn")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
