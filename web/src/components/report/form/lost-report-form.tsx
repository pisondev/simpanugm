"use client"

import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation } from "@tanstack/react-query"
import { TriangleAlert } from "lucide-react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useCallback, useMemo, useState } from "react"
import { Controller, type FieldPath, useForm, useWatch } from "react-hook-form"
import { toast } from "sonner"
import { z } from "zod"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { OwnerFeatureLabel } from "@/components/dashboard/owner-feature-label"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api/client"
import { call, fieldErrors, isApiError, messageFor } from "@/lib/api/problem"
import type { Category, Colour, LostReport, LostReportCreate, LostReportUpdate } from "@/lib/api/types"
import { formatRange } from "@/lib/format"
import { isoToWib, lostWindow, todayInWib, windowProblem } from "@/lib/time"

import { Detail, DetailList } from "../detail-list"
import { AssistNote, CategoryAlternatives, RedraftButton, SuggestedTag } from "./assist-ui"
import { CategoryPicker, leaves } from "./category-picker"
import { ColourPicker } from "./colour-picker"
import { Field } from "./field"
import { LocationPicker } from "./location-picker"
import { LostWindowFields } from "./lost-window-fields"
import { busy, maxPhotos, type PhotoItem, PhotoPicker, photoFromId, readyIds } from "./photo-picker"
import { Stepper } from "./stepper"
import { type AssistFields, useAssist } from "./use-assist"
import { useReportDraft } from "./use-report-draft"

type Mode = "create" | "edit" | "proxy"

const presetValues = ["pagi", "siang", "sore", "malam", "sehari", "custom"] as const

function buildSchema(t: ReturnType<typeof useTranslations<"form">>, te: ReturnType<typeof useTranslations<"errors">>, proxy: boolean) {
  const text = (min: number, max: number) =>
    z
      .string()
      .trim()
      .min(min, { error: (iss) => (String(iss.input ?? "").length === 0 ? te("required") : te("tooShort", { min, count: String(iss.input ?? "").length })) })
      .max(max, { error: te("tooLong", { max }) })
  return z
    .object({
      category_id: z.string().min(1, { error: te("required") }),
      item_name: text(3, 80),
      description: text(20, 1000),
      colours: z.array(z.string()).max(2),
      brand: z
        .string()
        .trim()
        .max(40, { error: te("tooLong", { max: 40 }) }),
      location_unknown: z.boolean(),
      places: z.array(z.object({ location_id: z.string(), name: z.string(), zone_name: z.string().optional() })).max(3),
      location_note: z
        .string()
        .trim()
        .max(200, { error: te("tooLong", { max: 200 }) }),
      date: z.string().min(1, { error: te("required") }),
      multi_day: z.boolean(),
      end_date: z.string(),
      preset: z.enum(presetValues),
      from: z.string(),
      until: z.string(),
      private_details: z
        .string()
        .trim()
        .max(500, { error: te("tooLong", { max: 500 }) }),
      proxy_name: z
        .string()
        .trim()
        .max(100, { error: te("tooLong", { max: 100 }) }),
      proxy_phone: z
        .string()
        .trim()
        .max(20, { error: te("tooLong", { max: 20 }) }),
      proxy_consent: z.boolean(),
    })
    .superRefine((v, ctx) => {
      if (!v.location_unknown && v.places.length === 0) {
        ctx.addIssue({ code: "custom", path: ["places"], message: t("places.required") })
      }
      if (v.date) {
        const problem = windowProblem(
          lostWindow({ date: v.date, endDate: v.multi_day ? v.end_date || v.date : undefined, preset: v.preset, from: v.from, until: v.until }),
        )
        if (problem) {
          ctx.addIssue({ code: "custom", path: ["date"], message: t(`lostWindow.${problem}`) })
        }
      }
      if (proxy) {
        if (v.proxy_name.length < 2) {
          ctx.addIssue({ code: "custom", path: ["proxy_name"], message: te("required") })
        }
        if (!v.proxy_consent) {
          ctx.addIssue({ code: "custom", path: ["proxy_consent"], message: t("proxy.consentRequired") })
        }
      }
    })
}

type Values = z.input<ReturnType<typeof buildSchema>>

function emptyValues(): Values {
  return {
    category_id: "",
    item_name: "",
    description: "",
    colours: [],
    brand: "",
    location_unknown: false,
    places: [],
    location_note: "",
    date: todayInWib(),
    multi_day: false,
    end_date: "",
    preset: "sehari",
    from: "00:00",
    until: "23:59",
    private_details: "",
    proxy_name: "",
    proxy_phone: "",
    proxy_consent: false,
  }
}

function fromReport(r: LostReport): Values {
  const from = isoToWib(r.lost_from)
  const until = isoToWib(r.lost_until)
  return {
    ...emptyValues(),
    category_id: r.category.id,
    item_name: r.item_name,
    description: r.description,
    colours: r.colours,
    brand: r.brand ?? "",
    location_unknown: r.location_unknown,
    places: r.places.flatMap((p) => (p.location ? [{ location_id: p.location.id, name: p.location.name }] : [])),
    location_note: r.location_note ?? "",
    date: from.date,
    multi_day: from.date !== until.date,
    end_date: until.date,
    preset: "custom",
    from: from.time,
    until: until.time,
    private_details: r.private_details ?? "",
  }
}

type StepId = "proxy" | "photos" | "detail" | "place" | "private"

const stepFields: Record<StepId, FieldPath<Values>[]> = {
  proxy: ["proxy_name", "proxy_phone", "proxy_consent"],
  photos: [],
  detail: ["category_id", "item_name", "description", "colours", "brand"],
  place: ["location_unknown", "places", "location_note", "date", "multi_day", "end_date", "preset", "from", "until"],
  private: ["private_details"],
}

// api field names to form fields, nested names are matched by their first segment
const apiFields: Record<string, FieldPath<Values> | "photos"> = {
  category_id: "category_id",
  item_name: "item_name",
  description: "description",
  colours: "colours",
  brand: "brand",
  places: "places",
  location_unknown: "places",
  location_note: "location_note",
  lost_from: "date",
  lost_until: "date",
  private_details: "private_details",
  media_ids: "photos",
  "proxy.name": "proxy_name",
  "proxy.phone": "proxy_phone",
}

type Props = { mode: Mode; categories: Category[]; colours: Colour[]; initial?: LostReport }

export function LostReportForm({ mode, categories, colours, initial }: Props) {
  const t = useTranslations("form")
  const te = useTranslations("errors")
  const tc = useTranslations("common")
  const tr = useTranslations("report")
  const router = useRouter()
  const proxy = mode === "proxy"
  const schema = useMemo(() => buildSchema(t, te, proxy), [t, te, proxy])
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: initial ? fromReport(initial) : emptyValues(),
    mode: "onTouched",
  })
  const { control, register, handleSubmit, trigger, formState, setError } = form
  const errors = formState.errors
  const [photos, setPhotos] = useState<PhotoItem[]>(() =>
    initial ? [...initial.photos].sort((a, b) => a.position - b.position).map((p) => photoFromId(p.media_id)) : [],
  )
  const [photoError, setPhotoError] = useState<string>()
  const [step, setStep] = useState(0)
  const [submitError, setSubmitError] = useState<string>()
  const [stale, setStale] = useState(false)
  const steps: StepId[] = proxy ? ["proxy", "photos", "detail", "place", "private"] : ["photos", "detail", "place", "private"]
  const titles = steps.map((s) => (s === "proxy" ? t("proxy.title") : t(`lost.steps.${s}`)))
  const onReview = step >= steps.length
  const current = steps[step]
  const locationUnknown = useWatch({ control, name: "location_unknown" })
  const read = useCallback(() => {
    const v = form.getValues()
    return { category_id: v.category_id, colours: v.colours, brand: v.brand, description: v.description }
  }, [form])
  const write = useCallback(
    (patch: Partial<AssistFields>) => (Object.keys(patch) as (keyof AssistFields)[]).forEach((k) => form.setValue(k, patch[k] as never, { shouldDirty: true })),
    [form],
  )
  const assist = useAssist({ enabled: mode !== "edit", reportType: "lost", mediaId: readyIds(photos)[0], read, write, drafts: true })

  const draft = useReportDraft({
    enabled: mode !== "edit",
    draftKey: `simpan:draft:lost:${mode}`,
    form,
    photos,
    setPhotos,
    empty: emptyValues,
    onDiscard: () => setStep(0),
  })

  async function next() {
    if (current === "photos") {
      if (busy(photos)) {
        setPhotoError(t("photos.waitUpload"))
        return
      }
      setPhotoError(undefined)
    } else if (current && !(await trigger(stepFields[current]))) {
      return
    }
    setStep((s) => s + 1)
    window.scrollTo({ top: 0 })
  }

  function stepOf(field: FieldPath<Values> | "photos"): number {
    const index = steps.findIndex((s) => (field === "photos" ? s === "photos" : stepFields[s].includes(field)))
    return index < 0 ? 0 : index
  }

  const submit = useMutation({
    mutationFn: async (v: Values) => {
      const span = lostWindow({ date: v.date, endDate: v.multi_day ? v.end_date || v.date : undefined, preset: v.preset, from: v.from, until: v.until })
      const places = v.location_unknown ? [] : v.places.map((p) => ({ location_id: p.location_id }))
      if (mode === "edit" && initial) {
        const body: LostReportUpdate = {
          category_id: v.category_id,
          item_name: v.item_name,
          description: v.description,
          colours: v.colours,
          brand: v.brand,
          ...span,
          location_unknown: v.location_unknown,
          places,
          location_note: v.location_note,
          private_details: v.private_details,
          media_ids: readyIds(photos),
        }
        return call(api.PATCH("/lost-reports/{id}", { params: { path: { id: initial.id }, header: { "If-Match": String(initial.version) } }, body }))
      }
      const body: LostReportCreate = {
        category_id: v.category_id,
        item_name: v.item_name,
        description: v.description,
        colours: v.colours,
        brand: v.brand || undefined,
        ...span,
        location_unknown: v.location_unknown,
        places,
        location_note: v.location_note || undefined,
        private_details: v.private_details || undefined,
        media_ids: readyIds(photos),
        proxy: proxy ? { name: v.proxy_name, phone: v.proxy_phone || undefined } : undefined,
        assist_outcome: assist.outcome({ category_id: v.category_id, colours: v.colours, brand: v.brand, description: v.description }),
      }
      return call(api.POST("/lost-reports", { params: { header: { "Idempotency-Key": draft.key.current } }, body }))
    },
    onSuccess: (report) => {
      draft.clear()
      toast.success(mode === "edit" ? t("lost.updated") : t("lost.created"))
      router.push(proxy ? `/petugas/laporan/hilang/${report.code}` : `/laporan/hilang/${report.code}`)
      router.refresh()
    },
    onError: (err) => {
      if (isApiError(err, 412)) {
        setStale(true)
        return
      }
      const fields = fieldErrors(err)
      let first = Number.POSITIVE_INFINITY
      for (const [name, message] of Object.entries(fields)) {
        const target = apiFields[name] ?? apiFields[name.split(/[.[]/)[0] ?? ""]
        if (!target) {
          continue
        }
        if (target === "photos") {
          setPhotoError(message)
        } else {
          setError(target, { message })
        }
        first = Math.min(first, stepOf(target))
      }
      if (Number.isFinite(first)) {
        setStep(first)
        return
      }
      setSubmitError(messageFor(err, te))
    },
  })

  const values = useWatch({ control })
  const leafList = leaves(categories)
  const category = leafList.find((l) => l.id === values.category_id)
  const span = values.date
    ? lostWindow({
        date: values.date,
        endDate: values.multi_day ? values.end_date || values.date : undefined,
        preset: values.preset ?? "sehari",
        from: values.from,
        until: values.until,
      })
    : null

  return (
    <form
      noValidate
      onSubmit={(e) => {
        e.preventDefault()
        if (!onReview) {
          void next()
          return
        }
        setSubmitError(undefined)
        void handleSubmit(
          (v) => submit.mutate(v),
          () => setSubmitError(t("fixErrors")),
        )()
      }}
      className="surface flex flex-col gap-6 p-4 sm:p-6"
    >
      <Stepper steps={titles} current={step} onJump={setStep} />

      {stale ? (
        <Alert variant="destructive">
          <TriangleAlert aria-hidden />
          <AlertTitle>{te("stale")}</AlertTitle>
          <AlertDescription>
            <Button
              type="button"
              variant="outline"
              size="sm"
              className="mt-2"
              onClick={() => {
                setStale(false)
                router.refresh()
              }}
            >
              {te("reload")}
            </Button>
          </AlertDescription>
        </Alert>
      ) : null}

      {current === "proxy" ? (
        <div className="flex flex-col gap-5">
          <Field id="proxy-name" label={t("proxy.visitorName")} error={errors.proxy_name?.message}>
            {(aria) => <Input {...aria} autoComplete="off" {...register("proxy_name")} />}
          </Field>
          <Field id="proxy-phone" label={t("proxy.visitorPhone")} optionalLabel={tc("optional")} error={errors.proxy_phone?.message}>
            {(aria) => <Input {...aria} type="tel" inputMode="tel" autoComplete="off" {...register("proxy_phone")} />}
          </Field>
          <Controller
            control={control}
            name="proxy_consent"
            render={({ field }) => (
              <div className="flex flex-col gap-1">
                <label className="flex min-h-11 items-start gap-3">
                  <Checkbox
                    className="mt-1"
                    checked={field.value}
                    onCheckedChange={(v) => field.onChange(Boolean(v))}
                    aria-invalid={errors.proxy_consent ? true : undefined}
                  />
                  <span>{t("proxy.consent")}</span>
                </label>
                {errors.proxy_consent ? <p className="text-sm text-destructive">{errors.proxy_consent.message}</p> : null}
              </div>
            )}
          />
        </div>
      ) : null}

      {current === "photos" ? (
        <section id="smart-report-assistant" className="scroll-mt-24">
          <OwnerFeatureLabel feature="assistant" />
          <Field id="photos" label={t("photos.label")} hint={t("photos.hint")} optionalLabel={tc("optional")} error={photoError}>
            {(aria) => (
              <PhotoPicker purpose="lost_report" items={photos} setItems={setPhotos} describedBy={aria["aria-describedby"]} invalid={aria["aria-invalid"]} />
            )}
          </Field>
        </section>
      ) : null}

      {current === "detail" ? (
        <div className="flex flex-col gap-5">
          {mode !== "edit" ? <OwnerFeatureLabel feature="assistant" /> : null}
          <AssistNote assist={assist} />
          <Controller
            control={control}
            name="category_id"
            render={({ field }) => (
              <Field
                id="category"
                label={t("category.label")}
                hint={assist.suggested("category", field.value) ? <SuggestedTag show /> : undefined}
                error={errors.category_id?.message}
              >
                {(aria) => (
                  <>
                    <CategoryPicker
                      tree={categories}
                      value={field.value}
                      onChange={field.onChange}
                      id={aria.id}
                      describedBy={aria["aria-describedby"]}
                      invalid={aria["aria-invalid"]}
                    />
                    <CategoryAlternatives assist={assist} onPick={assist.pick} />
                  </>
                )}
              </Field>
            )}
          />
          <Field id="item-name" label={t("itemName.label")} error={errors.item_name?.message}>
            {(aria) => <Input {...aria} placeholder={t("itemName.placeholder")} maxLength={80} {...register("item_name")} />}
          </Field>
          <Field
            id="description"
            label={t("description.label")}
            hint={tc("chars", { count: values.description?.length ?? 0, max: 1000 })}
            error={errors.description?.message}
          >
            {(aria) => (
              <>
                <Textarea {...aria} rows={5} placeholder={t("description.lostPlaceholder")} maxLength={1000} {...register("description")} />
                {mode !== "edit" ? <RedraftButton assist={assist} /> : null}
              </>
            )}
          </Field>
          <Controller
            control={control}
            name="colours"
            render={({ field }) => (
              <Field
                id="colours"
                as="fieldset"
                label={t("colours.label")}
                hint={
                  <>
                    {t("colours.hint")} <SuggestedTag show={assist.suggested("colours", field.value)} />
                  </>
                }
                optionalLabel={tc("optional")}
              >
                {(aria) => <ColourPicker palette={colours} value={field.value} onChange={field.onChange} describedBy={aria["aria-describedby"]} />}
              </Field>
            )}
          />
          <Field
            id="brand"
            label={t("brand.label")}
            hint={assist.suggested("brand", values.brand) ? <SuggestedTag show /> : undefined}
            optionalLabel={tc("optional")}
            error={errors.brand?.message}
          >
            {(aria) => <Input {...aria} placeholder={t("brand.placeholder")} maxLength={40} {...register("brand")} />}
          </Field>
        </div>
      ) : null}

      {current === "place" ? (
        <div className="flex flex-col gap-5">
          <Controller
            control={control}
            name="places"
            render={({ field }) => (
              <Field id="places" label={t("places.label")} hint={t("places.hint")} error={errors.places?.message}>
                {(aria) => (
                  <LocationPicker
                    id={aria.id}
                    value={field.value}
                    onChange={field.onChange}
                    max={3}
                    describedBy={aria["aria-describedby"]}
                    invalid={aria["aria-invalid"]}
                    disabled={locationUnknown}
                  />
                )}
              </Field>
            )}
          />
          <Controller
            control={control}
            name="location_unknown"
            render={({ field }) => (
              <label className="-mt-2 flex min-h-11 items-center gap-3">
                <Checkbox checked={field.value} onCheckedChange={(v) => field.onChange(Boolean(v))} />
                {t("places.unknown")}
              </label>
            )}
          />
          <Field id="location-note" label={t("locationNote.label")} optionalLabel={tc("optional")} error={errors.location_note?.message}>
            {(aria) => <Input {...aria} placeholder={t("locationNote.placeholder")} maxLength={200} {...register("location_note")} />}
          </Field>
          <LostWindowFields control={control} register={register} errors={errors} max={todayInWib()} />
        </div>
      ) : null}

      {current === "private" ? (
        <Field
          id="private-details"
          label={t("privateDetails.label")}
          hint={t("privateDetails.hint")}
          optionalLabel={tc("optional")}
          error={errors.private_details?.message}
        >
          {(aria) => <Textarea {...aria} rows={4} placeholder={t("privateDetails.placeholder")} maxLength={500} {...register("private_details")} />}
        </Field>
      ) : null}

      {onReview ? (
        <section aria-labelledby="review-title" className="flex flex-col gap-4">
          <div>
            <h2 id="review-title" className="text-lg font-semibold">
              {t("review")}
            </h2>
            <p className="text-muted-foreground">{t("reviewLead")}</p>
          </div>
          <div className="surface p-4">
            <DetailList>
              {proxy ? (
                <Detail label={tr("reporter")}>
                  {values.proxy_name}
                  {values.proxy_phone ? ` | ${values.proxy_phone}` : ""}
                </Detail>
              ) : null}
              <Detail label={tr("photos")}>{readyIds(photos).length > 0 ? `${readyIds(photos).length} / ${maxPhotos}` : tr("noPhotos")}</Detail>
              <Detail label={tr("category")}>{category ? `${category.name} (${category.parentName})` : tc("emptyValue")}</Detail>
              <Detail label={tr("itemName")}>{values.item_name || tc("emptyValue")}</Detail>
              <Detail label={tr("description")}>{values.description || tc("emptyValue")}</Detail>
              <Detail label={tr("colours")}>{values.colours?.length ? values.colours.join(", ") : tc("emptyValue")}</Detail>
              <Detail label={tr("brand")}>{values.brand || tc("emptyValue")}</Detail>
              <Detail label={tr("places")}>
                {values.location_unknown ? tr("placeUnknown") : values.places?.map((p) => p.name).join(", ") || tc("emptyValue")}
              </Detail>
              {values.location_note ? <Detail label={tr("locationNote")}>{values.location_note}</Detail> : null}
              <Detail label={tr("lostWindow")}>{span ? formatRange(span.lost_from, span.lost_until) : tc("emptyValue")}</Detail>
              <Detail label={tr("privateDetails")}>{values.private_details || tc("emptyValue")}</Detail>
            </DetailList>
          </div>
          <div className="flex flex-wrap gap-2">
            {titles.map((title, i) => (
              <Button key={title} type="button" variant="link" size="sm" onClick={() => setStep(i)}>
                {t("editStep")} {title.toLowerCase()}
              </Button>
            ))}
          </div>
        </section>
      ) : null}

      {submitError ? (
        <Alert variant="destructive">
          <TriangleAlert aria-hidden />
          <AlertDescription>{submitError}</AlertDescription>
        </Alert>
      ) : null}

      <div
        className={
          onReview
            ? "flex flex-wrap gap-3 pt-2"
            : "sticky bottom-16 z-10 -mx-4 flex flex-col gap-2 border-t bg-background/95 px-4 py-3 backdrop-blur sm:flex-row sm:gap-3 md:static md:mx-0 md:border-0 md:bg-transparent md:px-0 md:py-0 lg:bottom-0"
        }
      >
        {step > 0 ? (
          <Button type="button" variant="outline" onClick={() => setStep((s) => s - 1)}>
            {tc("back")}
          </Button>
        ) : null}
        {current === "photos" && photos.length === 0 && mode !== "edit" ? (
          <Button type="button" variant="ghost" className="w-full sm:w-auto" onClick={() => void next()}>
            {t("photos.skip")}
          </Button>
        ) : null}
        <Button type="submit" className="ml-auto w-full min-w-32 sm:w-auto" disabled={submit.isPending || submit.isSuccess}>
          {onReview ? (submit.isPending ? tc("submitting") : mode === "edit" ? t("lost.saveEdit") : t("lost.submit")) : tc("next")}
        </Button>
      </div>
    </form>
  )
}
