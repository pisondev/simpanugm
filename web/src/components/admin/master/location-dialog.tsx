"use client"

import { useTranslations } from "next-intl"
import { useState } from "react"

import { Field } from "@/components/report/form/field"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"
import type { AdminLocation, AdminZone, LocationKind } from "@/lib/api/types"

import { MasterDialog, num } from "./master-dialog"

export const locationKinds: LocationKind[] = ["building", "outdoor", "facility", "transport", "parking"]

function splitAliases(text: string): string[] {
  return text
    .split(",")
    .map((a) => a.trim())
    .filter(Boolean)
}

export function LocationDialog({ location, zones }: { location?: AdminLocation; zones: AdminZone[] }) {
  const t = useTranslations("admin.master")
  const [slug, setSlug] = useState("")
  const [name, setName] = useState("")
  const [zone, setZone] = useState("")
  const [kind, setKind] = useState<LocationKind>("building")
  const [aliases, setAliases] = useState("")
  const [lat, setLat] = useState("")
  const [lng, setLng] = useState("")
  return (
    <MasterDialog
      editing={location?.name}
      title={location ? t("editTitle", { name: location.name }) : t("tabs.locations")}
      reset={() => {
        setSlug(location?.slug ?? "")
        setName(location?.name ?? "")
        setZone(location?.zone_id ?? zones.find((z) => z.active)?.id ?? "")
        setKind(location?.kind ?? "building")
        setAliases(location?.aliases.join(", ") ?? "")
        setLat(location ? String(location.lat) : "")
        setLng(location ? String(location.lng) : "")
      }}
      save={() => {
        const body = { name, zone_id: zone, kind, aliases: splitAliases(aliases), lat: num(lat), lng: num(lng) }
        return location
          ? call(api.PATCH("/admin/locations/{id}", { params: { path: { id: location.id } }, body }))
          : call(api.POST("/admin/locations", { body: { ...body, slug } }))
      }}
    >
      {(errors) => (
        <>
          {location ? null : (
            <Field id="loc-slug" label={t("slug")} hint={t("slugHint")} error={errors.slug}>
              {(aria) => <Input {...aria} required maxLength={60} value={slug} onChange={(e) => setSlug(e.target.value)} />}
            </Field>
          )}
          <Field id="loc-name" label={t("name")} error={errors.name}>
            {(aria) => <Input {...aria} required maxLength={100} value={name} onChange={(e) => setName(e.target.value)} />}
          </Field>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field id="loc-zone" label={t("locations.zone")} error={errors.zone_id}>
              {(aria) => (
                <NativeSelect {...aria} value={zone} onChange={(e) => setZone(e.target.value)}>
                  {zones.map((z) => (
                    <option key={z.id} value={z.id}>
                      {z.active ? z.name : `${z.name} (${t("inactive")})`}
                    </option>
                  ))}
                </NativeSelect>
              )}
            </Field>
            <Field id="loc-kind" label={t("locations.kind")} error={errors.kind}>
              {(aria) => (
                <NativeSelect {...aria} value={kind} onChange={(e) => setKind(e.target.value as LocationKind)}>
                  {locationKinds.map((k) => (
                    <option key={k} value={k}>
                      {t(`locations.kinds.${k}`)}
                    </option>
                  ))}
                </NativeSelect>
              )}
            </Field>
          </div>
          <Field id="loc-aliases" label={t("locations.aliases")} hint={t("locations.aliasesHint")} error={errors.aliases}>
            {(aria) => <Input {...aria} value={aliases} onChange={(e) => setAliases(e.target.value)} />}
          </Field>
          <div className="grid gap-4 sm:grid-cols-2">
            <Field id="loc-lat" label={t("lat")} hint={t("coordinateHint")} error={errors.lat}>
              {(aria) => <Input {...aria} required inputMode="decimal" value={lat} onChange={(e) => setLat(e.target.value)} />}
            </Field>
            <Field id="loc-lng" label={t("lng")} error={errors.lng}>
              {(aria) => <Input {...aria} required inputMode="decimal" value={lng} onChange={(e) => setLng(e.target.value)} />}
            </Field>
          </div>
        </>
      )}
    </MasterDialog>
  )
}
