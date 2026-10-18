"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Bell, BellOff } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Skeleton } from "@/components/ui/skeleton"
import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"
import type { Notification } from "@/lib/api/types"

import { NotificationItem } from "./notification-item"
import { notificationKeys, useNotificationStream } from "./use-notification-stream"

export function useMarkRead() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => call(api.POST("/notifications/{id}/read", { params: { path: { id } } })),
    onSettled: () => client.invalidateQueries({ queryKey: notificationKeys.all }),
  })
}

export function useMarkAllRead() {
  const client = useQueryClient()
  return useMutation({
    mutationFn: () => call(api.POST("/notifications/read-all")),
    onSettled: () => client.invalidateQueries({ queryKey: notificationKeys.all }),
  })
}

export function useUnreadCount() {
  return useQuery({
    queryKey: notificationKeys.unread,
    queryFn: () => call(api.GET("/notifications/unread-count")),
    // the stream refreshes it at once, polling only covers a stream that could not connect
    refetchInterval: 120_000,
  })
}

export function NotificationBell() {
  const t = useTranslations("notifications")
  useNotificationStream()
  const [open, setOpen] = useState(false)
  const unread = useUnreadCount()
  const latest = useQuery({
    queryKey: [...notificationKeys.list(false), "latest"],
    queryFn: () => call(api.GET("/notifications", { params: { query: { limit: 8 } } })),
    enabled: open,
  })
  const markRead = useMarkRead()
  const markAll = useMarkAllRead()
  const count = unread.data?.count ?? 0
  const items = latest.data?.items ?? []
  function opened(n: Notification) {
    setOpen(false)
    if (!n.read_at) {
      markRead.mutate(n.id)
    }
  }
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger render={<Button variant="ghost" size="icon" className="relative" />} aria-label={t("bellUnread", { count })}>
        <Bell aria-hidden />
        {count > 0 ? (
          <span
            className="absolute top-1.5 right-1.5 flex min-w-5 items-center justify-center rounded-full bg-destructive px-1 text-[0.7rem] leading-5 font-semibold text-white"
            aria-hidden
          >
            {count > 99 ? "99+" : count}
          </span>
        ) : null}
      </PopoverTrigger>
      <PopoverContent align="end" className="w-[min(24rem,calc(100vw-2rem))] gap-0 p-0">
        <div className="flex items-center justify-between border-b px-3 py-2">
          <span className="font-semibold">{t("title")}</span>
          <Button variant="link" size="sm" onClick={() => markAll.mutate()} disabled={count === 0 || markAll.isPending}>
            {t("markAll")}
          </Button>
        </div>
        <div className="max-h-96 overflow-y-auto p-1">
          {latest.isPending ? (
            <div className="flex flex-col gap-2 p-2">
              <Skeleton className="h-14 w-full" />
              <Skeleton className="h-14 w-full" />
            </div>
          ) : items.length === 0 ? (
            <p className="flex flex-col items-center gap-2 px-4 py-8 text-center text-sm text-muted-foreground">
              <BellOff className="size-6" aria-hidden />
              {t("empty")}
            </p>
          ) : (
            items.map((n) => <NotificationItem key={n.id} n={n} onOpen={opened} compact />)
          )}
        </div>
        <Link
          href="/notifikasi"
          onClick={() => setOpen(false)}
          className="block border-t px-3 py-3 text-center text-sm font-medium text-primary hover:bg-muted"
        >
          {t("seeAll")}
        </Link>
      </PopoverContent>
    </Popover>
  )
}
