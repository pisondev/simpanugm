import { ChevronRight, ImageOff, MapPin, Warehouse } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import type { CSSProperties } from "react"

import { EmptyState } from "@/components/layout/empty-state"
import { CountUp } from "@/components/motion/primitives"

import { mediaUrl } from "@/lib/api/client"
import type { MatchCandidate } from "@/lib/api/types"
import { formatDateTime } from "@/lib/format"

import { TierBadge } from "./tier-badge"

// one found item as its owner sees it in the candidate list
export function CandidateCard({ candidate, href }: { candidate: MatchCandidate; href: string }) {
  const t = useTranslations("matching")
  const tc = useTranslations("common")
  const f = candidate.found_report
  const photo = f.photos[0]
  return (
    <Link
      href={href}
      className="surface motion-press motion-lift flex items-stretch gap-3 p-3 focus-visible:ring-[3px] focus-visible:ring-ring/50 focus-visible:outline-none sm:gap-4 sm:p-4"
    >
      <div className="flex size-20 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-muted text-muted-foreground sm:size-24">
        {photo ? (
          <img
            src={mediaUrl(photo.media_id, photo.variant === "thumb_blur" ? "thumb_blur" : "thumb")}
            alt={tc("photoAlt", { kind: tc("photoKindFound"), name: f.item_name })}
            className="size-full object-cover"
            loading="lazy"
          />
        ) : (
          <ImageOff className="size-6 text-muted-foreground" aria-hidden />
        )}
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex flex-wrap items-center gap-2">
          <TierBadge tier={candidate.tier} />
          <CountUp value={candidate.overall * 100} suffix="%" className="text-sm font-semibold" />
        </div>
        <p className="truncate font-medium">{f.item_name}</p>
        <p className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <MapPin className="size-3.5 shrink-0" aria-hidden />
          <span className="truncate">
            {t("foundAt", { when: formatDateTime(f.found_at) })} {t("foundPlace", { place: f.found_location.name })}
          </span>
        </p>
        <p className="flex items-center gap-1.5 text-sm text-muted-foreground">
          <Warehouse className="size-3.5 shrink-0" aria-hidden />
          <span className="truncate">{f.holding_post ? t("holdingPost", { post: f.holding_post.name }) : t("withFinder")}</span>
        </p>
      </div>
      <ChevronRight className="size-5 shrink-0 self-center text-muted-foreground" aria-hidden />
    </Link>
  )
}

export function CandidateList({ items, hrefFor }: { items: MatchCandidate[]; hrefFor: (c: MatchCandidate) => string }) {
  const t = useTranslations("matching")
  return (
    <section id="kandidat" aria-labelledby="kandidat-judul" className="flex scroll-mt-20 flex-col gap-3">
      <h2 id="kandidat-judul" className="font-semibold">
        {t("sectionTitle")}
      </h2>
      {items.length === 0 ? (
        <EmptyState icon={ImageOff}>{t("empty")}</EmptyState>
      ) : (
        <ol className="motion-stagger flex flex-col gap-3">
          {items.map((c, index) => (
            <li key={c.id} style={{ "--motion-index": index } as CSSProperties}>
              <CandidateCard candidate={c} href={hrefFor(c)} />
            </li>
          ))}
        </ol>
      )}
    </section>
  )
}
