import { useTranslations } from "next-intl"

import type { ClaimStatus, CustodyStatus, FoundStatus, LostStatus } from "@/lib/api/types"
import { claimStatus, custodyStatus, foundStatus, lostStatus, toneClass } from "@/lib/status"
import { cn } from "@/lib/utils"

type Props =
  | { kind: "lost"; status: LostStatus; className?: string }
  | { kind: "found"; status: FoundStatus; className?: string }
  | { kind: "custody"; status: CustodyStatus; className?: string }
  | { kind: "claim"; status: ClaimStatus; className?: string }

// label, icon and colour together, colour is only the third cue (NFR-USE-05)
export function StatusBadge(props: Props) {
  const t = useTranslations("status")
  let meta
  let label: string
  switch (props.kind) {
    case "lost":
      meta = lostStatus[props.status]
      label = t(`lost.${props.status}`)
      break
    case "found":
      meta = foundStatus[props.status]
      label = t(`found.${props.status}`)
      break
    case "claim":
      meta = claimStatus[props.status]
      label = t(`claim.${props.status}`)
      break
    default:
      meta = custodyStatus[props.status]
      label = t(`custody.${props.status}`)
  }
  const Icon = meta.icon
  return (
    <span
      className={cn(
        "inline-flex w-fit shrink-0 items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium whitespace-nowrap",
        toneClass[meta.tone],
        props.className,
      )}
    >
      <Icon className="size-3.5" aria-hidden />
      {label}
    </span>
  )
}
