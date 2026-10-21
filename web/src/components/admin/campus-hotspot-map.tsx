"use client"

import dynamic from "next/dynamic"
import { MapPin } from "lucide-react"

import type { HotspotLocation } from "@/lib/api/types"

interface CampusHotspotMapProps {
  locations: HotspotLocation[]
  mapboxToken?: string
  selectedType: "lost" | "found" | "all"
  onSelectLocation?: (locationId: string) => void
}

function displayCount(location: HotspotLocation, selectedType: CampusHotspotMapProps["selectedType"]): string {
  const display = selectedType === "lost" ? location.display_lost : selectedType === "found" ? location.display_found : location.display_total
  return display.replace("< ", "<")
}

const CampusHotspotMapClient = dynamic(() => import("./campus-hotspot-map-client").then((module) => module.CampusHotspotMapClient), {
  ssr: false,
  loading: () => (
    <div className="flex size-full min-h-[360px] items-center justify-center bg-muted text-sm text-muted-foreground" role="status">
      Memuat peta kampus...
    </div>
  ),
})

export function CampusHotspotMap({ locations, mapboxToken, selectedType, onSelectLocation }: CampusHotspotMapProps) {
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

      <div className="relative aspect-[5/3] min-h-[360px] w-full overflow-hidden" role="region" aria-label="Peta interaktif kampus UGM">
        <CampusHotspotMapClient locations={locations} mapboxToken={mapboxToken} selectedType={selectedType} onSelectLocation={onSelectLocation} />
      </div>
      <div className="sr-only" aria-label="Ringkasan laporan per lokasi">
        {locations.map((location) => (
          <span key={location.location_id}>
            <span>{location.location_name}: </span>
            <span>{displayCount(location, selectedType)}</span>
          </span>
        ))}
      </div>
    </div>
  )
}
