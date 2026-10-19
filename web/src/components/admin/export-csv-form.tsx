"use client"

import { Download } from "lucide-react"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

export function ExportCsvForm() {
  const [month, setMonth] = useState(() => {
    const d = new Date()
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`
  })

  const handleDownload = () => {
    if (!month) return
    window.open(`/api/v1/analytics/export?month=${encodeURIComponent(month)}`, "_blank")
  }

  return (
    <div className="flex items-center gap-2">
      <Input type="month" value={month} onChange={(e) => setMonth(e.target.value)} className="w-36 text-sm" aria-label="Pilih bulan untuk ekspor CSV" />
      <Button variant="outline" onClick={handleDownload} disabled={!month}>
        <Download className="size-4" aria-hidden />
        Unduh CSV
      </Button>
    </div>
  )
}
