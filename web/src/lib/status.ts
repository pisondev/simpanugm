import {
  Archive,
  ArchiveX,
  Ban,
  CircleCheck,
  CircleSlash,
  FileText,
  Hand,
  Handshake,
  Hourglass,
  Inbox,
  type LucideIcon,
  MessageCircleQuestion,
  Pause,
  ShieldCheck,
  ShieldQuestion,
  Sparkles,
  Undo2,
  Warehouse,
  XCircle,
} from "lucide-react"

import type { ClaimStatus, CustodyStatus, FoundStatus, LostStatus } from "./api/types"

export type Tone = "neutral" | "info" | "warning" | "primary" | "success" | "danger"

export type StatusMeta = { icon: LucideIcon; tone: Tone }

// ui-guidelines section 4, labels live in messages/id.json under status
export const lostStatus = {
  reported: { icon: FileText, tone: "neutral" },
  potential_match: { icon: Sparkles, tone: "info" },
  under_verification: { icon: ShieldQuestion, tone: "warning" },
  claimed: { icon: ShieldCheck, tone: "primary" },
  returned: { icon: CircleCheck, tone: "success" },
  closed: { icon: CircleSlash, tone: "neutral" },
} satisfies Record<LostStatus, StatusMeta>

export const foundStatus = {
  reported: { icon: FileText, tone: "neutral" },
  potential_match: { icon: Sparkles, tone: "info" },
  under_verification: { icon: ShieldQuestion, tone: "warning" },
  claimed: { icon: ShieldCheck, tone: "primary" },
  returned: { icon: CircleCheck, tone: "success" },
  unclaimed: { icon: Archive, tone: "warning" },
  closed: { icon: CircleSlash, tone: "neutral" },
} satisfies Record<FoundStatus, StatusMeta>

export const custodyStatus = {
  with_finder: { icon: Hand, tone: "warning" },
  in_custody: { icon: Warehouse, tone: "primary" },
  released: { icon: Handshake, tone: "success" },
  disposed: { icon: ArchiveX, tone: "neutral" },
  cancelled: { icon: Ban, tone: "neutral" },
} satisfies Record<CustodyStatus, StatusMeta>

export const claimStatus = {
  submitted: { icon: Inbox, tone: "info" },
  under_review: { icon: ShieldQuestion, tone: "warning" },
  info_requested: { icon: MessageCircleQuestion, tone: "warning" },
  on_hold: { icon: Pause, tone: "neutral" },
  approved: { icon: ShieldCheck, tone: "primary" },
  handed_over: { icon: CircleCheck, tone: "success" },
  rejected: { icon: XCircle, tone: "danger" },
  withdrawn: { icon: Undo2, tone: "neutral" },
  expired: { icon: Hourglass, tone: "neutral" },
} satisfies Record<ClaimStatus, StatusMeta>

// claims that still need a decision or a handover
export const claimOpen: readonly ClaimStatus[] = ["submitted", "under_review", "info_requested", "on_hold", "approved"]

// tinted badges use the darker text tokens, the base tones fall just under 4.5 to 1 on their own tint
export const toneClass: Record<Tone, string> = {
  neutral: "bg-secondary text-secondary-foreground border-border",
  info: "bg-primary-soft text-info border-info/30",
  warning: "bg-warning/10 text-warning-text border-warning/30",
  primary: "bg-primary-soft text-primary border-primary/30",
  success: "bg-success/10 text-success-text border-success/30",
  danger: "bg-destructive/10 text-destructive border-destructive/30",
}

// the edit rule of FR-LOST-09 and FR-FOUND-08, the api decides and the ui only hides buttons early
export const lostOpen: readonly LostStatus[] = ["reported", "potential_match", "under_verification", "claimed"]
export const foundOpen: readonly FoundStatus[] = ["reported", "potential_match", "under_verification", "claimed", "unclaimed"]
