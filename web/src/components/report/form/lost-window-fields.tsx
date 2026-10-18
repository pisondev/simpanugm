"use client"

import { useTranslations } from "next-intl"
import { type Control, Controller, type FieldErrors, type UseFormRegister, useWatch } from "react-hook-form"

import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { presets } from "@/lib/time"
import { cn } from "@/lib/utils"

import { Field } from "./field"

export type WindowValues = {
  date: string
  multi_day: boolean
  end_date: string
  preset: keyof typeof presets | "custom"
  from: string
  until: string
}

const choices = [...(Object.keys(presets) as (keyof typeof presets)[]), "custom"] as const

type Props<T extends WindowValues> = {
  control: Control<T>
  register: UseFormRegister<T>
  errors: FieldErrors<T>
  max: string
}

// information-architecture 3.1: a date, then a part of day preset or two times, a window of at most seven days
export function LostWindowFields<T extends WindowValues>(props: Props<T>) {
  const { control, register, errors, max } = props as unknown as Props<WindowValues>
  const t = useTranslations("form.lostWindow")
  const preset = useWatch({ control, name: "preset" })
  const multi = useWatch({ control, name: "multi_day" })
  return (
    <Field id="lost-window" as="fieldset" label={t("label")} error={errors.date?.message ?? errors.until?.message}>
      {() => (
        <div className="flex flex-col gap-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="flex flex-col gap-2">
              <label htmlFor="lost-date" className="text-sm">
                {t("date")}
              </label>
              <Input id="lost-date" type="date" max={max} {...register("date")} aria-invalid={errors.date ? true : undefined} />
            </div>
            {multi ? (
              <div className="flex flex-col gap-2">
                <label htmlFor="lost-end-date" className="text-sm">
                  {t("endDate")}
                </label>
                <Input id="lost-end-date" type="date" max={max} {...register("end_date")} />
              </div>
            ) : null}
          </div>
          <Controller
            control={control}
            name="multi_day"
            render={({ field }) => (
              <label className="flex min-h-11 items-center gap-3 text-sm">
                <Checkbox checked={field.value} onCheckedChange={(v) => field.onChange(Boolean(v))} />
                {t("multiDay")}
              </label>
            )}
          />
          <div className="flex flex-col gap-2">
            <span className="text-sm">{t("preset")}</span>
            <Controller
              control={control}
              name="preset"
              render={({ field }) => (
                <div className="flex flex-wrap gap-2">
                  {choices.map((c) => (
                    <label
                      key={c}
                      className={cn(
                        "inline-flex min-h-11 cursor-pointer items-center rounded-full border bg-card px-4 text-sm has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-ring/50",
                        field.value === c && "border-primary bg-primary-soft text-primary",
                      )}
                    >
                      <input type="radio" className="sr-only" name={field.name} value={c} checked={field.value === c} onChange={() => field.onChange(c)} />
                      {t(`presets.${c}`)}
                      {c !== "custom" ? (
                        <span className="ml-1 text-xs text-muted-foreground">
                          {t("presetRange", { from: presets[c].from.replace(":", "."), until: presets[c].until.replace(":", ".") })}
                        </span>
                      ) : null}
                    </label>
                  ))}
                </div>
              )}
            />
          </div>
          {preset === "custom" ? (
            <div className="grid grid-cols-2 gap-3">
              <div className="flex flex-col gap-2">
                <label htmlFor="lost-from" className="text-sm">
                  {t("from")}
                </label>
                <Input id="lost-from" type="time" {...register("from")} />
              </div>
              <div className="flex flex-col gap-2">
                <label htmlFor="lost-until" className="text-sm">
                  {t("until")}
                </label>
                <Input id="lost-until" type="time" {...register("until")} aria-invalid={errors.until ? true : undefined} />
              </div>
            </div>
          ) : null}
        </div>
      )}
    </Field>
  )
}
