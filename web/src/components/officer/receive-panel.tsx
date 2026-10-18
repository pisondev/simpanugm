"use client"

import { useMutation } from "@tanstack/react-query"
import { CircleCheck, Info, Search } from "lucide-react"
import { useTranslations } from "next-intl"
import { useEffect, useRef, useState } from "react"

import { FormAlert } from "@/components/auth/form-alert"
import { Field } from "@/components/report/form/field"
import { PostPicker } from "@/components/report/form/post-picker"
import { Detail, DetailList } from "@/components/report/detail-list"
import { PhotoStrip } from "@/components/report/photo-strip"
import { StatusBadge } from "@/components/report/status-badge"
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { ButtonLink } from "@/components/ui/button-link"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api/client"
import { call, fieldErrors, isApiError, messageFor } from "@/lib/api/problem"
import type { FoundReport, Post } from "@/lib/api/types"
import { formatDate, formatDateTime } from "@/lib/format"

// US-30: type the code, compare the photos with the item, choose the slot, confirm
export function ReceivePanel({ posts, initialCode }: { posts: Post[]; initialCode?: string }) {
  const t = useTranslations("officer")
  const tr = useTranslations("report")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const ts = useTranslations("status")
  const [code, setCode] = useState(initialCode ?? "")
  const [postId, setPostId] = useState(posts.length === 1 ? (posts[0]?.id ?? "") : "")
  const [slot, setSlot] = useState("")
  const [condition, setCondition] = useState("")
  const [confirm, setConfirm] = useState(false)
  const [slotError, setSlotError] = useState<string>()
  const codeInput = useRef<HTMLInputElement>(null)

  const lookup = useMutation({
    mutationFn: (value: string) => call(api.GET("/found-reports/{id}", { params: { path: { id: value.trim().toUpperCase() } } })),
    onSuccess: (report: FoundReport) => {
      const assigned = posts.find((p) => p.id === report.intended_post?.id)
      if (assigned) {
        setPostId(assigned.id)
      }
    },
  })
  const receive = useMutation({
    mutationFn: (report: FoundReport) =>
      call(
        api.POST("/found-reports/{id}/receive", {
          params: { path: { id: report.id } },
          body: { post_id: postId, storage_slot: slot.trim(), condition_note: condition.trim() || undefined },
        }),
      ),
    onSettled: () => setConfirm(false),
    onError: (err) => {
      const fields = fieldErrors(err)
      if (fields.storage_slot) {
        setSlotError(fields.storage_slot)
      }
    },
  })

  const { mutate: find } = lookup
  useEffect(() => {
    if (initialCode) {
      find(initialCode)
    }
  }, [find, initialCode])

  function restart() {
    lookup.reset()
    receive.reset()
    setCode("")
    setSlot("")
    setCondition("")
    setSlotError(undefined)
    codeInput.current?.focus()
  }

  const report = lookup.data
  const done = receive.data
  if (done) {
    return (
      <div className="flex flex-col items-start gap-4 rounded-xl border border-success/30 bg-success/5 p-5">
        <CircleCheck className="size-8 text-success" aria-hidden />
        <p className="text-lg font-medium" role="status">
          {t("received", {
            post: done.holding_post?.name ?? "",
            slot: done.storage_slot ?? "",
            due: done.retention_due_at ? formatDate(done.retention_due_at) : "",
          })}
        </p>
        <div className="flex flex-wrap gap-2">
          <Button onClick={restart}>{t("receiveAnother")}</Button>
          <ButtonLink href={`/petugas/laporan/temuan/${done.code}`} variant="outline">
            {t("openRecord", { code: done.code })}
          </ButtonLink>
        </div>
      </div>
    )
  }

  const notFound = isApiError(lookup.error, 404) || isApiError(lookup.error, 400)
  const receivable = report?.custody_status === "with_finder"
  const chosen = posts.find((p) => p.id === postId)
  return (
    <div className="flex flex-col gap-6">
      <form
        className="flex flex-col gap-2 rounded-xl border bg-card p-4"
        onSubmit={(e) => {
          e.preventDefault()
          if (code.trim()) {
            receive.reset()
            lookup.mutate(code)
          }
        }}
      >
        <label htmlFor="receive-code" className="text-sm font-medium">
          {t("codeLabel")}
        </label>
        <div className="flex gap-2">
          <Input
            ref={codeInput}
            id="receive-code"
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder={t("codePlaceholder")}
            autoComplete="off"
            autoCapitalize="characters"
            spellCheck={false}
            className="font-mono uppercase"
            autoFocus
          />
          <Button type="submit" disabled={lookup.isPending}>
            <Search aria-hidden />
            {t("lookup")}
          </Button>
        </div>
        <p aria-live="polite" className="text-sm text-destructive">
          {lookup.error ? (notFound ? t("notFound") : messageFor(lookup.error, te)) : ""}
        </p>
      </form>

      {report ? (
        <div className="grid gap-6 lg:grid-cols-2">
          <section className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-mono text-sm font-semibold">{report.code}</span>
              <StatusBadge kind="found" status={report.status} />
              <StatusBadge kind="custody" status={report.custody_status} />
            </div>
            <h2 className="text-xl font-semibold">{report.item_name}</h2>
            <PhotoStrip photos={report.photos} alt={tc("photoAlt", { kind: tc("photoKindFound"), name: report.item_name })} />
            <div className="rounded-xl border bg-card p-4">
              <DetailList>
                <Detail label={tr("category")}>
                  {report.category.name} | {ts(`sensitivity.${report.category.sensitivity}`)}
                </Detail>
                <Detail label={tr("description")}>{report.description}</Detail>
                {report.hidden_details ? <Detail label={tr("hiddenDetails")}>{report.hidden_details}</Detail> : null}
                <Detail label={tr("colours")}>{report.colours.join(", ") || tc("emptyValue")}</Detail>
                <Detail label={tr("foundAt")}>{formatDateTime(report.found_at)}</Detail>
                <Detail label={tr("foundLocation")}>{report.found_location.name}</Detail>
                <Detail label={tr("finder")}>{report.finder?.full_name ?? report.proxy_finder_name ?? tc("emptyValue")}</Detail>
                {report.intended_post ? <Detail label={tr("intendedPost")}>{report.intended_post.name}</Detail> : null}
              </DetailList>
            </div>
          </section>
          <section className="flex flex-col gap-5">
            {receivable ? (
              <form
                className="flex flex-col gap-5"
                onSubmit={(e) => {
                  e.preventDefault()
                  if (!slot.trim()) {
                    setSlotError(te("required"))
                    return
                  }
                  setSlotError(undefined)
                  setConfirm(true)
                }}
              >
                <div className="flex items-start gap-3 rounded-xl border border-info/30 bg-primary-soft/50 p-4">
                  <Info className="mt-0.5 size-5 shrink-0 text-info" aria-hidden />
                  <p>{t("compare")}</p>
                </div>
                <Field id="receive-post" as="fieldset" label={t("post")}>
                  {(aria) => (
                    <PostPicker name="post" posts={posts} value={postId} onChange={setPostId} describedBy={aria["aria-describedby"]} showNearest={false} />
                  )}
                </Field>
                <Field id="receive-slot" label={t("slot")} error={slotError}>
                  {(aria) => <Input {...aria} value={slot} onChange={(e) => setSlot(e.target.value)} placeholder={t("slotPlaceholder")} maxLength={40} />}
                </Field>
                <Field id="receive-condition" label={t("condition")} optionalLabel={tc("optional")}>
                  {(aria) => (
                    <Textarea
                      {...aria}
                      rows={3}
                      value={condition}
                      onChange={(e) => setCondition(e.target.value)}
                      placeholder={t("conditionPlaceholder")}
                      maxLength={500}
                    />
                  )}
                </Field>
                <FormAlert>
                  {receive.error && !fieldErrors(receive.error).storage_slot
                    ? isApiError(receive.error, 403)
                      ? t("notAssigned")
                      : messageFor(receive.error, te)
                    : null}
                </FormAlert>
                <Button type="submit" size="lg" disabled={!postId || receive.isPending}>
                  {t("confirmReceive")}
                </Button>
              </form>
            ) : (
              <FormAlert>{t("alreadyReceived", { status: ts(`custody.${report.custody_status}`) })}</FormAlert>
            )}
          </section>
        </div>
      ) : null}

      <AlertDialog open={confirm} onOpenChange={setConfirm}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("confirmTitle", { post: chosen?.name ?? "" })}</AlertDialogTitle>
            <AlertDialogDescription>{t("confirmBody")}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tc("cancel")}</AlertDialogCancel>
            <Button onClick={() => report && receive.mutate(report)} disabled={receive.isPending}>
              {t("confirmAction")}
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
