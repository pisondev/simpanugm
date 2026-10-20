"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Send } from "lucide-react"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { api } from "@/lib/api/client"
import { call, messageFor } from "@/lib/api/problem"
import { formatDateTime } from "@/lib/format"
import { cn } from "@/lib/utils"

// FR-CLAIM-09: one thread between the claimant and the officers, the finder never sees it
export function ClaimThread({ code, canPost }: { code: string; canPost: boolean }) {
  const t = useTranslations("claims.thread")
  const te = useTranslations("errors")
  const client = useQueryClient()
  const [body, setBody] = useState("")
  const key = ["claim-messages", code]
  const messages = useQuery({
    queryKey: key,
    queryFn: () => call(api.GET("/claims/{id}/messages", { params: { path: { id: code } } })),
    refetchInterval: 30_000,
  })
  const send = useMutation({
    mutationFn: () => call(api.POST("/claims/{id}/messages", { params: { path: { id: code } }, body: { message: body.trim() } })),
    onSuccess: () => {
      setBody("")
      void client.invalidateQueries({ queryKey: key })
    },
    onError: (err) => toast(messageFor(err, te)),
  })
  const items = messages.data?.items ?? []
  return (
    <section aria-labelledby="claim-thread" className="flex flex-col gap-3">
      <h2 id="claim-thread" className="font-semibold">
        {t("title")}
      </h2>
      {messages.isPending ? (
        <p className="text-sm text-muted-foreground">{t("loading")}</p>
      ) : items.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t("empty")}</p>
      ) : (
        <ol className="flex flex-col gap-2" aria-live="polite">
          {items.map((m) => (
            <li
              key={m.id}
              className={cn("surface flex max-w-[85%] flex-col gap-1 p-3", m.mine ? "self-end border-primary/30 bg-primary-soft/50" : "self-start")}
            >
              <p className="text-xs text-muted-foreground">
                {m.mine ? t("you") : m.author_name} | {t(`role.${m.author_role}`)} | {formatDateTime(m.created_at)}
              </p>
              <p className="break-words whitespace-pre-line">{m.body}</p>
            </li>
          ))}
        </ol>
      )}
      {canPost ? (
        <form
          className="flex flex-col gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            if (body.trim()) {
              send.mutate()
            }
          }}
        >
          <label htmlFor="claim-thread-body" className="sr-only">
            {t("label")}
          </label>
          <Textarea id="claim-thread-body" rows={2} maxLength={2000} placeholder={t("placeholder")} value={body} onChange={(e) => setBody(e.target.value)} />
          <Button type="submit" className="w-fit" disabled={send.isPending || !body.trim()}>
            <Send aria-hidden />
            {t("send")}
          </Button>
        </form>
      ) : null}
    </section>
  )
}
