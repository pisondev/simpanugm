"use client"

import { useMutation } from "@tanstack/react-query"
import { Ban, CheckCircle, KeyRound, ShieldAlert, Trash2 } from "lucide-react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"
import type { AdminUser } from "@/lib/api/types"

export function UserModerationMenu({ user }: { user: AdminUser }) {
  const router = useRouter()
  const tc = useTranslations("common")

  const [suspendOpen, setSuspendOpen] = useState(false)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const suspendMutation = useMutation({
    mutationFn: () =>
      call(
        api.POST("/admin/users/{id}/suspend", {
          params: { path: { id: user.id } },
        }),
      ),
    onSuccess: () => {
      toast.success("Akun berhasil dibekukan.")
      setSuspendOpen(false)
      router.refresh()
    },
    onError: (err: Error) => {
      toast.error(err.message || "Gagal membekukan akun.")
    },
  })

  const unsuspendMutation = useMutation({
    mutationFn: () =>
      call(
        api.POST("/admin/users/{id}/unsuspend", {
          params: { path: { id: user.id } },
        }),
      ),
    onSuccess: () => {
      toast.success("Pembekuan akun berhasil dibuka.")
      router.refresh()
    },
    onError: (err: Error) => {
      toast.error(err.message || "Gagal membuka pembekuan akun.")
    },
  })

  const revokeSessionsMutation = useMutation({
    mutationFn: () =>
      call(
        api.POST("/admin/users/{id}/revoke-sessions", {
          params: { path: { id: user.id } },
        }),
      ),
    onSuccess: () => {
      toast.success("Semua sesi pengguna berhasil dicabut.")
      router.refresh()
    },
    onError: (err: Error) => {
      toast.error(err.message || "Gagal mencabut sesi pengguna.")
    },
  })

  const clearRestrictionMutation = useMutation({
    mutationFn: () =>
      call(
        api.POST("/admin/users/{id}/clear-claim-restriction", {
          params: { path: { id: user.id } },
        }),
      ),
    onSuccess: () => {
      toast.success("Pembatasan klaim berhasil dicabut.")
      router.refresh()
    },
    onError: (err: Error) => {
      toast.error(err.message || "Gagal mencabut pembatasan klaim.")
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () =>
      call(
        api.DELETE("/admin/users/{id}", {
          params: { path: { id: user.id } },
        }),
      ),
    onSuccess: () => {
      toast.success("Pengguna berhasil dihapus dan dianonimkan.")
      setDeleteOpen(false)
      router.refresh()
    },
    onError: (err: Error) => {
      toast.error(err.message || "Gagal menghapus pengguna.")
    },
  })

  const isSuspended = user.status === "suspended"
  const isDeleted = user.status === "deleted"
  const hasRestriction = Boolean(user.claim_restricted_at)

  if (isDeleted) {
    return null
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button variant="ghost" size="sm" />}>
          <ShieldAlert className="size-4" aria-hidden />
          <span className="sr-only">Moderasi {user.full_name}</span>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {isSuspended ? (
            <DropdownMenuItem onClick={() => unsuspendMutation.mutate()} disabled={unsuspendMutation.isPending}>
              <CheckCircle className="size-4" aria-hidden />
              Buka Pembekuan
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem onClick={() => setSuspendOpen(true)}>
              <Ban className="size-4" aria-hidden />
              Bekukan Akun
            </DropdownMenuItem>
          )}

          <DropdownMenuItem onClick={() => revokeSessionsMutation.mutate()} disabled={revokeSessionsMutation.isPending}>
            <KeyRound className="size-4" aria-hidden />
            Cabut Sesi Aktif
          </DropdownMenuItem>

          {hasRestriction ? (
            <DropdownMenuItem onClick={() => clearRestrictionMutation.mutate()} disabled={clearRestrictionMutation.isPending}>
              <CheckCircle className="size-4" aria-hidden />
              Cabut Pembatasan Klaim
            </DropdownMenuItem>
          ) : null}

          <DropdownMenuSeparator />

          <DropdownMenuItem onClick={() => setDeleteOpen(true)} className="text-destructive focus:text-destructive">
            <Trash2 className="size-4" aria-hidden />
            Hapus & Anonimkan
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Suspend Dialog */}
      <Dialog open={suspendOpen} onOpenChange={setSuspendOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Bekukan Akun {user.full_name}?</DialogTitle>
            <DialogDescription>Pengguna tidak akan bisa masuk atau mengakses sistem hingga pembekuan dibuka kembali.</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setSuspendOpen(false)}>
              {tc("cancel")}
            </Button>
            <Button variant="destructive" onClick={() => suspendMutation.mutate()} disabled={suspendMutation.isPending}>
              {suspendMutation.isPending ? tc("saving") : "Bekukan Akun"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Dialog */}
      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Hapus & Anonimkan Akun {user.full_name}?</DialogTitle>
            <DialogDescription>
              Tindakan ini permanen. Nama dan email pengguna akan dianonimkan, laporan kehilangan aktif akan ditutup, dan seluruh sesi akan dicabut.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteOpen(false)}>
              {tc("cancel")}
            </Button>
            <Button variant="destructive" onClick={() => deleteMutation.mutate()} disabled={deleteMutation.isPending}>
              {deleteMutation.isPending ? tc("saving") : "Ya, Hapus & Anonimkan"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
