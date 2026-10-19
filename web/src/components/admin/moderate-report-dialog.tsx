"use client"

import { useMutation } from "@tanstack/react-query"
import { Eye, EyeOff } from "lucide-react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"

export function ModerateReportDialog({ type, id, hidden = false }: { type: "lost" | "found"; id: string; hidden?: boolean }) {
  const router = useRouter()
  const tc = useTranslations("common")
  const [open, setOpen] = useState(false)
  const [reason, setReason] = useState("")

  const hideMutation = useMutation({
    mutationFn: () =>
      call(
        api.POST("/admin/reports/{type}/{id}/hide", {
          params: { path: { type, id } },
          body: { reason },
        }),
      ),
    onSuccess: () => {
      toast.success("Laporan berhasil disembunyikan.")
      setOpen(false)
      setReason("")
      router.refresh()
    },
    onError: (err: Error) => {
      toast.error(err.message || "Gagal menyembunyikan laporan.")
    },
  })

  const unhideMutation = useMutation({
    mutationFn: () =>
      call(
        api.POST("/admin/reports/{type}/{id}/unhide", {
          params: { path: { type, id } },
        }),
      ),
    onSuccess: () => {
      toast.success("Laporan berhasil ditampilkan kembali.")
      router.refresh()
    },
    onError: (err: Error) => {
      toast.error(err.message || "Gagal menampilkan laporan.")
    },
  })

  if (hidden) {
    return (
      <Button variant="outline" size="sm" onClick={() => unhideMutation.mutate()} disabled={unhideMutation.isPending}>
        <Eye className="size-4" aria-hidden />
        Tampilkan Kembali
      </Button>
    )
  }

  return (
    <>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogTrigger render={<Button variant="outline" size="sm" />}>
          <EyeOff className="size-4" aria-hidden />
          Sembunyikan
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Sembunyikan Laporan?</DialogTitle>
            <DialogDescription>Laporan yang disembunyikan tidak akan muncul dalam pencarian publik, pencocokan AI, atau daftar laporan.</DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-2 py-2">
            <label htmlFor={`hide-reason-${id}`} className="text-sm font-medium">
              Alasan penyembunyian
            </label>
            <Input
              id={`hide-reason-${id}`}
              placeholder="Contoh: Informasi pribadi sensitif atau konten tidak pantas"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setOpen(false)}>
              {tc("cancel")}
            </Button>
            <Button variant="destructive" onClick={() => hideMutation.mutate()} disabled={hideMutation.isPending || !reason.trim()}>
              {hideMutation.isPending ? tc("saving") : "Sembunyikan Laporan"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
