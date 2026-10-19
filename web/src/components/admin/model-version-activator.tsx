"use client"

import { RefreshCw, RotateCcw } from "lucide-react"
import { useRouter } from "next/navigation"
import { useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { NativeSelect } from "@/components/ui/native-select"
import { api } from "@/lib/api/client"
import type { ModelVersionItem } from "@/lib/api/types"

export function ModelVersionActivator({ allVersions }: { allVersions: ModelVersionItem[] }) {
  const [open, setOpen] = useState(false)
  const [selectedId, setSelectedId] = useState<string>("")
  const [loading, setLoading] = useState(false)
  const router = useRouter()

  const inactiveVersions = allVersions.filter((v) => v.status !== "active")

  const handleActivate = async () => {
    if (!selectedId) return
    setLoading(true)
    try {
      const res = await api.POST("/admin/model-versions/{id}/activate", {
        params: { path: { id: selectedId } },
      })
      if (res.error) {
        toast.error("Gagal mengaktifkan versi model.")
      } else {
        toast.success(`Versi model ${res.data?.name} kini aktif.`)
        setOpen(false)
        router.refresh()
      }
    } catch {
      toast.error("Gagal menghubungi server.")
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button variant="outline" size="sm" />}>
        <RotateCcw className="size-4" aria-hidden />
        Kelola & Rollback Model
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Ganti atau Rollback Versi Model</DialogTitle>
          <DialogDescription>
            Pilih versi model kandidat atau model sebelumnya yang ingin diaktifkan. Model aktif untuk komponen tersebut akan dipensiunkan.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4 py-4">
          <div className="flex flex-col gap-2">
            <label className="text-sm font-medium">Pilih Versi Model</label>
            <NativeSelect value={selectedId} onChange={(e) => setSelectedId(e.target.value)} className="text-sm" aria-label="Pilih model versi">
              <option value="">-- Pilih versi model --</option>
              {inactiveVersions.map((v) => (
                <option key={v.id} value={v.id}>
                  [{v.component}] {v.name} ({v.status})
                </option>
              ))}
            </NativeSelect>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)} disabled={loading}>
            Batal
          </Button>
          <Button onClick={handleActivate} disabled={!selectedId || loading}>
            {loading ? <RefreshCw className="size-4 animate-spin" aria-hidden /> : null}
            Aktifkan Versi Ini
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
