"use client"

import { LoaderCircle, Sparkles } from "lucide-react"
import { useTranslations } from "next-intl"
import { useEffect, useState } from "react"

const searchingWindowMs = 2 * 60_000

export function CandidateStatus({ candidateCount, createdAt, now }: { candidateCount: number; createdAt: string; now: number }) {
  const t = useTranslations("report")
  const initialSearching = candidateCount === 0 && now - new Date(createdAt).getTime() < searchingWindowMs
  const [searching, setSearching] = useState(initialSearching)

  useEffect(() => {
    if (!initialSearching) return
    const remaining = Math.max(0, searchingWindowMs - (now - new Date(createdAt).getTime()))
    const timer = window.setTimeout(() => setSearching(false), remaining)
    return () => window.clearTimeout(timer)
  }, [createdAt, initialSearching, now])

  if (candidateCount > 0) {
    return (
      <>
        <Sparkles className="mt-0.5 size-5 shrink-0 text-info" aria-hidden />
        <p>{t("candidateCount", { count: candidateCount })}</p>
      </>
    )
  }
  if (searching) {
    return (
      <>
        <LoaderCircle className="mt-0.5 size-5 shrink-0 animate-spin text-info" aria-hidden />
        <p>{t("searching")}</p>
      </>
    )
  }
  return (
    <>
      <Sparkles className="mt-0.5 size-5 shrink-0 text-info" aria-hidden />
      <p>{t("noCandidates")}</p>
    </>
  )
}
