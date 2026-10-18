"use client"

import { useQueryClient } from "@tanstack/react-query"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useEffect } from "react"
import { toast } from "sonner"

import type { Notification } from "@/lib/api/types"

export const notificationKeys = {
  all: ["notifications"] as const,
  unread: ["notifications", "unread"] as const,
  list: (unread: boolean) => ["notifications", "list", unread] as const,
}

// US-35: one EventSource per page, the browser reconnects with Last-Event-ID so nothing sent meanwhile is lost
export function useNotificationStream() {
  const client = useQueryClient()
  const router = useRouter()
  const t = useTranslations("notifications")
  useEffect(() => {
    if (typeof EventSource === "undefined") {
      return
    }
    const source = new EventSource("/api/v1/notifications/stream")
    const onNotification = (event: MessageEvent<string>) => {
      void client.invalidateQueries({ queryKey: notificationKeys.all })
      try {
        const n = JSON.parse(event.data) as Notification
        toast(n.title, { description: n.body, action: { label: t("open"), onClick: () => router.push(n.link) } })
      } catch {
        // a malformed event only refreshes the counts
      }
    }
    source.addEventListener("notification", onNotification)
    return () => {
      source.removeEventListener("notification", onNotification)
      source.close()
    }
  }, [client, router, t])
}
