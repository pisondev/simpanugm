"use client"

import { Camera, ImagePlus, LoaderCircle, TriangleAlert, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { type Dispatch, type SetStateAction, useEffect, useId, useRef } from "react"

import { Progress } from "@/components/ui/progress"
import { api, mediaUrl, type UploadPurpose, uploadMedia } from "@/lib/api/client"
import { prepareImage } from "@/lib/image"
import { cn } from "@/lib/utils"

export type PhotoItem = {
  key: string
  mediaId?: string
  preview?: string
  progress: number
  state: "preparing" | "uploading" | "done" | "error"
  error?: string
  // uploaded in this session and not yet attached, so removing it may delete it
  staged?: boolean
}

export const maxPhotos = 5
const maxBytes = 10 * 1024 * 1024

export function photoFromId(mediaId: string, staged = false): PhotoItem {
  return { key: mediaId, mediaId, progress: 1, state: "done", staged }
}

export function readyIds(items: PhotoItem[]): string[] {
  return items.filter((p) => p.state === "done" && p.mediaId).map((p) => p.mediaId as string)
}

export function busy(items: PhotoItem[]): boolean {
  return items.some((p) => p.state === "preparing" || p.state === "uploading")
}

type Props = {
  purpose: UploadPurpose
  items: PhotoItem[]
  setItems: Dispatch<SetStateAction<PhotoItem[]>>
  describedBy?: string
  invalid?: boolean
}

// FR-MEDIA-01 to FR-MEDIA-03: camera or gallery, HEIC conversion, resize, per image progress
export function PhotoPicker({ purpose, items, setItems, describedBy, invalid }: Props) {
  const t = useTranslations("form.photos")
  const galleryId = useId()
  const cameraId = useId()
  const previews = useRef(new Set<string>())

  useEffect(() => {
    const urls = previews.current
    return () => urls.forEach((u) => URL.revokeObjectURL(u))
  }, [])

  function patch(key: string, change: Partial<PhotoItem>) {
    setItems((prev) => prev.map((p) => (p.key === key ? { ...p, ...change } : p)))
  }

  async function handle(file: File, key: string) {
    let prepared: File
    try {
      prepared = await prepareImage(file)
    } catch {
      patch(key, { state: "error", error: t("unsupported") })
      return
    }
    if (prepared.size > maxBytes) {
      patch(key, { state: "error", error: t("tooLarge") })
      return
    }
    const preview = URL.createObjectURL(prepared)
    previews.current.add(preview)
    patch(key, { preview, state: "uploading", progress: 0 })
    try {
      const { id } = await uploadMedia(prepared, purpose, (progress) => patch(key, { progress }))
      patch(key, { mediaId: id, state: "done", progress: 1, staged: true })
    } catch (err) {
      const status = (err as { status?: number }).status
      const body = (err as { body?: { detail?: string } }).body
      const message = status === 413 ? t("tooLarge") : (body?.detail ?? t("failed"))
      patch(key, { state: "error", error: message })
    }
  }

  function add(files: FileList | null) {
    if (!files) {
      return
    }
    const room = maxPhotos - items.length
    const picked = Array.from(files).slice(0, Math.max(0, room))
    const fresh = picked.map((file) => ({ file, item: { key: crypto.randomUUID(), progress: 0, state: "preparing" as const } }))
    setItems((prev) => [...prev, ...fresh.map((f) => f.item)])
    for (const f of fresh) {
      void handle(f.file, f.item.key)
    }
  }

  function remove(item: PhotoItem) {
    setItems((prev) => prev.filter((p) => p.key !== item.key))
    if (item.preview) {
      URL.revokeObjectURL(item.preview)
      previews.current.delete(item.preview)
    }
    if (item.staged && item.mediaId) {
      void api.DELETE("/media/{id}", { params: { path: { id: item.mediaId } } })
    }
  }

  const full = items.length >= maxPhotos
  return (
    <div className="flex flex-col gap-3" aria-describedby={describedBy}>
      <ul className="grid grid-cols-3 gap-2 sm:grid-cols-5">
        {items.map((item, i) => (
          <li
            key={item.key}
            className={cn("relative aspect-square overflow-hidden rounded-lg border bg-muted", item.state === "error" && "border-destructive")}
          >
            {item.preview || item.mediaId ? (
              <img src={item.preview ?? mediaUrl(item.mediaId as string, "thumb")} alt="" className="size-full object-cover" />
            ) : null}
            {item.state === "preparing" || item.state === "uploading" ? (
              <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-card/80 p-2 text-center text-xs">
                <LoaderCircle className="size-5 animate-spin text-primary" aria-hidden />
                <span>{item.state === "preparing" ? t("processing") : t("uploading", { percent: Math.round(item.progress * 100) })}</span>
                {item.state === "uploading" ? <Progress value={Math.round(item.progress * 100)} className="w-full" /> : null}
              </div>
            ) : null}
            {item.state === "error" ? (
              <div className="absolute inset-0 flex flex-col items-center justify-center gap-1 bg-card/90 p-2 text-center text-xs text-destructive">
                <TriangleAlert className="size-5" aria-hidden />
                <span>{item.error}</span>
              </div>
            ) : null}
            <button
              type="button"
              onClick={() => remove(item)}
              className="absolute top-1 right-1 flex size-9 items-center justify-center rounded-full bg-foreground/70 text-white hover:bg-foreground"
              aria-label={t("remove", { n: i + 1 })}
            >
              <X className="size-4" aria-hidden />
            </button>
          </li>
        ))}
      </ul>
      {full ? (
        <p className="text-sm text-muted-foreground">{t("limit")}</p>
      ) : (
        <div className="flex flex-wrap gap-2">
          <label
            htmlFor={cameraId}
            className="inline-flex min-h-11 cursor-pointer items-center gap-2 rounded-lg bg-primary px-4 text-base font-medium text-primary-foreground focus-within:ring-3 focus-within:ring-ring/50 hover:bg-primary-hover sm:hidden"
          >
            <Camera className="size-5" aria-hidden />
            {t("camera")}
            <input
              id={cameraId}
              type="file"
              accept="image/*"
              capture="environment"
              className="sr-only"
              onChange={(e) => {
                add(e.target.files)
                e.target.value = ""
              }}
            />
          </label>
          <label
            htmlFor={galleryId}
            className={cn(
              "inline-flex min-h-11 cursor-pointer items-center gap-2 rounded-lg border bg-card px-4 text-base font-medium focus-within:ring-3 focus-within:ring-ring/50 hover:bg-muted",
              invalid && "border-destructive",
            )}
          >
            <ImagePlus className="size-5" aria-hidden />
            {t("add")}
            <input
              id={galleryId}
              type="file"
              accept="image/jpeg,image/png,image/webp,image/heic,image/heif,.heic,.heif"
              multiple
              className="sr-only"
              aria-describedby={describedBy}
              onChange={(e) => {
                add(e.target.files)
                e.target.value = ""
              }}
            />
          </label>
        </div>
      )}
    </div>
  )
}
