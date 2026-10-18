"use client"

import { useQuery } from "@tanstack/react-query"
import { useCallback, useEffect, useRef, useState } from "react"

import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"
import type { AssistOutcome, AssistSuggestions, CategorySuggestion } from "@/lib/api/types"

// NFR-PERF-04: a suggestion that takes longer than this is abandoned without a word
const abandonAfterMs = 8_000

export type AssistFields = { category_id: string; colours: string[]; brand: string; description: string }

type Applied = Partial<{ category: string; colours: string[]; brand: string; description: string }>

type Options = {
  enabled: boolean
  reportType: "lost" | "found"
  // the first photo that finished uploading, the assistant looks at that one
  mediaId?: string
  read: () => AssistFields
  write: (patch: Partial<AssistFields>) => void
  // found reports also get a drafted public description (AI-04)
  drafts?: boolean
}

export type Assist = {
  status: "idle" | "loading" | "ready" | "unavailable"
  alternatives: CategorySuggestion[]
  // true while the field still holds what the assistant put there
  suggested: (field: keyof Applied, current: unknown) => boolean
  pick: (categoryId: string) => void
  redraft: () => Promise<void>
  outcome: (values: AssistFields) => AssistOutcome | undefined
}

function same(a: string[], b: string[]): boolean {
  return a.length === b.length && a.every((x) => b.includes(x))
}

// AI-03 and AI-04: suggestions arrive as editable defaults, empty fields only, and what became of each is reported back
export function useAssist({ enabled, reportType, mediaId, read, write, drafts = false }: Options): Assist {
  const [applied, setApplied] = useState<Applied>({})
  const done = useRef<string>(undefined)
  const query = useQuery({
    queryKey: ["assist", reportType, mediaId],
    enabled: enabled && Boolean(mediaId),
    retry: false,
    staleTime: Number.POSITIVE_INFINITY,
    queryFn: ({ signal }) =>
      call(
        api.POST("/assist/analyze-image", {
          body: { media_id: mediaId as string, report_type: reportType },
          signal: AbortSignal.any([signal, AbortSignal.timeout(abandonAfterMs)]),
        }),
      ),
  })

  const draftFor = useCallback(async (s: AssistSuggestions, values: AssistFields): Promise<string | undefined> => {
    if (!values.category_id) return undefined
    const identity = s.categories.find((c) => c.category.id === values.category_id)?.category.sensitivity === "identity"
    try {
      const out = await call(
        api.POST("/assist/describe", {
          body: {
            category_id: values.category_id,
            colours: values.colours,
            brand: values.brand || undefined,
            visible_text: identity ? undefined : (s.visible_text ?? undefined),
          },
        }),
      )
      return out.description
    } catch {
      return undefined
    }
  }, [])

  useEffect(() => {
    const s = query.data
    if (!s || !mediaId || done.current === mediaId) return
    done.current = mediaId
    const now = read()
    const next: Applied = {}
    const patch: Partial<AssistFields> = {}
    const top = s.categories[0]
    if (top && !now.category_id) {
      patch.category_id = top.category.id
      next.category = top.category.id
    }
    if (s.colours.length > 0 && now.colours.length === 0) {
      patch.colours = s.colours
      next.colours = s.colours
    }
    if (s.brand && !now.brand) {
      patch.brand = s.brand
      next.brand = s.brand
    }
    write(patch)
    setApplied(next)
    if (drafts && !now.description) {
      void draftFor(s, { ...now, ...patch }).then((text) => {
        if (text && !read().description) {
          write({ description: text })
          setApplied((prev) => ({ ...prev, description: text }))
        }
      })
    }
  }, [query.data, mediaId, read, write, drafts, draftFor])

  const status: Assist["status"] = !enabled || !mediaId ? "idle" : query.isError ? "unavailable" : query.data ? "ready" : "loading"
  const chosen = read().category_id
  return {
    status,
    alternatives: (query.data?.categories ?? [])
      .slice(0, 3)
      .filter((c) => c.category.id !== chosen)
      .slice(0, 2),
    suggested: (field, current) => {
      const value = applied[field]
      if (value === undefined) return false
      return Array.isArray(value) ? Array.isArray(current) && same(value, current as string[]) : value === current
    },
    pick: (categoryId) => {
      write({ category_id: categoryId })
      setApplied((prev) => ({ ...prev, category: prev.category ?? categoryId }))
    },
    redraft: async () => {
      if (!query.data) return
      const text = await draftFor(query.data, read())
      if (text) {
        write({ description: text })
        setApplied((prev) => ({ ...prev, description: text }))
      }
    },
    outcome: (values) => {
      const out: AssistOutcome = {}
      const judge = (applied: boolean, equal: boolean, empty: boolean): AssistOutcome["category"] | undefined =>
        !applied ? undefined : equal ? "accepted" : empty ? "ignored" : "edited"
      out.category = judge(applied.category !== undefined, applied.category === values.category_id, !values.category_id)
      out.colours = judge(applied.colours !== undefined, same(applied.colours ?? [], values.colours), values.colours.length === 0)
      out.brand = judge(applied.brand !== undefined, applied.brand === values.brand, !values.brand)
      out.description = judge(applied.description !== undefined, applied.description === values.description, !values.description)
      const kept = Object.fromEntries(Object.entries(out).filter(([, v]) => v !== undefined)) as AssistOutcome
      return Object.keys(kept).length > 0 ? kept : undefined
    },
  }
}
