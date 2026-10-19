"use client"

import { useMutation } from "@tanstack/react-query"
import { TriangleAlert } from "lucide-react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useRef, useState } from "react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api/client"
import { call, fieldErrors, messageFor } from "@/lib/api/problem"
import type { Claim } from "@/lib/api/types"

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
}

const maxEvidence = 3

// FR-CLAIM-01 to FR-CLAIM-04: the statement, the category questions and up to three evidence photos
export function ClaimForm({ foundId, foundName, questions, lostOptions, presetLostId, matchId }: Props) {
  const t = useTranslations("claims.form")
  const te = useTranslations("errors")
  const router = useRouter()
  const key = useRef<string>(crypto.randomUUID())
  const [lostId, setLostId] = useState(presetLostId ?? lostOptions[0]?.id ?? "")
  const [statement, setStatement] = useState("")
  const [answers, setAnswers] = useState<Record<string, string>>({})
  const [visit, setVisit] = useState("")
  const [photos, setPhotos] = useState<PhotoItem[]>([])
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [formError, setFormError] = useState<string>()

  const submit = useMutation({
    mutationFn: (): Promise<Claim> =>
      call(
        api.POST("/claims", {
          params: { header: { "Idempotency-Key": key.current } },
          body: {
            found_report_id: foundId,
            lost_report_id: lostId,
            match_id: matchId,
            statement: statement.trim(),
            answers: questions.map((q) => ({ key: q.key, answer: (answers[q.key] ?? "").trim() })).filter((a) => a.answer !== ""),
            evidence_media_ids: readyIds(photos),
            preferred_visit: visit.trim() || undefined,
          },
        }),
      ),
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
    const length = statement.trim().length
    if (!lostId) {
      found.lost_report_id = t("lostRequired")
    }
    if (length < 30 || length > 1000) {
      found.statement = t("statementLength")
    }
    for (const q of questions) {
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

  if (lostOptions.length === 0) {
    return (
      <Alert>
        <TriangleAlert aria-hidden />
        <AlertTitle>{t("noLostTitle")}</AlertTitle>
        <AlertDescription>
          {t("noLostBody")}{" "}
          <Link href="/lapor/hilang" className="font-medium underline underline-offset-4">
            {t("noLostAction")}
          </Link>
        </AlertDescription>
      </Alert>
    )
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

      {presetLostId ? null : (
        <Field id="claim-lost" label={t("lostLabel")} hint={t("lostHint")} error={errors.lost_report_id}>
          {(aria) => (
            <NativeSelect {...aria} value={lostId} onChange={(e) => setLostId(e.target.value)}>
              {lostOptions.map((o) => (
                <option key={o.id} value={o.id}>
                  {o.label}
                </option>
              ))}
            </NativeSelect>
          )}
        </Field>
      )}

      <Field id="claim-statement" label={t("statementLabel", { name: foundName })} hint={t("statementHint")} error={errors.statement}>
        {(aria) => <Textarea {...aria} rows={5} maxLength={1000} value={statement} onChange={(e) => setStatement(e.target.value)} />}
      </Field>

      {questions.length > 0 ? (
        <section aria-labelledby="claim-questions" className="flex flex-col gap-4 rounded-xl border bg-card p-4">
          <div>
            <h2 id="claim-questions" className="font-semibold">
              {t("questionsTitle")}
            </h2>
            <p className="text-sm text-muted-foreground">{t("questionsLead")}</p>
          </div>
          {questions.map((q) => (
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
