"use client"

import Link from "next/link"

import type { Notification } from "@/lib/api/types"
import { formatRelative } from "@/lib/format"
import { cn } from "@/lib/utils"

export function NotificationItem({ n, onOpen, compact = false }: { n: Notification; onOpen: (n: Notification) => void; compact?: boolean }) {
  const unread = !n.read_at
  return (
    <Link
      href={n.link}
      onClick={() => onOpen(n)}
      className={cn(
        "flex gap-3 rounded-lg px-3 py-3 transition-colors hover:bg-muted focus-visible:ring-3 focus-visible:ring-ring/50",
        unread && "bg-primary-soft/40",
      )}
    >
      <span className={cn("mt-1.5 size-2 shrink-0 rounded-full", unread ? "bg-primary" : "bg-transparent")} aria-hidden />
      <span className="flex min-w-0 flex-col gap-0.5">
        <span className={cn("text-sm", unread ? "font-semibold" : "font-medium")}>{n.title}</span>
        <span className={cn("text-sm text-muted-foreground", compact && "line-clamp-2")}>{n.body}</span>
        <time dateTime={n.created_at} className="text-xs text-muted-foreground">
          {formatRelative(n.created_at)}
        </time>
      </span>
    </Link>
  )
}
