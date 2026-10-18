"use client"

import { useTranslations } from "next-intl"
import { useState } from "react"

import { Field } from "@/components/report/form/field"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"
import type { AdminZone } from "@/lib/api/types"

import { MasterDialog, num } from "./master-dialog"

export function ZoneDialog({ zone }: { zone?: AdminZone }) {
  const t = useTranslations("admin.master")
  const [slug, setSlug] = useState("")
  const [name, setName] = useState("")
  const [lat, setLat] = useState("")
  const [lng, setLng] = useState("")
  return (
    <MasterDialog
      editing={zone?.name}
      title={zone ? t("editTitle", { name: zone.name }) : t("tabs.zones")}
      reset={() => {
        setSlug(zone?.slug ?? "")
        setName(zone?.name ?? "")
        setLat(zone ? String(zone.centroid_lat) : "")
        setLng(zone ? String(zone.centroid_lng) : "")
      }}
      save={() =>
        zone
          ? call(api.PATCH("/admin/zones/{id}", { params: { path: { id: zone.id } }, body: { name, centroid_lat: num(lat), centroid_lng: num(lng) } }))
          : call(api.POST("/admin/zones", { body: { slug, name, centroid_lat: num(lat), centroid_lng: num(lng) } }))
      }
    >
      {(errors) => (
        <>
          {zone ? null : (
            <Field id="zone-slug" label={t("slug")} hint={t("slugHint")} error={errors.slug}>
              {(aria) => <Input {...aria} required maxLength={60} value={slug} onChange={(e) => setSlug(e.target.value)} />}
            </Field>
          )}
          <Field id="zone-name" label={t("name")} error={errors.name}>
            {(aria) => <Input {...aria} required maxLength={80} value={name} onChange={(e) => setName(e.target.value)} />}
          </Field>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field id="zone-lat" label={t("lat")} hint={t("coordinateHint")} error={errors.centroid_lat}>
              {(aria) => <Input {...aria} required inputMode="decimal" value={lat} onChange={(e) => setLat(e.target.value)} />}
            </Field>
            <Field id="zone-lng" label={t("lng")} error={errors.centroid_lng}>
              {(aria) => <Input {...aria} required inputMode="decimal" value={lng} onChange={(e) => setLng(e.target.value)} />}
            </Field>
          </div>
        </>
      )}
    </MasterDialog>
  )
}
