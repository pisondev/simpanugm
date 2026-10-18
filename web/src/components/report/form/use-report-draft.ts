"use client"

import { useTranslations } from "next-intl"
import { type Dispatch, type SetStateAction, useEffect, useRef, useState } from "react"
import type { DefaultValues, FieldValues, UseFormReturn } from "react-hook-form"
import { toast } from "sonner"

import { idempotencyKey } from "@/lib/api/client"
import { clearDraft, loadDraft, saveDraft } from "@/lib/draft"

import { type PhotoItem, photoFromId, readyIds } from "./photo-picker"

type Draft<V> = { values: V; photos: string[]; key: string }

type Options<V extends FieldValues> = {
  enabled: boolean
  draftKey: string
  form: UseFormReturn<V>
  photos: PhotoItem[]
  setPhotos: Dispatch<SetStateAction<PhotoItem[]>>
  empty: () => V
  onDiscard: () => void
}

// US-08 criteria 4 and 5: the draft survives a reload and the idempotency key survives with it,
// so a retry after a dropped connection can never create a second report
export function useReportDraft<V extends FieldValues>({ enabled, draftKey, form, photos, setPhotos, empty, onDiscard }: Options<V>) {
  const t = useTranslations("form")
  const key = useRef("")
  const [restored, setRestored] = useState(false)
  const { reset, getValues, watch } = form

  useEffect(() => {
    key.current = idempotencyKey()
    if (!enabled) {
      return
    }
    const draft = loadDraft<Draft<V>>(draftKey)
    // local storage exists only after hydration, so the one time read happens here and not in render
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setRestored(true)
    if (!draft?.values) {
      return
    }
    reset({ ...empty(), ...draft.values } as DefaultValues<V>)
    setPhotos(draft.photos.map((id) => photoFromId(id, true)))
    key.current = draft.key || key.current
    toast(t("draftRestored"), {
      id: draftKey,
      action: {
        label: t("discardDraft"),
        onClick: () => {
          clearDraft(draftKey)
          reset(empty() as DefaultValues<V>)
          setPhotos([])
          key.current = idempotencyKey()
          onDiscard()
        },
      },
    })
    // the restore runs once per mount, later changes to the callbacks must not restore again
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draftKey, enabled])

  // saving starts only after the restore, so an empty first render never overwrites a stored draft
  useEffect(() => {
    if (!enabled || !restored) {
      return
    }
    const save = () => saveDraft<Draft<V>>(draftKey, { values: getValues(), photos: readyIds(photos), key: key.current })
    save()
    const sub = watch(save)
    return () => sub.unsubscribe()
  }, [draftKey, enabled, getValues, photos, restored, watch])

  return { key, clear: () => clearDraft(draftKey) }
}
