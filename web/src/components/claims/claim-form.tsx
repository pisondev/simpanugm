"use client"

import { useMutation } from "@tanstack/react-query"
import { Info, TriangleAlert } from "lucide-react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useMemo, useRef, useState } from "react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api/client"
import { call, fieldErrors, messageFor } from "@/lib/api/problem"
import type { Category, Claim, ClaimCreate, Colour } from "@/lib/api/types"
import { lostWindow, todayInWib } from "@/lib/time"

import { leaves } from "../report/form/category-picker"
import { ColourPicker } from "../report/form/colour-picker"
import { Field } from "../report/form/field"
import { busy, type PhotoItem, PhotoPicker, readyIds } from "../report/form/photo-picker"

export type Question = { key: string; question: string; required: boolean; check_hint?: string }
export type LostOption = { id: string; code: string; label: string }

type Props = {
  foundId: string
  foundName: string
  questions: Question[]
  lostOptions: LostOption[]
  presetLostId?: string
  matchId?: string
  categories?: Category[]
  colours?: Colour[]
  defaultCategoryId?: string
}

const maxEvidence = 3

// FR-CLAIM-01 to FR-CLAIM-04: the statement, category questions, up to three evidence photos,
// and inline lost report submission for public listing claims without prior lost reports.
export function ClaimForm({ foundId, foundName, questions, lostOptions, presetLostId, matchId, categories, colours, defaultCategoryId }: Props) {
  const t = useTranslations("claims.form")
  const te = useTranslations("errors")
  const router = useRouter()
  const key = useRef<string>(crypto.randomUUID())

  const hasExistingLost = lostOptions.length > 0
  const [isNewLost, setIsNewLost] = useState(!presetLostId && !hasExistingLost)
  const [lostId, setLostId] = useState(presetLostId ?? lostOptions[0]?.id ?? "")

  const leafCategories = useMemo(() => (categories ? leaves(categories) : []), [categories])

  const [newLostItemName, setNewLostItemName] = useState(foundName || "")
  const [newLostCategoryId, setNewLostCategoryId] = useState(defaultCategoryId ?? leafCategories[0]?.id ?? "")
  const [newLostColours, setNewLostColours] = useState<string[]>([])
  const [newLostDate, setNewLostDate] = useState(todayInWib())
  const [newLostDescription, setNewLostDescription] = useState("")
  const [newLostBrand, setNewLostBrand] = useState("")
  const [newLostPrivateDetails, setNewLostPrivateDetails] = useState("")

  const [statement, setStatement] = useState("")
  const [answers, setAnswers] = useState<Record<string, string>>({})
  const [visit, setVisit] = useState("")
  const [photos, setPhotos] = useState<PhotoItem[]>([])
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string>()

  const activeQuestions = useMemo(() => {
    if (isNewLost && newLostCategoryId && leafCategories.length > 0) {
      const selected = leafCategories.find((c) => c.id === newLostCategoryId)
      if (selected?.verification_questions && selected.verification_questions.length > 0) {
        return selected.verification_questions.map((q) => ({
          key: q.key,
          question: q.question,
          required: q.required,
          check_hint: q.check_hint ?? undefined,
        }))
      }
    }
    return questions
  }, [isNewLost, newLostCategoryId, leafCategories, questions])

  const submit = useMutation({
    mutationFn: (): Promise<Claim> => {
      let body: ClaimCreate
      if (isNewLost) {
        const span = lostWindow({ date: newLostDate, preset: "sehari" })
        body = {
          found_report_id: foundId,
          lost_report: {
            category_id: newLostCategoryId,
            item_name: newLostItemName.trim(),
            description: newLostDescription.trim(),
            colours: newLostColours,
            brand: newLostBrand.trim() || undefined,
            lost_from: span.lost_from,
            lost_until: span.lost_until,
            location_unknown: true,
            places: [],
            private_details: newLostPrivateDetails.trim() || undefined,
            media_ids: [],
          },
          match_id: matchId,
          statement: statement.trim(),
          answers: activeQuestions.map((q) => ({ key: q.key, answer: (answers[q.key] ?? "").trim() })).filter((a) => a.answer !== ""),
          evidence_media_ids: readyIds(photos),
          preferred_visit: visit.trim() || undefined,
        }
      } else {
        body = {
          found_report_id: foundId,
          lost_report_id: lostId,
          match_id: matchId,
          statement: statement.trim(),
          answers: activeQuestions.map((q) => ({ key: q.key, answer: (answers[q.key] ?? "").trim() })).filter((a) => a.answer !== ""),
          evidence_media_ids: readyIds(photos),
          preferred_visit: visit.trim() || undefined,
        }
      }

      return call(
        api.POST("/claims", {
          params: { header: { "Idempotency-Key": key.current } },
          body,
        }),
      )
    },
    onSuccess: (claim) => router.push(`/klaim/${claim.code}`),
    onError: (err) => {
      const fields = fieldErrors(err)
      if (Object.keys(fields).length > 0) {
        setErrors(fields)
        setFormError(undefined)
      } else {
        setFormError(messageFor(err, te))
      }
    },
  })

  function validate(): Record<string, string> {
    const found: Record<string, string> = {}
    if (isNewLost) {
      const nameLen = newLostItemName.trim().length
      if (nameLen < 3 || nameLen > 80) {
        found["lost_report.item_name"] = nameLen === 0 ? te("required") : te("tooShort", { min: 3, count: nameLen })
      }
      if (!newLostCategoryId) {
        found["lost_report.category_id"] = te("required")
      }
      const descLen = newLostDescription.trim().length
      if (descLen < 20 || descLen > 1000) {
        found["lost_report.description"] = descLen === 0 ? te("required") : te("tooShort", { min: 20, count: descLen })
      }
      if (!newLostDate) {
        found["lost_report.date"] = te("required")
      }
      if (newLostBrand.trim().length > 40) {
        found["lost_report.brand"] = te("tooLong", { max: 40 })
      }
      if (newLostPrivateDetails.trim().length > 500) {
        found["lost_report.private_details"] = te("tooLong", { max: 500 })
      }
    } else {
      if (!lostId) {
        found.lost_report_id = t("lostRequired")
      }
    }

    const length = statement.trim().length
    if (length < 30 || length > 1000) {
      found.statement = t("statementLength")
    }
    for (const q of activeQuestions) {
      if (q.required && !(answers[q.key] ?? "").trim()) {
        found[`answers.${q.key}`] = te("required")
      }
    }
    return found
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const found = validate()
    setErrors(found)
    setFormError(undefined)
    if (Object.keys(found).length === 0) {
      submit.mutate()
    }
  }

  return (
    <form onSubmit={onSubmit} noValidate className="flex flex-col gap-6">
      {formError ? (
        <Alert variant="destructive">
          <TriangleAlert aria-hidden />
          <AlertTitle>{t("failed")}</AlertTitle>
          <AlertDescription>{formError}</AlertDescription>
        </Alert>
      ) : null}

      {!hasExistingLost && !presetLostId ? (
        <Alert className="border-primary/30 bg-primary-soft/50">
          <Info className="size-4 text-primary" aria-hidden />
          <AlertTitle className="text-sm font-semibold">{t("inlineLostTitle")}</AlertTitle>
          <AlertDescription className="text-sm text-muted-foreground">{t("inlineLostLead")}</AlertDescription>
        </Alert>
      ) : null}

      {presetLostId ? null : hasExistingLost ? (
        <Field id="claim-lost" label={t("lostLabel")} hint={t("lostHint")} error={errors.lost_report_id}>
          {(aria) => (
            <NativeSelect
              {...aria}
              value={isNewLost ? "new" : lostId}
              onChange={(e) => {
                if (e.target.value === "new") {
                  setIsNewLost(true)
                } else {
                  setIsNewLost(false)
                  setLostId(e.target.value)
                }
              }}
            >
              {lostOptions.map((o) => (
                <option key={o.id} value={o.id}>
                  {o.label}
                </option>
              ))}
              <option value="new">{t("createNewLost")}</option>
            </NativeSelect>
          )}
        </Field>
      ) : null}

      {isNewLost ? (
        <section aria-labelledby="inline-lost-heading" className="surface flex flex-col gap-4 p-4">
          <div>
            <h2 id="inline-lost-heading" className="font-semibold">
              {t("inlineSectionTitle")}
            </h2>
            <p className="text-sm text-muted-foreground">{t("inlineSectionLead")}</p>
          </div>

          <Field id="claim-lost-name" label={t("lostNameLabel")} hint={t("lostNameHint")} error={errors["lost_report.item_name"]}>
            {(aria) => <Input {...aria} maxLength={80} value={newLostItemName} onChange={(e) => setNewLostItemName(e.target.value)} />}
          </Field>

          {leafCategories.length > 0 ? (
            <Field id="claim-lost-category" label={t("lostCategoryLabel")} hint={t("lostCategoryHint")} error={errors["lost_report.category_id"]}>
              {(aria) => (
                <NativeSelect {...aria} value={newLostCategoryId} onChange={(e) => setNewLostCategoryId(e.target.value)}>
                  <option value="">{t("lostCategoryHint")}</option>
                  {leafCategories.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.name} ({c.parentName})
                    </option>
                  ))}
                </NativeSelect>
              )}
            </Field>
          ) : null}

          <Field id="claim-lost-date" label={t("lostDateLabel")} hint={t("lostDateHint")} error={errors["lost_report.date"]}>
            {(aria) => <Input {...aria} type="date" max={todayInWib()} value={newLostDate} onChange={(e) => setNewLostDate(e.target.value)} />}
          </Field>

          {colours && colours.length > 0 ? (
            <Field
              id="claim-lost-colours"
              as="fieldset"
              label={t("lostColoursLabel")}
              optionalLabel={t("optional")}
              hint={t("lostColoursHint")}
              error={errors["lost_report.colours"]}
            >
              {() => <ColourPicker palette={colours} value={newLostColours} onChange={setNewLostColours} />}
            </Field>
          ) : null}

          <Field id="claim-lost-desc" label={t("lostDescriptionLabel")} hint={t("lostDescriptionHint")} error={errors["lost_report.description"]}>
            {(aria) => <Textarea {...aria} rows={4} maxLength={1000} value={newLostDescription} onChange={(e) => setNewLostDescription(e.target.value)} />}
          </Field>

          <Field id="claim-lost-brand" label={t("lostBrandLabel")} optionalLabel={t("optional")} error={errors["lost_report.brand"]}>
            {(aria) => <Input {...aria} maxLength={40} value={newLostBrand} onChange={(e) => setNewLostBrand(e.target.value)} />}
          </Field>

          <Field
            id="claim-lost-private"
            label={t("lostPrivateLabel")}
            optionalLabel={t("optional")}
            hint={t("lostPrivateHint")}
            error={errors["lost_report.private_details"]}
          >
            {(aria) => <Input {...aria} maxLength={500} value={newLostPrivateDetails} onChange={(e) => setNewLostPrivateDetails(e.target.value)} />}
          </Field>
        </section>
      ) : null}

      <Field id="claim-statement" label={t("statementLabel", { name: foundName })} hint={t("statementHint")} error={errors.statement}>
        {(aria) => <Textarea {...aria} rows={5} maxLength={1000} value={statement} onChange={(e) => setStatement(e.target.value)} />}
      </Field>

      {activeQuestions.length > 0 ? (
        <section aria-labelledby="claim-questions" className="surface flex flex-col gap-4 p-4">
          <div>
            <h2 id="claim-questions" className="font-semibold">
              {t("questionsTitle")}
            </h2>
            <p className="text-sm text-muted-foreground">{t("questionsLead")}</p>
          </div>
          {activeQuestions.map((q) => (
            <Field
              key={q.key}
              id={`claim-q-${q.key}`}
              label={q.question}
              optionalLabel={q.required ? undefined : t("optional")}
              error={errors[`answers.${q.key}`]}
            >
              {(aria) => (
                <Input {...aria} maxLength={500} value={answers[q.key] ?? ""} onChange={(e) => setAnswers((prev) => ({ ...prev, [q.key]: e.target.value }))} />
              )}
            </Field>
          ))}
        </section>
      ) : null}

      <Field
        id="claim-evidence"
        as="fieldset"
        label={t("evidenceLabel")}
        optionalLabel={t("optional")}
        hint={t("evidenceHint")}
        error={errors.evidence_media_ids}
      >
        {() => <PhotoPicker purpose="claim_evidence" max={maxEvidence} items={photos} setItems={setPhotos} />}
      </Field>

      <Field id="claim-visit" label={t("visitLabel")} optionalLabel={t("optional")} hint={t("visitHint")} error={errors.preferred_visit}>
        {(aria) => <Input {...aria} maxLength={100} value={visit} onChange={(e) => setVisit(e.target.value)} />}
      </Field>

      <div className="flex flex-wrap gap-2">
        <Button type="submit" disabled={submit.isPending || busy(photos)}>
          {submit.isPending ? t("sending") : t("send")}
        </Button>
        <Button type="button" variant="ghost" onClick={() => router.back()}>
          {t("cancel")}
        </Button>
      </div>
    </form>
  )
}
