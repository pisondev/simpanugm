"use client"

import { useTranslations } from "next-intl"
import { useState } from "react"

import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog"
import { mediaUrl } from "@/lib/api/client"
import { cn } from "@/lib/utils"

type Photo = { media_id: string; position: number }

// media is served by api with the session cookie and fixed variants, so next/image would add nothing
export function PhotoStrip({ photos, alt }: { photos: Photo[]; alt: string }) {
  const t = useTranslations("report")
  const [open, setOpen] = useState<string | null>(null)
  if (photos.length === 0) {
    return <p className="text-muted-foreground">{t("noPhotos")}</p>
  }
  const sorted = [...photos].sort((a, b) => a.position - b.position)
  const [first, ...rest] = sorted
  return (
    <>
      <div className="grid grid-cols-4 gap-2">
        {first ? (
          <button
            type="button"
            onClick={() => setOpen(first.media_id)}
            className={cn("col-span-4 overflow-hidden rounded-xl bg-muted sm:col-span-2 sm:row-span-2", rest.length === 0 && "sm:col-span-4")}
          >
            <img src={mediaUrl(first.media_id, "display")} alt={alt} className="aspect-[4/3] w-full object-cover" />
          </button>
        ) : null}
        {rest.map((p, i) => (
          <button key={p.media_id} type="button" onClick={() => setOpen(p.media_id)} className="overflow-hidden rounded-lg bg-muted">
            <img src={mediaUrl(p.media_id, "thumb")} alt={`${alt} (${i + 2})`} className="aspect-square w-full object-cover" loading="lazy" />
          </button>
        ))}
      </div>
      <Dialog open={open !== null} onOpenChange={(v) => !v && setOpen(null)}>
        <DialogContent className="max-w-[calc(100%-1rem)] p-2 sm:max-w-3xl">
          <DialogTitle className="sr-only">{alt}</DialogTitle>
          {open ? <img src={mediaUrl(open, "display")} alt={alt} className="max-h-[80dvh] w-full rounded-lg object-contain" /> : null}
        </DialogContent>
      </Dialog>
    </>
  )
}
