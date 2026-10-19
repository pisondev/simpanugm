"use client"

import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation } from "@tanstack/react-query"
import { Info, TriangleAlert } from "lucide-react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useCallback, useEffect, useMemo, useState } from "react"
import { Controller, type FieldPath, useForm, useWatch } from "react-hook-form"
import { toast } from "sonner"
import { z } from "zod"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import { Textarea } from "@/components/ui/textarea"
import { api, mediaUrl } from "@/lib/api/client"
import { usePosts } from "@/lib/api/hooks"
import { call, fieldErrors, isApiError, messageFor } from "@/lib/api/problem"
import type { Category, Colour, DuplicateCandidate, FoundReport, FoundReportCreate, FoundReportUpdate, Post } from "@/lib/api/types"
import { formatDateTime } from "@/lib/format"
import { foundTimeProblem, isoToWib, nowTimeInWib, todayInWib, wibToIso } from "@/lib/time"
import { cn } from "@/lib/utils"

import { Detail, DetailList } from "../detail-list"
import { AssistNote, CategoryAlternatives, RedraftButton, SuggestedTag } from "./assist-ui"
import { CategoryPicker, leaves } from "./category-picker"
import { ColourPicker } from "./colour-picker"
import { Field } from "./field"
import { LocationPicker } from "./location-picker"
import { busy, type PhotoItem, PhotoPicker, photoFromId, readyIds } from "./photo-picker"
import { PostPicker } from "./post-picker"
import { Stepper } from "./stepper"
import { type AssistFields, useAssist } from "./use-assist"
import { useReportDraft } from "./use-report-draft"

type Mode = "create" | "edit" | "proxy"

function buildSchema(
  t: ReturnType<typeof useTranslations<"form">>,
  te: ReturnType<typeof useTranslations<"errors">>,
  mode: Mode,
  sensitivityOf: (id: string) => string | undefined,
  custodyEditable: boolean,
) {
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
      description: text(10, 500),
      colours: z.array(z.string()).max(2),
      brand: z
        .string()
        .trim()
        .max(40, { error: te("tooLong", { max: 40 }) }),
      location: z.array(z.object({ location_id: z.string(), name: z.string(), zone_name: z.string().optional() })).max(1),
      location_note: z
        .string()
        .trim()
        .max(200, { error: te("tooLong", { max: 200 }) }),
      found_date: z.string().min(1, { error: te("required") }),
      found_time: z.string().min(1, { error: te("required") }),
      custody_mode: z.enum(["handover", "keep"]),
      intended_post_id: z.string(),
      hidden_details: z
        .string()
        .trim()
        .max(500, { error: te("tooLong", { max: 500 }) }),
      proxy_finder_name: z
        .string()
        .trim()
        .max(100, { error: te("tooLong", { max: 100 }) }),
      proxy_post_id: z.string(),
      proxy_slot: z
        .string()
        .trim()
        .max(40, { error: te("tooLong", { max: 40 }) }),
      proxy_condition: z
        .string()
        .trim()
        .max(500, { error: te("tooLong", { max: 500 }) }),
    })
    .superRefine((v, ctx) => {
      if (v.location.length === 0) {
        ctx.addIssue({ code: "custom", path: ["location"], message: t("foundPlace.required") })
      }
      if (v.found_date && v.found_time) {
        const problem = foundTimeProblem(wibToIso(v.found_date, v.found_time))
        if (problem) {
          ctx.addIssue({ code: "custom", path: ["found_date"], message: t(`foundTime.${problem}`) })
        }
      }
      if (mode === "proxy") {
        if (!v.proxy_post_id) {
          ctx.addIssue({ code: "custom", path: ["proxy_post_id"], message: t("custody.postRequired") })
        }
        if (v.proxy_slot.length === 0) {
          ctx.addIssue({ code: "custom", path: ["proxy_slot"], message: te("required") })
        }
        return
      }
      if (!custodyEditable) {
        return
      }
      if (v.custody_mode === "keep" && v.category_id && sensitivityOf(v.category_id) !== "normal") {
        ctx.addIssue({ code: "custom", path: ["custody_mode"], message: t("custody.mustHandOver") })
      }
      if (v.custody_mode === "handover" && !v.intended_post_id) {
        ctx.addIssue({ code: "custom", path: ["intended_post_id"], message: t("custody.postRequired") })
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
    location: [],
    location_note: "",
    found_date: todayInWib(),
    found_time: nowTimeInWib(),
    custody_mode: "handover",
    intended_post_id: "",
    hidden_details: "",
    proxy_finder_name: "",
    proxy_post_id: "",
    proxy_slot: "",
    proxy_condition: "",
  }
}

function fromReport(r: FoundReport): Values {
  const at = isoToWib(r.found_at)
  return {
    ...emptyValues(),
    category_id: r.category.id,
    item_name: r.item_name,
    description: r.description,
    colours: r.colours,
    brand: r.brand ?? "",
    location: [{ location_id: r.found_location.id, name: r.found_location.name, zone_name: r.found_location.zone_name }],
    location_note: r.location_note ?? "",
    found_date: at.date,
    found_time: at.time,
    custody_mode: r.custody_keep ? "keep" : "handover",
    intended_post_id: r.intended_post?.id ?? "",
    hidden_details: r.hidden_details ?? "",
  }
}

type StepId = "photos" | "detail" | "place" | "custody"

const stepFields: Record<StepId, FieldPath<Values>[]> = {
  photos: [],
  detail: ["category_id", "item_name", "description", "colours", "brand"],
  place: ["location", "location_note", "found_date", "found_time"],
  custody: ["custody_mode", "intended_post_id", "hidden_details", "proxy_finder_name", "proxy_post_id", "proxy_slot", "proxy_condition"],
}

const apiFields: Record<string, FieldPath<Values> | "photos"> = {
  category_id: "category_id",
  item_name: "item_name",
  description: "description",
  colours: "colours",
  brand: "brand",
  found_location_id: "location",
  location_note: "location_note",
  found_at: "found_date",
  hidden_details: "hidden_details",
  media_ids: "photos",
  custody: "custody_mode",
  "custody.mode": "custody_mode",
  "custody.intended_post_id": "intended_post_id",
  "proxy.finder_name": "proxy_finder_name",
  "proxy.post_id": "proxy_post_id",
  "proxy.storage_slot": "proxy_slot",
  "proxy.condition_note": "proxy_condition",
}

type Props = { mode: Mode; categories: Category[]; colours: Colour[]; initial?: FoundReport; staffPosts?: Post[] }

export function FoundReportForm({ mode, categories, colours, initial, staffPosts = [] }: Props) {
  const t = useTranslations("form")
  const te = useTranslations("errors")
  const tc = useTranslations("common")
  const tr = useTranslations("report")
  const router = useRouter()
  const proxy = mode === "proxy"
  const custodyEditable = !initial || initial.custody_status === "with_finder"
  const leafList = useMemo(() => leaves(categories), [categories])
  const schema = useMemo(
    () => buildSchema(t, te, mode, (id) => leafList.find((l) => l.id === id)?.sensitivity, custodyEditable),
    [t, te, mode, leafList, custodyEditable],
  )
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: initial ? fromReport(initial) : { ...emptyValues(), proxy_post_id: staffPosts.length === 1 ? (staffPosts[0]?.id ?? "") : "" },
    mode: "onTouched",
  })
  const { control, register, handleSubmit, trigger, formState, setError, setValue } = form
  const errors = formState.errors
  const [photos, setPhotos] = useState<PhotoItem[]>(() =>
    initial ? [...initial.photos].sort((a, b) => a.position - b.position).map((p) => photoFromId(p.media_id)) : [],
  )
  const [photoError, setPhotoError] = useState<string>()
  const [step, setStep] = useState(0)
  const [submitError, setSubmitError] = useState<string>()
  const [stale, setStale] = useState(false)
  const steps: StepId[] = ["photos", "detail", "place", "custody"]
  const titles = steps.map((s) => (s === "custody" && proxy ? t("found.stepsProxy.custody") : t(`found.steps.${s}`)))
  const onReview = step >= steps.length
  const current = steps[step]

  const draft = useReportDraft({
    enabled: mode !== "edit",
    draftKey: `simpan:draft:found:${mode}`,
    form,
    photos,
    setPhotos,
    empty: emptyValues,
    onDiscard: () => setStep(0),
  })

  const values = useWatch({ control })
  const read = useCallback(() => {
    const v = form.getValues()
    return { category_id: v.category_id, colours: v.colours, brand: v.brand, description: v.description }
  }, [form])
  const write = useCallback(
    (patch: Partial<AssistFields>) => (Object.keys(patch) as (keyof AssistFields)[]).forEach((k) => form.setValue(k, patch[k] as never, { shouldDirty: true })),
    [form],
  )
  const assist = useAssist({ enabled: mode !== "edit", reportType: "found", mediaId: readyIds(photos)[0], read, write, drafts: true })
  const category = leafList.find((l) => l.id === values.category_id)
  const mustHandOver = category ? category.sensitivity !== "normal" : false
  const locationId = values.location?.[0]?.location_id
  const posts = usePosts(proxy ? undefined : locationId)

  useEffect(() => {
    if (mustHandOver && values.custody_mode === "keep") {
      setValue("custody_mode", "handover")
    }
  }, [mustHandOver, setValue, values.custody_mode])

  // US-13 criterion 3: the nearest active post is recommended by preselecting it
  const nearest = posts.data?.items[0]?.id
  useEffect(() => {
    if (!proxy && locationId && nearest && !form.getValues("intended_post_id")) {
      setValue("intended_post_id", nearest)
    }
  }, [form, locationId, nearest, proxy, setValue])

  const [duplicates, setDuplicates] = useState<DuplicateCandidate[]>([])
  const [duplicatesDismissed, setDuplicatesDismissed] = useState(false)

  useEffect(() => {
    if (!onReview || mode === "edit") return
    let active = true
    const checkDuplicates = async () => {
      const catId = values.category_id
      const locId = values.location?.[0]?.location_id
      const fDate = values.found_date
      const fTime = values.found_time
      const itemName = values.item_name
      if (!catId || !locId || !fDate || !fTime || !itemName) return
      try {
        const res = await api.POST("/assist/duplicates", {
          body: {
            category_id: catId,
            found_location_id: locId,
            found_at: wibToIso(fDate, fTime),
            item_name: itemName,
            description: values.description,
            media_ids: readyIds(photos),
          },
        })
        if (active && res.data?.items) {
          setDuplicates(res.data.items)
        }
      } catch {
        // ignore duplicate check failure
      }
    }
    void checkDuplicates()
    return () => {
      active = false
    }
  }, [onReview, mode, values.category_id, values.location, values.found_date, values.found_time, values.item_name, values.description, photos])

  async function next() {
    if (current === "photos") {
      if (busy(photos)) {
        setPhotoError(t("photos.waitUpload"))
        return
      }
      if (readyIds(photos).length === 0) {
        setPhotoError(t("photos.requiredFound"))
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
      const foundAt = wibToIso(v.found_date, v.found_time)
      const locationIdValue = v.location[0]?.location_id ?? ""
      const custody = v.custody_mode === "keep" ? { mode: "keep" as const } : { mode: "handover" as const, intended_post_id: v.intended_post_id }
      if (mode === "edit" && initial) {
        const body: FoundReportUpdate = {
          category_id: v.category_id,
          item_name: v.item_name,
          description: v.description,
          hidden_details: v.hidden_details,
          colours: v.colours,
          brand: v.brand,
          found_at: foundAt,
          found_location_id: locationIdValue,
          location_note: v.location_note,
          media_ids: readyIds(photos),
          custody: custodyEditable ? custody : undefined,
        }
        return call(api.PATCH("/found-reports/{id}", { params: { path: { id: initial.id }, header: { "If-Match": String(initial.version) } }, body }))
      }
      const body: FoundReportCreate = {
        category_id: v.category_id,
        item_name: v.item_name,
        description: v.description,
        hidden_details: v.hidden_details || undefined,
        colours: v.colours,
        brand: v.brand || undefined,
        found_at: foundAt,
        found_location_id: locationIdValue,
        location_note: v.location_note || undefined,
        media_ids: readyIds(photos),
        custody: proxy ? { mode: "handover", intended_post_id: v.proxy_post_id } : custody,
        proxy: proxy
          ? {
              finder_name: v.proxy_finder_name || undefined,
              post_id: v.proxy_post_id,
              storage_slot: v.proxy_slot,
              condition_note: v.proxy_condition || undefined,
            }
          : undefined,
        assist_outcome: assist.outcome({ category_id: v.category_id, colours: v.colours, brand: v.brand, description: v.description }),
      }
      return call(api.POST("/found-reports", { params: { header: { "Idempotency-Key": draft.key.current } }, body }))
    },
    onSuccess: (report) => {
      draft.clear()
      toast.success(mode === "edit" ? t("found.updated") : t("found.created"))
      router.push(proxy ? `/petugas/laporan/temuan/${report.code}` : `/laporan/temuan/${report.code}`)
      router.refresh()
    },
    onError: (err) => {
      if (isApiError(err, 412)) {
        setStale(true)
        return
      }
      let first = Number.POSITIVE_INFINITY
      for (const [name, message] of Object.entries(fieldErrors(err))) {
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

  const postList = proxy ? staffPosts : (posts.data?.items ?? [])
  const chosenPost = postList.find((p) => p.id === (proxy ? values.proxy_post_id : values.intended_post_id))

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

      {current === "photos" ? (
        <div className="flex flex-col gap-5">
          <div className="ai-panel flex flex-col gap-1 p-4">
            <p className="text-sm font-semibold text-primary">{t("assist.photoTitle")}</p>
            <p className="text-sm leading-relaxed text-muted-foreground">{t("assist.photoHint")}</p>
          </div>
          <Field
            id="photos"
            label={t("photos.label")}
            hint={t("photos.hint")}
            error={photoError}
            className="rounded-xl border border-dashed bg-surface-inset/40 p-5"
          >
            {(aria) => (
              <PhotoPicker purpose="found_report" items={photos} setItems={setPhotos} describedBy={aria["aria-describedby"]} invalid={aria["aria-invalid"]} />
            )}
          </Field>
          <AssistNote assist={assist} />
        </div>
      ) : null}

      {current === "detail" ? (
        <div className="flex flex-col gap-5">
          <AssistNote assist={assist} />
          <Controller
            control={control}
            name="category_id"
            render={({ field }) => (
              <Field
                id="category"
                suggested={assist.suggested("category", field.value)}
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
            suggested={assist.suggested("description", values.description)}
            motionIndex={3}
            label={t("description.label")}
            hint={
              <>
                {tc("chars", { count: values.description?.length ?? 0, max: 500 })}{" "}
                {assist.suggested("description", values.description) ? <SuggestedTag show /> : null}
              </>
            }
            error={errors.description?.message}
          >
            {(aria) => (
              <>
                <Textarea {...aria} rows={4} placeholder={t("description.foundPlaceholder")} maxLength={500} {...register("description")} />
                <RedraftButton assist={assist} />
              </>
            )}
          </Field>
          <Controller
            control={control}
            name="colours"
            render={({ field }) => (
              <Field
                id="colours"
                suggested={assist.suggested("colours", field.value)}
                motionIndex={1}
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
            suggested={assist.suggested("brand", values.brand)}
            motionIndex={2}
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
            name="location"
            render={({ field }) => (
              <Field id="location" label={t("foundPlace.label")} error={errors.location?.message}>
                {(aria) => (
                  <LocationPicker
                    id={aria.id}
                    value={field.value}
                    onChange={field.onChange}
                    max={1}
                    describedBy={aria["aria-describedby"]}
                    invalid={aria["aria-invalid"]}
                  />
                )}
              </Field>
            )}
          />
          <Field id="location-note" label={t("locationNote.label")} optionalLabel={tc("optional")} error={errors.location_note?.message}>
            {(aria) => <Input {...aria} placeholder={t("locationNote.placeholder")} maxLength={200} {...register("location_note")} />}
          </Field>
          <Field id="found-time" as="fieldset" label={t("foundTime.label")} error={errors.found_date?.message ?? errors.found_time?.message}>
            {() => (
              <div className="grid grid-cols-2 gap-3">
                <div className="flex flex-col gap-2">
                  <label htmlFor="found-date" className="text-sm">
                    {t("foundTime.date")}
                  </label>
                  <Input id="found-date" type="date" max={todayInWib()} {...register("found_date")} aria-invalid={errors.found_date ? true : undefined} />
                </div>
                <div className="flex flex-col gap-2">
                  <label htmlFor="found-clock" className="text-sm">
                    {t("foundTime.time")}
                  </label>
                  <Input id="found-clock" type="time" {...register("found_time")} />
                </div>
              </div>
            )}
          </Field>
        </div>
      ) : null}

      {current === "custody" && proxy ? (
        <div className="flex flex-col gap-5">
          <Controller
            control={control}
            name="proxy_post_id"
            render={({ field }) => (
              <Field id="proxy-post" as="fieldset" label={t("proxy.post")} error={errors.proxy_post_id?.message}>
                {(aria) => (
                  <PostPicker
                    name={field.name}
                    posts={staffPosts}
                    value={field.value}
                    onChange={field.onChange}
                    describedBy={aria["aria-describedby"]}
                    showNearest={false}
                  />
                )}
              </Field>
            )}
          />
          <Field id="proxy-slot" label={t("proxy.slot")} error={errors.proxy_slot?.message}>
            {(aria) => <Input {...aria} placeholder={t("proxy.slotPlaceholder")} maxLength={40} {...register("proxy_slot")} />}
          </Field>
          <Field id="proxy-condition" label={t("proxy.condition")} optionalLabel={tc("optional")} error={errors.proxy_condition?.message}>
            {(aria) => <Textarea {...aria} rows={3} placeholder={t("proxy.conditionPlaceholder")} maxLength={500} {...register("proxy_condition")} />}
          </Field>
          <Field id="proxy-finder" label={t("proxy.finderName")} optionalLabel={tc("optional")} error={errors.proxy_finder_name?.message}>
            {(aria) => <Input {...aria} autoComplete="off" maxLength={100} {...register("proxy_finder_name")} />}
          </Field>
          <Field
            id="hidden-details"
            label={t("hiddenDetails.label")}
            hint={t("hiddenDetails.hint")}
            optionalLabel={tc("optional")}
            error={errors.hidden_details?.message}
          >
            {(aria) => <Textarea {...aria} rows={3} placeholder={t("hiddenDetails.placeholder")} maxLength={500} {...register("hidden_details")} />}
          </Field>
        </div>
      ) : null}

      {current === "custody" && !proxy ? (
        <div className="flex flex-col gap-5">
          {custodyEditable ? (
            <>
              <Controller
                control={control}
                name="custody_mode"
                render={({ field }) => (
                  <Field id="custody-mode" as="fieldset" label={t("custody.label")} error={errors.custody_mode?.message}>
                    {() => (
                      <div className="grid gap-2 sm:grid-cols-2">
                        {(["handover", "keep"] as const).map((m) => {
                          const disabled = m === "keep" && mustHandOver
                          return (
                            <label
                              key={m}
                              className={cn(
                                "flex cursor-pointer items-start gap-3 rounded-xl border bg-card p-3 has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-ring/50",
                                field.value === m && "border-primary bg-primary-soft/40",
                                disabled && "cursor-not-allowed opacity-50",
                              )}
                            >
                              <input
                                type="radio"
                                name={field.name}
                                value={m}
                                checked={field.value === m}
                                disabled={disabled}
                                onChange={() => field.onChange(m)}
                                className="mt-1 size-4 accent-primary"
                              />
                              <span>
                                <span className="block font-medium">{t(`custody.${m}`)}</span>
                                <span className="block text-sm text-muted-foreground">{t(m === "handover" ? "custody.handoverHint" : "custody.keepHint")}</span>
                              </span>
                            </label>
                          )
                        })}
                      </div>
                    )}
                  </Field>
                )}
              />
              {mustHandOver ? (
                <Alert>
                  <Info aria-hidden />
                  <AlertDescription className="text-foreground">{t("custody.mustHandOver")}</AlertDescription>
                </Alert>
              ) : null}
              {values.custody_mode === "handover" ? (
                <Controller
                  control={control}
                  name="intended_post_id"
                  render={({ field }) => (
                    <Field
                      id="intended-post"
                      as="fieldset"
                      label={t("custody.post")}
                      hint={locationId ? undefined : t("custody.pickPlaceFirst")}
                      error={errors.intended_post_id?.message}
                    >
                      {(aria) =>
                        posts.isPending ? (
                          <div className="grid gap-2">
                            <Skeleton className="h-24 w-full rounded-xl" />
                            <Skeleton className="h-24 w-full rounded-xl" />
                          </div>
                        ) : (
                          <PostPicker name={field.name} posts={postList} value={field.value} onChange={field.onChange} describedBy={aria["aria-describedby"]} />
                        )
                      }
                    </Field>
                  )}
                />
              ) : null}
            </>
          ) : (
            <Alert>
              <Info aria-hidden />
              <AlertDescription className="text-foreground">{tr("inCustodyBody", { post: initial?.holding_post?.name ?? "" })}</AlertDescription>
            </Alert>
          )}
          <Field
            id="hidden-details"
            label={t("hiddenDetails.label")}
            hint={t("hiddenDetails.hint")}
            optionalLabel={tc("optional")}
            error={errors.hidden_details?.message}
          >
            {(aria) => <Textarea {...aria} rows={3} placeholder={t("hiddenDetails.placeholder")} maxLength={500} {...register("hidden_details")} />}
          </Field>
        </div>
      ) : null}

      {onReview ? (
        <section aria-labelledby="review-title" className="flex flex-col gap-4">
          <div>
            <h2 id="review-title" className="text-lg font-semibold">
              {t("review")}
            </h2>
            <p className="text-muted-foreground">{t("reviewLead")}</p>
          </div>

          {!duplicatesDismissed && duplicates.length > 0 ? (
            <div className="rounded-xl border border-warning/30 bg-warning/10 p-4">
              <div className="flex items-start gap-3">
                <TriangleAlert className="mt-0.5 size-5 shrink-0 text-warning" />
                <div className="flex flex-1 flex-col gap-3">
                  <div>
                    <h3 className="font-semibold text-warning-text">{t("found.duplicateWarningTitle")}</h3>
                    <p className="mt-1 text-sm text-warning-text/90">{t("found.duplicateWarningBody")}</p>
                  </div>

                  <div className="grid gap-2">
                    {duplicates.map((dup) => (
                      <div key={dup.id} className="flex items-center gap-3 rounded-lg border bg-card p-3 shadow-sm">
                        {dup.thumb_media_id ? (
                          <img src={mediaUrl(dup.thumb_media_id, "thumb")} alt={dup.item_name} className="size-14 rounded-md border object-cover" />
                        ) : (
                          <div className="flex size-14 items-center justify-center rounded-md border bg-muted text-xs text-muted-foreground">
                            {tr("photos")}
                          </div>
                        )}
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2">
                            <span className="truncate font-medium">{dup.item_name}</span>
                            <span className="shrink-0 rounded-full bg-warning/20 px-2 py-0.5 text-xs font-semibold text-warning-text">
                              {t("found.duplicateMatchPercent", { percent: Math.round(dup.duplicate_score * 100) })}
                            </span>
                          </div>
                          <p className="truncate text-xs text-muted-foreground">
                            {dup.category_name} &bull; {dup.location_name}
                          </p>
                          <p className="mt-0.5 text-xs text-muted-foreground">{formatDateTime(dup.found_at)}</p>
                        </div>
                        <Button type="button" variant="outline" size="sm" onClick={() => window.open(`/laporan/temuan/${dup.code}`, "_blank")}>
                          {t("found.duplicateSameItem")}
                        </Button>
                      </div>
                    ))}
                  </div>

                  <div className="flex justify-end pt-1">
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => setDuplicatesDismissed(true)}
                      className="text-warning-text hover:bg-warning/20"
                    >
                      {t("found.duplicateDismiss")}
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          ) : null}
          <div className="rounded-xl border bg-card p-4">
            <DetailList>
              <Detail label={tr("photos")}>{readyIds(photos).length}</Detail>
              <Detail label={tr("category")}>{category ? `${category.name} (${category.parentName})` : tc("emptyValue")}</Detail>
              <Detail label={tr("itemName")}>{values.item_name || tc("emptyValue")}</Detail>
              <Detail label={tr("description")}>{values.description || tc("emptyValue")}</Detail>
              <Detail label={tr("colours")}>{values.colours?.length ? values.colours.join(", ") : tc("emptyValue")}</Detail>
              <Detail label={tr("brand")}>{values.brand || tc("emptyValue")}</Detail>
              <Detail label={tr("foundLocation")}>{values.location?.[0]?.name ?? tc("emptyValue")}</Detail>
              {values.location_note ? <Detail label={tr("locationNote")}>{values.location_note}</Detail> : null}
              <Detail label={tr("foundAt")}>
                {values.found_date && values.found_time ? formatDateTime(wibToIso(values.found_date, values.found_time)) : tc("emptyValue")}
              </Detail>
              {proxy ? (
                <>
                  <Detail label={tr("holdingPost")}>{chosenPost?.name ?? tc("emptyValue")}</Detail>
                  <Detail label={tr("storageSlot")}>{values.proxy_slot || tc("emptyValue")}</Detail>
                  <Detail label={tr("finder")}>{values.proxy_finder_name || tc("emptyValue")}</Detail>
                </>
              ) : custodyEditable ? (
                <Detail label={tr("custody")}>
                  {values.custody_mode === "keep" ? t("custody.keep") : `${t("custody.handover")}: ${chosenPost?.name ?? tc("emptyValue")}`}
                </Detail>
              ) : null}
              <Detail label={tr("hiddenDetails")}>{values.hidden_details || tc("emptyValue")}</Detail>
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

      <div className="sticky bottom-16 z-10 -mx-4 flex gap-3 border-t bg-background/95 px-4 py-3 backdrop-blur md:static md:mx-0 md:border-0 md:bg-transparent md:px-0 md:py-0">
        {step > 0 ? (
          <Button type="button" variant="outline" onClick={() => setStep((s) => s - 1)}>
            {tc("back")}
          </Button>
        ) : null}
        <Button type="submit" className="ml-auto min-w-32" disabled={submit.isPending || submit.isSuccess}>
          {onReview ? (submit.isPending ? tc("submitting") : mode === "edit" ? t("found.saveEdit") : t("found.submit")) : tc("next")}
        </Button>
      </div>
    </form>
  )
}
