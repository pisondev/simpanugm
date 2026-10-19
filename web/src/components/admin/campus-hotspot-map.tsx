"use client"

import { MapPin } from "lucide-react"
import { useState } from "react"

import { Badge } from "@/components/ui/badge"
import type { HotspotLocation } from "@/lib/api/types"

interface CampusHotspotMapProps {
  locations: HotspotLocation[]
  selectedType: "lost" | "found" | "all"
  onSelectLocation?: (locationId: string) => void
}

// UGM Campus bounds approximately
// Lat: -7.780 to -7.760
// Lng: 110.365 to 110.390
const LAT_MIN = -7.782
const LAT_MAX = -7.76
const LNG_MIN = 110.365
const LNG_MAX = 110.39

export function CampusHotspotMap({ locations, selectedType, onSelectLocation }: CampusHotspotMapProps) {
  const [activeLoc, setActiveLoc] = useState<HotspotLocation | null>(null)

  // Map lat/lng to SVG 0-1000 width, 0-600 height
  const project = (lat: number, lng: number) => {
    const x = ((lng - LNG_MIN) / (LNG_MAX - LNG_MIN)) * 1000
    // Lat increases north (up), so invert Y
    const y = ((LAT_MAX - lat) / (LAT_MAX - LAT_MIN)) * 600
    return { x: Math.max(40, Math.min(960, x)), y: Math.max(40, Math.min(560, y)) }
  }

  const getCount = (loc: HotspotLocation) => {
    if (selectedType === "lost") return loc.lost_count
    if (selectedType === "found") return loc.found_count
    return loc.total_count
  }

  const maxCount = Math.max(...locations.map(getCount), 1)

  return (
    <div className="relative flex flex-col overflow-hidden rounded-xl border bg-card shadow-sm">
      <div className="flex items-center justify-between border-b bg-muted/40 px-4 py-2.5 text-xs">
        <div className="flex items-center gap-2">
          <MapPin className="size-4 text-primary" aria-hidden />
          <span className="font-semibold">Peta Kampus UGM (Hotspot Sebaran)</span>
        </div>
        <div className="flex items-center gap-3 text-muted-foreground">
          <div className="flex items-center gap-1.5">
            <span className="inline-block size-2.5 rounded-full bg-emerald-500" />
            <span>Rendah</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="inline-block size-2.5 rounded-full bg-amber-500" />
            <span>Sedang</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="inline-block size-2.5 rounded-full bg-rose-500" />
            <span>Tinggi</span>
          </div>
        </div>
      </div>

      <div className="relative aspect-[5/3] min-h-[360px] w-full overflow-hidden bg-slate-50 dark:bg-slate-950">
        <svg viewBox="0 0 1000 600" className="size-full select-none" role="img" aria-label="Peta interaktif kampus UGM">
          {/* Background Campus Zones Outline */}
          <rect x="150" y="80" width="300" height="200" rx="16" fill="currentColor" className="text-slate-200/50 dark:text-slate-900/50" />
          <text x="170" y="110" className="fill-slate-400 text-xs font-semibold dark:fill-slate-600">
            Klaster Vokasi & Pertanian (Sekip)
          </text>

          <rect x="500" y="60" width="360" height="220" rx="16" fill="currentColor" className="text-slate-200/50 dark:text-slate-900/50" />
          <text x="520" y="90" className="fill-slate-400 text-xs font-semibold dark:fill-slate-600">
            Klaster Sains & Teknologi (Teknik / MIPA)
          </text>

          <rect x="350" y="240" width="300" height="180" rx="16" fill="currentColor" className="border border-primary/10 text-primary/5" />
          <text x="370" y="270" className="fill-primary/60 text-xs font-semibold">
            Pusat Kampus (Balairung & Perpus)
          </text>

          <rect x="200" y="360" width="320" height="190" rx="16" fill="currentColor" className="text-slate-200/50 dark:text-slate-900/50" />
          <text x="220" y="390" className="fill-slate-400 text-xs font-semibold dark:fill-slate-600">
            Klaster Sosiohumaniora (FIB, FISIPOL, FEB)
          </text>

          <rect x="560" y="340" width="300" height="200" rx="16" fill="currentColor" className="text-slate-200/50 dark:text-slate-900/50" />
          <text x="580" y="370" className="fill-slate-400 text-xs font-semibold dark:fill-slate-600">
            Klaster Kesehatan & GSP (Kedokteran / RS)
          </text>

          {/* Location Nodes */}
          {locations.map((loc) => {
            const { x, y } = project(loc.lat, loc.lng)
            const count = getCount(loc)
            const radius = count === 0 ? 6 : Math.min(26, 8 + (count / maxCount) * 18)

            let fillClass = "fill-emerald-500/80 stroke-emerald-600"
            if (count >= 10) {
              fillClass = "fill-rose-500/80 stroke-rose-600 animate-pulse"
            } else if (count >= 4) {
              fillClass = "fill-amber-500/80 stroke-amber-600"
            } else if (count === 0) {
              fillClass = "fill-slate-400/50 stroke-slate-500"
            }

            const isHovered = activeLoc?.location_id === loc.location_id

            return (
              <g
                key={loc.location_id}
                className="cursor-pointer transition-transform hover:scale-110 focus:outline-none"
                onClick={() => {
                  setActiveLoc(loc)
                  onSelectLocation?.(loc.location_id)
                }}
                onMouseEnter={() => setActiveLoc(loc)}
              >
                {/* Heat aura if count >= 3 */}
                {count >= 3 ? <circle cx={x} cy={y} r={radius * 1.8} fill="currentColor" className="text-primary/10" /> : null}

                <circle cx={x} cy={y} r={radius} className={`${fillClass} stroke-2 transition-all ${isHovered ? "stroke-foreground stroke-[3px]" : ""}`} />

                {count > 0 ? (
                  <text x={x} y={y + 3.5} textAnchor="middle" className="pointer-events-none fill-white font-mono text-[10px] font-bold">
                    {count < 3 ? "<3" : count}
                  </text>
                ) : null}
              </g>
            )
          })}
        </svg>

        {/* Floating Details Tooltip / Card */}
        {activeLoc ? (
          <div className="absolute bottom-4 left-4 z-10 w-72 rounded-lg border bg-background/95 p-3 shadow-lg backdrop-blur">
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="text-sm font-semibold">{activeLoc.location_name}</p>
                <p className="text-xs text-muted-foreground">{activeLoc.zone_name}</p>
              </div>
              <Badge variant="outline" className="text-[11px]">
                {activeLoc.display_total} Laporan
              </Badge>
            </div>
            <div className="mt-2 grid grid-cols-2 gap-2 border-t pt-2 text-xs">
              <div>
                <span className="text-muted-foreground">Hilang: </span>
                <span className="font-medium text-destructive">{activeLoc.display_lost}</span>
              </div>
              <div>
                <span className="text-muted-foreground">Temuan: </span>
                <span className="font-medium text-primary">{activeLoc.display_found}</span>
              </div>
              <div className="col-span-2 flex justify-between">
                <span className="text-muted-foreground">Tingkat Kembali:</span>
                <span className="font-semibold text-emerald-600">{(activeLoc.return_rate * 100).toFixed(0)}%</span>
              </div>
            </div>
          </div>
        ) : null}
      </div>
    </div>
  )
}
