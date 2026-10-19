"use client"

import { useMutation } from "@tanstack/react-query"
import { RefreshCw, ThumbsUp } from "lucide-react"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { api } from "@/lib/api/client"
import { call, messageFor } from "@/lib/api/problem"

// FR-AI-09: a positive label from the owner, the evaluation set learns from both answers
export function MaybeMineButton({ matchId }: { matchId: string }) {
  const t = useTranslations("matching")
  const te = useTranslations("errors")
  const [sent, setSent] = useState(false)
  const feedback = useMutation({
    mutationFn: () => call(api.POST("/matches/{id}/feedback", { params: { path: { id: matchId } }, body: { verdict: "relevant" } })),
    onSuccess: () => {
      setSent(true)
      toast.success(t("maybeMineDone"))
    },
    onError: (err) => toast(messageFor(err, te)),
  })
  return (
    <Button variant="outline" onClick={() => feedback.mutate()} disabled={sent || feedback.isPending} aria-pressed={sent}>
      <ThumbsUp aria-hidden />
      {t("maybeMine")}
    </Button>
  )
}

export function RematchButton({ type, id }: { type: "lost" | "found"; id: string }) {
  const t = useTranslations("matching.officer")
  const te = useTranslations("errors")
  const rematch = useMutation({
    mutationFn: () => call(api.POST("/reports/{type}/{id}/rematch", { params: { path: { type, id } } })),
    onSuccess: () => toast.success(t("rematchDone")),
    onError: (err) => toast(messageFor(err, te)),
  })
  return (
    <Button variant="outline" size="sm" onClick={() => rematch.mutate()} disabled={rematch.isPending}>
      <RefreshCw aria-hidden className={rematch.isPending ? "animate-spin" : undefined} />
      {t("rematch")}
    </Button>
  )
}
