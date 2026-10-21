"use client"

import { Fragment, useState } from "react"
import { Circle, CircleMarker, MapContainer, Popup, TileLayer, Tooltip, ZoomControl } from "react-leaflet"
import type { LatLngBoundsExpression } from "leaflet"

import type { HotspotLocation } from "@/lib/api/types"

interface CampusHotspotMapClientProps {
  locations: HotspotLocation[]
  selectedType: "lost" | "found" | "all"
  onSelectLocation?: (locationId: string) => void
}

const CAMPUS_BOUNDS: LatLngBoundsExpression = [
  [-7.776005, 110.3716393],
  [-7.764283, 110.3863],
]
const CAMPUS_CENTER: [number, number] = [-7.7693293, 110.3782801]
const TILE_URL = process.env.NEXT_PUBLIC_MAP_TILE_URL ?? "https://tile.openstreetmap.org/{z}/{x}/{y}.png"

function countFor(location: HotspotLocation, selectedType: CampusHotspotMapClientProps["selectedType"]): number {
  if (selectedType === "lost") return location.lost_count
  if (selectedType === "found") return location.found_count
  return location.total_count
}

function markerColour(count: number): string {
  if (count >= 10) return "#e11d48"
  if (count >= 4) return "#d97706"
  if (count === 0) return "#64748b"
  return "#059669"
}

export function CampusHotspotMapClient({ locations, selectedType, onSelectLocation }: CampusHotspotMapClientProps) {
  const [activeLocationId, setActiveLocationId] = useState<string | null>(null)
  const maxCount = Math.max(...locations.map((location) => countFor(location, selectedType)), 1)

  return (
    <MapContainer
      bounds={CAMPUS_BOUNDS}
      boundsOptions={{ padding: [16, 16] }}
      center={CAMPUS_CENTER}
      className="size-full min-h-[360px]"
      maxZoom={19}
      minZoom={14}
      scrollWheelZoom
      zoomControl={false}
    >
      <TileLayer
        attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
        url={TILE_URL}
      />
      <ZoomControl position="topright" />
      {locations.map((location) => {
        const count = countFor(location, selectedType)
        const colour = markerColour(count)
        const radius = count === 0 ? 6 : Math.min(26, 8 + (count / maxCount) * 18)
        const active = activeLocationId === location.location_id

        return (
          <Fragment key={location.location_id}>
            {count >= 3 ? (
              <Circle
                center={[location.lat, location.lng]}
                pathOptions={{ color: colour, fillColor: colour, fillOpacity: 0.12, weight: 1 }}
                radius={Math.min(160, 45 + count * 12)}
              />
            ) : null}
            <CircleMarker
              center={[location.lat, location.lng]}
              eventHandlers={{
                click: () => {
                  setActiveLocationId(location.location_id)
                  onSelectLocation?.(location.location_id)
                },
              }}
              pathOptions={{ color: active ? "#0f172a" : colour, fillColor: colour, fillOpacity: 0.86, weight: active ? 3 : 2 }}
              radius={radius}
            >
              <Tooltip
                permanent
                direction="center"
                className="!border-0 !bg-transparent !font-bold !text-white !shadow-none"
                offset={[0, 0]}
              >
                {count < 3 ? "<3" : count}
              </Tooltip>
              <Tooltip direction="top" offset={[0, -8]}>
                {location.location_name}
              </Tooltip>
              <Popup>
                <div className="min-w-52 space-y-2 text-sm">
                  <div>
                    <p className="font-semibold">{location.location_name}</p>
                    <p className="text-xs text-slate-500">{location.zone_name}</p>
                  </div>
                  <div className="grid grid-cols-2 gap-x-4 gap-y-1 border-t pt-2 text-xs">
                    <span>Hilang</span>
                    <strong>{location.display_lost}</strong>
                    <span>Temuan</span>
                    <strong>{location.display_found}</strong>
                    <span>Total</span>
                    <strong>{location.display_total}</strong>
                    <span>Tingkat kembali</span>
                    <strong>{(location.return_rate * 100).toFixed(0)}%</strong>
                  </div>
                  <p className="border-t pt-2 text-[11px] text-slate-500">
                    Koordinat: {location.lat.toFixed(6)}, {location.lng.toFixed(6)}
                  </p>
                </div>
              </Popup>
            </CircleMarker>
          </Fragment>
        )
      })}
    </MapContainer>
  )
}
