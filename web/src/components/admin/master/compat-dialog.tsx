"use client"

import { useTranslations } from "next-intl"
import { useState } from "react"

import { Field } from "@/components/report/form/field"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"
import type { AdminCategory, CompatibilityPair } from "@/lib/api/types"

import { MasterDialog, num } from "./master-dialog"

// FR-MD-02: a pair is stored once, switching it off keeps it out of matching and away from the seed
export function CompatDialog({ leaves, pair }: { leaves: AdminCategory[]; pair?: CompatibilityPair }) {
  const t = useTranslations("admin.master")
  const [a, setA] = useState("")
  const [b, setB] = useState("")
  const [factor, setFactor] = useState("0.85")
  const [active, setActive] = useState(true)
  const options = leaves.map((l) => (
    <option key={l.id} value={l.id}>
      {l.parent_name ? `${l.name} | ${l.parent_name}` : l.name}
    </option>
  ))
  return (
    <MasterDialog
      editing={pair ? t("categories.pairName", { a: pair.name_a, b: pair.name_b }) : undefined}
      createLabel={t("categories.compatAdd")}
      title={t("categories.compatTitle")}
      description={t("categories.compatLead")}
      reset={() => {
        setA(pair?.category_a ?? leaves[0]?.id ?? "")
        setB(pair?.category_b ?? leaves[1]?.id ?? "")
        setFactor(pair ? String(pair.factor) : "0.85")
        setActive(pair?.active ?? true)
      }}
      save={() => call(api.PUT("/admin/category-compatibility", { body: { category_a: a, category_b: b, factor: num(factor), active } }))}
    >
      {(errors) => (
        <>
          <Field id="compat-a" label={t("categories.compatA")} error={errors.category_a}>
            {(aria) => (
              <NativeSelect {...aria} value={a} onChange={(e) => setA(e.target.value)} disabled={pair !== undefined}>
                {options}
              </NativeSelect>
            )}
          </Field>
          <Field id="compat-b" label={t("categories.compatB")} error={errors.category_b}>
            {(aria) => (
              <NativeSelect {...aria} value={b} onChange={(e) => setB(e.target.value)} disabled={pair !== undefined}>
                {options}
              </NativeSelect>
            )}
          </Field>
          <Field id="compat-factor" label={t("categories.factor")} error={errors.factor}>
            {(aria) => <Input {...aria} required inputMode="decimal" value={factor} onChange={(e) => setFactor(e.target.value)} />}
          </Field>
          <label className="flex min-h-11 items-center gap-2 text-sm">
            <input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} className="size-4 accent-primary" />
            {t("categories.compatActive")}
          </label>
        </>
      )}
    </MasterDialog>
  )
}
