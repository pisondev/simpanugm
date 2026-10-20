"use client"

import { useMutation } from "@tanstack/react-query"
import { ArchiveX, ArrowRightLeft, CalendarPlus } from "lucide-react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api/client"
import { call, messageFor } from "@/lib/api/problem"
import type { Post, Sensitivity } from "@/lib/api/types"

const dispositionKinds = ["donated", "destroyed", "transferred_to_unit", "returned_to_finder"] as const

type CustodyActionReport = { id: string; code: string; category: { sensitivity: Sensitivity } }

export function CustodyManagementActions({
  report,
  posts,
  currentPostId,
  canDispose,
}: {
  report: CustodyActionReport
  posts: Post[]
  currentPostId: string
  canDispose: boolean
}) {
  const t = useTranslations("officer.custodyActions")
  return (
    <div className="flex flex-wrap gap-2">
      <CustodyTransferDialog report={report} posts={posts} currentPostId={currentPostId} />
      <RetentionExtensionDialog report={report} />
      {canDispose ? <CustodyDispositionDialog report={report} /> : null}
      <span className="sr-only">{t("available")}</span>
    </div>
  )
}

function CustodyTransferDialog({ report, posts, currentPostId }: { report: CustodyActionReport; posts: Post[]; currentPostId: string }) {
  const t = useTranslations("officer.custodyActions.transfer")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const [postId, setPostId] = useState("")
  const [note, setNote] = useState("")
  const [error, setError] = useState<string>()
  const destinations = posts.filter((post) => post.id !== currentPostId)
  const selected = destinations.find((post) => post.id === postId)
  const transfer = useMutation({
    mutationFn: () =>
      call(
        api.POST("/found-reports/{id}/transfer", {
          params: { path: { id: report.code } },
          body: { to_post_id: postId, note: note.trim() || undefined },
        }),
      ),
    onSuccess: () => {
      setOpen(false)
      setPostId("")
      setNote("")
      toast.success(t("done"))
      router.refresh()
    },
    onError: (err) => setError(messageFor(err, te)),
  })

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger render={<Button variant="outline" size="sm" disabled={destinations.length === 0} />}>
        <ArrowRightLeft aria-hidden />
        {t("trigger")}
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("title")}</AlertDialogTitle>
          <AlertDialogDescription>{t("lead", { code: report.code })}</AlertDialogDescription>
        </AlertDialogHeader>
        <form
          className="flex flex-col gap-3 py-2"
          onSubmit={(event) => {
            event.preventDefault()
            setError(undefined)
            if (!postId) {
              setError(t("choosePostError"))
              return
            }
            transfer.mutate()
          }}
        >
          <div className="flex flex-col gap-2">
            <label htmlFor={`transfer-post-${report.id}`} className="text-sm font-medium">
              {t("postLabel")}
            </label>
            <NativeSelect id={`transfer-post-${report.id}`} value={postId} onChange={(event) => setPostId(event.target.value)}>
              <option value="">{t("postPlaceholder")}</option>
              {destinations.map((post) => (
                <option key={post.id} value={post.id}>
                  {post.name}
                </option>
              ))}
            </NativeSelect>
          </div>
          <div className="flex flex-col gap-2">
            <label htmlFor={`transfer-note-${report.id}`} className="text-sm font-medium">
              {t("noteLabel")}
            </label>
            <Textarea id={`transfer-note-${report.id}`} rows={3} maxLength={500} value={note} onChange={(event) => setNote(event.target.value)} />
          </div>
          {error ? (
            <p className="text-sm text-destructive" aria-live="polite">
              {error}
            </p>
          ) : null}
          <AlertDialogFooter className="mt-2">
            <AlertDialogCancel>{tc("cancel")}</AlertDialogCancel>
            <Button type="submit" disabled={transfer.isPending || !selected}>
              {transfer.isPending ? tc("saving") : t("submit", { post: selected?.name ?? "" })}
            </Button>
          </AlertDialogFooter>
        </form>
      </AlertDialogContent>
    </AlertDialog>
  )
}

function RetentionExtensionDialog({ report }: { report: CustodyActionReport }) {
  const t = useTranslations("officer.custodyActions.retention")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const [days, setDays] = useState("7")
  const [reason, setReason] = useState("")
  const [error, setError] = useState<string>()
  const extend = useMutation({
    mutationFn: () =>
      call(
        api.POST("/found-reports/{id}/extend-retention", {
          params: { path: { id: report.code } },
          body: { days: Number(days), reason: reason.trim() },
        }),
      ),
    onSuccess: () => {
      setOpen(false)
      setReason("")
      toast.success(t("done"))
      router.refresh()
    },
    onError: (err) => setError(messageFor(err, te)),
  })

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger render={<Button variant="outline" size="sm" />}>
        <CalendarPlus aria-hidden />
        {t("trigger")}
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("title")}</AlertDialogTitle>
          <AlertDialogDescription>{t("lead", { code: report.code })}</AlertDialogDescription>
        </AlertDialogHeader>
        <form
          className="flex flex-col gap-3 py-2"
          onSubmit={(event) => {
            event.preventDefault()
            setError(undefined)
            if (Number(days) < 1 || Number(days) > 30 || reason.trim().length < 5) {
              setError(t("validation"))
              return
            }
            extend.mutate()
          }}
        >
          <div className="flex flex-col gap-2">
            <label htmlFor={`retention-days-${report.id}`} className="text-sm font-medium">
              {t("daysLabel")}
            </label>
            <Input id={`retention-days-${report.id}`} type="number" min={1} max={30} value={days} onChange={(event) => setDays(event.target.value)} />
          </div>
          <div className="flex flex-col gap-2">
            <label htmlFor={`retention-reason-${report.id}`} className="text-sm font-medium">
              {t("reasonLabel")}
            </label>
            <Textarea
              id={`retention-reason-${report.id}`}
              rows={3}
              minLength={5}
              maxLength={500}
              value={reason}
              onChange={(event) => setReason(event.target.value)}
            />
          </div>
          {error ? (
            <p className="text-sm text-destructive" aria-live="polite">
              {error}
            </p>
          ) : null}
          <AlertDialogFooter className="mt-2">
            <AlertDialogCancel>{tc("cancel")}</AlertDialogCancel>
            <Button type="submit" disabled={extend.isPending || Number(days) < 1 || Number(days) > 30 || reason.trim().length < 5}>
              {extend.isPending ? tc("saving") : t("submit")}
            </Button>
          </AlertDialogFooter>
        </form>
      </AlertDialogContent>
    </AlertDialog>
  )
}

// US-33, FR-CUST-06: record disposition for unclaimed items
export function CustodyDispositionDialog({ report }: { report: { code: string; category: { sensitivity: Sensitivity } } }) {
  const t = useTranslations("officer.disposition")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const isIdentity = report.category.sensitivity === "identity"
  const [kind, setKind] = useState<string>(isIdentity ? "transferred_to_unit" : "donated")
  const [recipient, setRecipient] = useState("")
  const [note, setNote] = useState("")
  const [error, setError] = useState<string>()

  const dispose = useMutation({
    mutationFn: () =>
      call(
        api.POST("/found-reports/{id}/dispose", {
          params: { path: { id: report.code } },
          body: {
            kind: kind as (typeof dispositionKinds)[number],
            note: note.trim(),
            recipient: recipient.trim() || undefined,
          },
        }),
      ),
    onSuccess: () => {
      setOpen(false)
      toast.success(t("done"))
      router.refresh()
    },
    onError: (err) => setError(messageFor(err, te)),
  })

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger render={<Button variant="outline" />}>
        <ArchiveX aria-hidden />
        {t("trigger")}
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("title")}</AlertDialogTitle>
          <AlertDialogDescription>{t("lead")}</AlertDialogDescription>
        </AlertDialogHeader>
        <form
          className="flex flex-col gap-3 py-2"
          onSubmit={(e) => {
            e.preventDefault()
            setError(undefined)
            dispose.mutate()
          }}
        >
          <div className="flex flex-col gap-2">
            <label htmlFor="disp-kind" className="text-sm font-medium">
              {t("kindLabel")}
            </label>
            <NativeSelect id="disp-kind" value={kind} disabled={isIdentity} onChange={(e) => setKind(e.target.value)}>
              {isIdentity ? (
                <option value="transferred_to_unit">{t("kinds.transferred_to_unit")}</option>
              ) : (
                dispositionKinds.map((k) => (
                  <option key={k} value={k}>
                    {t(`kinds.${k}`)}
                  </option>
                ))
              )}
            </NativeSelect>
          </div>
          <div className="flex flex-col gap-2">
            <label htmlFor="disp-recipient" className="text-sm font-medium">
              {t("recipientLabel")}
            </label>
            <Input
              id="disp-recipient"
              placeholder={t("recipientPlaceholder")}
              maxLength={100}
              value={recipient}
              onChange={(e) => setRecipient(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-2">
            <label htmlFor="disp-note" className="text-sm font-medium">
              {t("noteLabel")}
            </label>
            <Textarea
              id="disp-note"
              rows={3}
              placeholder={t("notePlaceholder")}
              minLength={5}
              maxLength={500}
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </div>
          {error ? (
            <p className="text-sm text-destructive" aria-live="polite">
              {error}
            </p>
          ) : null}
          <AlertDialogFooter className="mt-2">
            <AlertDialogCancel>{tc("cancel")}</AlertDialogCancel>
            <Button type="submit" disabled={dispose.isPending || note.trim().length < 5}>
              {dispose.isPending ? tc("saving") : t("submit")}
            </Button>
          </AlertDialogFooter>
        </form>
      </AlertDialogContent>
    </AlertDialog>
  )
}
