"use client"

import { useEffect, useRef } from "react"
import mapboxgl from "mapbox-gl"

import type { HotspotLocation } from "@/lib/api/types"

import { hotspotMarkerStyle } from "./hotspot-marker-style"

interface CampusHotspotMapClientProps {
  locations: HotspotLocation[]
  selectedType: "lost" | "found" | "all"
  mapboxToken?: string
  onSelectLocation?: (locationId: string) => void
}

const CAMPUS_BOUNDS: mapboxgl.LngLatBoundsLike = [
  [110.3716393, -7.776005],
  [110.3863, -7.764283],
]
const CAMPUS_CENTER: [number, number] = [110.3782801, -7.7693293]
const MAP_STYLE = "mapbox://styles/mapbox/streets-v12"

function countFor(location: HotspotLocation, selectedType: CampusHotspotMapClientProps["selectedType"]): number {
  if (selectedType === "lost") return location.lost_count
  if (selectedType === "found") return location.found_count
  return location.total_count
}

function escapeHtml(value: string): string {
  return value.replace(/[&<>"']/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[character] ?? character)
}

function popupHtml(location: HotspotLocation): string {
  return `
    <strong>${escapeHtml(location.location_name)}</strong>
    <div>${escapeHtml(location.zone_name)}</div>
    <hr />
    <div>Hilang: ${escapeHtml(location.display_lost)}</div>
    <div>Temuan: ${escapeHtml(location.display_found)}</div>
    <div>Total: ${escapeHtml(location.display_total)}</div>
    <div>Tingkat kembali: ${(location.return_rate * 100).toFixed(0)}%</div>
    <hr />
    <small>Koordinat: ${location.lat.toFixed(6)}, ${location.lng.toFixed(6)}</small>
  `
}

function markerElement(location: HotspotLocation, count: number, maxCount: number): HTMLButtonElement {
  const element = document.createElement("button")
  const style = hotspotMarkerStyle(count, maxCount)

  element.type = "button"
  element.title = location.location_name
  element.setAttribute("aria-label", `${location.location_name}, ${count < 3 ? "kurang dari 3" : `${count}`} laporan`)
  element.textContent = count < 3 ? "<3" : `${count}`
  element.style.width = `${style.diameter}px`
  element.style.height = `${style.diameter}px`
  element.style.border = "2px solid white"
  element.style.borderRadius = "9999px"
  element.style.background = style.background
  element.style.boxShadow = "0 2px 8px rgb(15 23 42 / 0.35)"
  element.style.color = "white"
  element.style.cursor = "pointer"
  element.style.fontSize = "11px"
  element.style.fontWeight = "700"
  element.style.padding = "0"

  return element
}

export function CampusHotspotMapClient({ locations, selectedType, mapboxToken, onSelectLocation }: CampusHotspotMapClientProps) {
  const mapContainerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!mapContainerRef.current || !mapboxToken) return

    mapboxgl.accessToken = mapboxToken
    const map = new mapboxgl.Map({
      container: mapContainerRef.current,
      style: MAP_STYLE,
      center: CAMPUS_CENTER,
      maxBounds: CAMPUS_BOUNDS,
      minZoom: 14,
      maxZoom: 19,
      attributionControl: true,
    })
    const maxCount = Math.max(...locations.map((location) => countFor(location, selectedType)), 1)
    const markerElements: HTMLButtonElement[] = []

    map.addControl(new mapboxgl.NavigationControl(), "top-right")
    map.addControl(new mapboxgl.ScaleControl({ maxWidth: 120, unit: "metric" }), "bottom-left")
    map.on("load", () => {
      map.fitBounds(CAMPUS_BOUNDS, { padding: 24, maxZoom: 16 })

      locations.forEach((location) => {
        const count = countFor(location, selectedType)
        const element = markerElement(location, count, maxCount)
        const marker = new mapboxgl.Marker({ element, anchor: "center" })
          .setLngLat([location.lng, location.lat])
          .setPopup(new mapboxgl.Popup({ offset: 18 }).setHTML(popupHtml(location)))
          .addTo(map)

        const selectLocation = () => {
          markerElements.forEach((markerElement) => {
            markerElement.style.borderColor = "white"
          })
          element.style.borderColor = "#0f172a"
          onSelectLocation?.(location.location_id)
        }

        element.addEventListener("click", selectLocation)
        element.addEventListener("keydown", (event) => {
          if (event.key === "Enter" || event.key === " ") {
            event.preventDefault()
            selectLocation()
            marker.togglePopup()
          }
        })
        markerElements.push(element)
      })
    })

    return () => map.remove()
  }, [locations, mapboxToken, onSelectLocation, selectedType])

  if (!mapboxToken) {
    return (
      <div className="flex size-full min-h-[360px] items-center justify-center bg-muted p-6 text-center text-sm text-muted-foreground" role="status">
        Token Mapbox belum dikonfigurasi untuk lingkungan ini.
      </div>
    )
  }

  return <div ref={mapContainerRef} className="size-full min-h-[360px]" aria-label="Peta interaktif kampus UGM" />
}
