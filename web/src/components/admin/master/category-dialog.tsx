"use client"

import { Plus, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { useState } from "react"

import { Field } from "@/components/report/form/field"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"
import type { AdminCategory, Sensitivity } from "@/lib/api/types"

import { MasterDialog } from "./master-dialog"

const sensitivities: Sensitivity[] = ["normal", "valuable", "identity"]

type Question = { key: string; question: string; required: boolean; check_hint: string }

const blank: Question = { key: "", question: "", required: true, check_hint: "" }

type Props = {
  category?: AdminCategory
  // creating a leaf needs its parent, creating a parent needs none
  parents: AdminCategory[]
  leaf: boolean
}

// FR-MD-01: a leaf carries sensitivity, retention, verification questions and the prompts of the photo classifier
export function CategoryDialog({ category, parents, leaf }: Props) {
  const t = useTranslations("admin.master")
  const ts = useTranslations("status.sensitivity")
  const [slug, setSlug] = useState("")
  const [name, setName] = useState("")
  const [parent, setParent] = useState("")
  const [sensitivity, setSensitivity] = useState<Sensitivity>("normal")
  const [retention, setRetention] = useState("")
  const [prompts, setPrompts] = useState("")
  const [questions, setQuestions] = useState<Question[]>([blank])
  const [order, setOrder] = useState("0")
  const leafBody = () => ({
    sensitivity,
    clip_prompts: prompts
      .split("\n")
      .map((p) => p.trim())
      .filter(Boolean),
    verification_questions: questions.map((q) => ({ ...q, check_hint: q.check_hint || undefined })),
  })
  return (
    <MasterDialog
      editing={category?.name}
      createLabel={leaf ? t("categories.addLeaf") : t("categories.addParent")}
      title={category ? t("editTitle", { name: category.name }) : leaf ? t("categories.addLeaf") : t("categories.addParent")}
      description={leaf ? t("categories.questionsHint") : undefined}
      reset={() => {
        setSlug(category?.slug ?? "")
        setName(category?.name ?? "")
        setParent(category?.parent_id ?? parents.find((p) => p.active)?.id ?? "")
        setSensitivity(category?.sensitivity ?? "normal")
        setRetention(category?.retention_days ? String(category.retention_days) : "")
        setPrompts(category?.clip_prompts.join("\n") ?? "")
        setQuestions(
          category && category.verification_questions.length > 0
            ? category.verification_questions.map((q) => ({ key: q.key, question: q.question, required: q.required, check_hint: q.check_hint ?? "" }))
            : leaf
              ? [blank]
              : [],
        )
        setOrder(String(category?.sort_order ?? 0))
      }}
      save={() => {
        const days = retention.trim() === "" ? undefined : Number(retention)
        if (category) {
          return call(
            api.PATCH("/admin/categories/{id}", {
              params: { path: { id: category.id } },
              body: { name, sort_order: Number(order), ...(leaf ? { ...leafBody(), retention_days: days ?? 0 } : {}) },
            }),
          )
        }
        return call(
          api.POST("/admin/categories", {
            body: { slug, name, sort_order: Number(order), ...(leaf ? { ...leafBody(), parent_id: parent, retention_days: days } : {}) },
          }),
        )
      }}
    >
      {(errors) => (
        <>
          {category ? null : (
            <Field id="cat-slug" label={t("slug")} hint={t("slugHint")} error={errors.slug}>
              {(aria) => <Input {...aria} required maxLength={60} value={slug} onChange={(e) => setSlug(e.target.value)} />}
            </Field>
          )}
          {leaf && !category ? (
            <Field id="cat-parent" label={t("categories.parent")} error={errors.parent_id}>
              {(aria) => (
                <NativeSelect {...aria} value={parent} onChange={(e) => setParent(e.target.value)}>
                  {parents.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name}
                    </option>
                  ))}
                </NativeSelect>
              )}
            </Field>
          ) : null}
          <div className="grid gap-4 sm:grid-cols-[1fr_8rem]">
            <Field id="cat-name" label={t("name")} error={errors.name}>
              {(aria) => <Input {...aria} required maxLength={80} value={name} onChange={(e) => setName(e.target.value)} />}
            </Field>
            <Field id="cat-order" label={t("categories.sortOrder")} error={errors.sort_order}>
              {(aria) => <Input {...aria} type="number" min={0} max={1000} value={order} onChange={(e) => setOrder(e.target.value)} />}
            </Field>
          </div>
          {leaf ? (
            <>
              <div className="grid gap-4 sm:grid-cols-2">
                <Field id="cat-sensitivity" label={t("categories.sensitivity")} error={errors.sensitivity}>
                  {(aria) => (
                    <NativeSelect {...aria} value={sensitivity} onChange={(e) => setSensitivity(e.target.value as Sensitivity)}>
                      {sensitivities.map((s) => (
                        <option key={s} value={s}>
                          {ts(s)}
                        </option>
                      ))}
                    </NativeSelect>
                  )}
                </Field>
                <Field id="cat-retention" label={t("categories.retention")} hint={t("categories.retentionHint")} error={errors.retention_days}>
                  {(aria) => <Input {...aria} type="number" min={1} max={365} value={retention} onChange={(e) => setRetention(e.target.value)} />}
                </Field>
              </div>
              <Field id="cat-prompts" label={t("categories.prompts")} hint={t("categories.promptsHint")} error={errors.clip_prompts}>
                {(aria) => <Textarea {...aria} rows={3} value={prompts} onChange={(e) => setPrompts(e.target.value)} />}
              </Field>
              <fieldset className="flex flex-col gap-3">
                <legend className="mb-1 text-sm font-medium">{t("categories.questions")}</legend>
                {errors.verification_questions ? <p className="text-sm text-destructive">{errors.verification_questions}</p> : null}
                {questions.map((q, i) => {
                  const set = (patch: Partial<Question>) => setQuestions((prev) => prev.map((x, j) => (j === i ? { ...x, ...patch } : x)))
                  return (
                    <div key={i} className="flex flex-col gap-2 rounded-lg border p-3">
                      <div className="grid gap-2 sm:grid-cols-[10rem_1fr]">
                        <Field id={`q-key-${i}`} label={t("categories.questionKey")}>
                          {(aria) => <Input {...aria} required maxLength={40} value={q.key} onChange={(e) => set({ key: e.target.value })} />}
                        </Field>
                        <Field id={`q-text-${i}`} label={t("categories.questionText")}>
                          {(aria) => <Input {...aria} required maxLength={200} value={q.question} onChange={(e) => set({ question: e.target.value })} />}
                        </Field>
                      </div>
                      <Field id={`q-hint-${i}`} label={t("categories.questionHint")}>
                        {(aria) => <Input {...aria} maxLength={200} value={q.check_hint} onChange={(e) => set({ check_hint: e.target.value })} />}
                      </Field>
                      <div className="flex items-center justify-between gap-2">
                        <label className="flex min-h-11 items-center gap-2 text-sm">
                          <input type="checkbox" checked={q.required} onChange={(e) => set({ required: e.target.checked })} className="size-4 accent-primary" />
                          {t("categories.questionRequired")}
                        </label>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => setQuestions((prev) => prev.filter((_, j) => j !== i))}
                          aria-label={t("categories.removeQuestion", { n: i + 1 })}
                        >
                          <Trash2 aria-hidden />
                        </Button>
                      </div>
                    </div>
                  )
                })}
                <Button type="button" variant="outline" className="w-fit" onClick={() => setQuestions((prev) => [...prev, blank])}>
                  <Plus aria-hidden />
                  {t("categories.addQuestion")}
                </Button>
              </fieldset>
            </>
          ) : null}
        </>
      )}
    </MasterDialog>
  )
}
