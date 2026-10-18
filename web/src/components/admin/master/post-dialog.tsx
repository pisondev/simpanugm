"use client"

import { useTranslations } from "next-intl"
import { useState } from "react"

import { Field } from "@/components/report/form/field"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"
import type { AdminLocation, AdminPost } from "@/lib/api/types"
import { parseRanges, rangesText, weekdays } from "@/lib/hours"

import { MasterDialog } from "./master-dialog"

type Days = Record<(typeof weekdays)[number], string>

function textOf(hours: Record<string, string[][]> | undefined): Days {
  const out = {} as Days
  for (const d of weekdays) {
    out[d] = rangesText(hours?.[d] ?? [])
  }
  return out
}

export function PostDialog({ post, locations }: { post?: AdminPost; locations: AdminLocation[] }) {
  const t = useTranslations("admin.master")
  const th = useTranslations("help")
  const [slug, setSlug] = useState("")
  const [name, setName] = useState("")
  const [location, setLocation] = useState("")
  const [phone, setPhone] = useState("")
  const [days, setDays] = useState<Days>(textOf(undefined))
  const hours = () => Object.fromEntries(weekdays.map((d) => [d, parseRanges(days[d]) ?? []]))
  return (
    <MasterDialog
      editing={post?.name}
      title={post ? t("editTitle", { name: post.name }) : t("tabs.posts")}
      reset={() => {
        setSlug(post?.slug ?? "")
        setName(post?.name ?? "")
        setLocation(post?.location_id ?? locations.find((l) => l.active)?.id ?? "")
        setPhone(post?.phone ?? "")
        setDays(textOf(post?.opening_hours))
      }}
      check={() => {
        const bad = weekdays.find((d) => parseRanges(days[d]) === null)
        return bad ? t("posts.hoursInvalid", { day: th(`weekdays.${bad}`) }) : undefined
      }}
      save={() =>
        post
          ? call(api.PATCH("/admin/posts/{id}", { params: { path: { id: post.id } }, body: { name, location_id: location, phone, opening_hours: hours() } }))
          : call(api.POST("/admin/posts", { body: { slug, name, location_id: location, phone: phone || undefined, opening_hours: hours() } }))
      }
    >
      {(errors) => (
        <>
          {post ? null : (
            <Field id="post-slug" label={t("slug")} hint={t("slugHint")} error={errors.slug}>
              {(aria) => <Input {...aria} required maxLength={60} value={slug} onChange={(e) => setSlug(e.target.value)} />}
            </Field>
          )}
          <Field id="post-name" label={t("name")} error={errors.name}>
            {(aria) => <Input {...aria} required maxLength={100} value={name} onChange={(e) => setName(e.target.value)} />}
          </Field>
          <Field id="post-location" label={t("posts.location")} error={errors.location_id}>
            {(aria) => (
              <NativeSelect {...aria} value={location} onChange={(e) => setLocation(e.target.value)}>
                {locations
                  .filter((l) => l.active || l.id === post?.location_id)
                  .map((l) => (
                    <option key={l.id} value={l.id}>
                      {l.name} | {l.zone_name}
                    </option>
                  ))}
              </NativeSelect>
            )}
          </Field>
          <Field id="post-phone" label={t("posts.phone")} error={errors.phone}>
            {(aria) => <Input {...aria} type="tel" maxLength={30} value={phone} onChange={(e) => setPhone(e.target.value)} />}
          </Field>
          <fieldset className="flex flex-col gap-2" aria-describedby="post-hours-hint">
            <legend className="mb-1 text-sm font-medium">{t("posts.hours")}</legend>
            <p id="post-hours-hint" className="text-sm text-muted-foreground">
              {t("posts.hoursHint")}
            </p>
            {errors.opening_hours ? <p className="text-sm text-destructive">{errors.opening_hours}</p> : null}
            {weekdays.map((d) => (
              <div key={d} className="grid grid-cols-[6rem_1fr] items-center gap-2">
                <label htmlFor={`post-hours-${d}`} className="text-sm">
                  {th(`weekdays.${d}`)}
                </label>
                <Input
                  id={`post-hours-${d}`}
                  value={days[d]}
                  placeholder={t("posts.closed")}
                  onChange={(e) => setDays((prev) => ({ ...prev, [d]: e.target.value }))}
                />
              </div>
            ))}
          </fieldset>
        </>
      )}
    </MasterDialog>
  )
}
