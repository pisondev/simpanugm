"use client"

import { Play, RefreshCw } from "lucide-react"
import { useRouter } from "next/navigation"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import { api } from "@/lib/api/client"
import { toast } from "sonner"

export function ClusterRunTrigger() {
  const [loading, setLoading] = useState(false)
  const router = useRouter()

  const handleRun = async () => {
    setLoading(true)
    try {
      const res = await api.POST("/admin/ai/clusters/runs", {})
      if (res.error) {
        toast.error("Gagal menjalankan klaster.")
      } else {
        toast.success(`Berhasil mengelompokkan laporan ke dalam ${res.data?.cluster_count ?? 0} klaster.`)
        router.refresh()
      }
    } catch {
      toast.error("Gagal menghubungi server.")
    } finally {
      setLoading(false)
    }
  }

  return (
    <Button onClick={handleRun} disabled={loading}>
      {loading ? <RefreshCw className="size-4 animate-spin" aria-hidden /> : <Play className="size-4" aria-hidden />}
      Jalankan Pengelompokan Baru
    </Button>
  )
}
