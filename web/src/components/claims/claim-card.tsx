import { ImageOff, KeyRound } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"

import { StatusBadge } from "@/components/report/status-badge"
import { mediaUrl } from "@/lib/api/client"
import type { ClaimSummary } from "@/lib/api/types"
import { formatRelative } from "@/lib/format"

function Thumb({ id, alt }: { id?: string | null; alt: string }) {
  if (!id) {
    return (
      <div className="flex size-16 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground">
        <ImageOff className="size-5" aria-hidden />
      </div>
    )
  }
  return <img src={mediaUrl(id, "thumb")} alt={alt} className="size-16 shrink-0 rounded-lg bg-muted object-cover" loading="lazy" />
}

export function ClaimCard({ claim, href, staff = false }: { claim: ClaimSummary; href: string; staff?: boolean }) {
  const t = useTranslations("claims")
  const tc = useTranslations("common")
  return (
    <li>
      <Link href={href} className="surface motion-press motion-lift flex gap-3 p-3 focus-visible:ring-3 focus-visible:ring-ring/50">
        <Thumb id={claim.thumb_media_id} alt={tc("photoAlt", { kind: tc("photoKindFound"), name: claim.found_item_name })} />
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-xs text-muted-foreground">{claim.code}</span>
            <StatusBadge kind="claim" status={claim.status} />
            {claim.status === "approved" && !staff ? (
              <span className="inline-flex items-center gap-1 rounded-full bg-primary-soft px-2 py-0.5 text-xs font-medium text-primary">
                <KeyRound className="size-3" aria-hidden />
                {t("handoverReady")}
              </span>
            ) : null}
          </div>
          <p className="truncate font-medium">{claim.found_item_name}</p>
          <p className="text-sm text-muted-foreground">
            {t("lostItem")}: <span className="font-mono text-xs">{claim.lost_code}</span> ({claim.lost_item_name})
          </p>
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
            {staff && claim.claimant_name ? <span>{claim.claimant_name}</span> : null}
            {claim.holding_post_name ? <span>{claim.holding_post_name}</span> : null}
            <span>{formatRelative(claim.created_at)}</span>
          </div>
        </div>
      </Link>
    </li>
  )
}
