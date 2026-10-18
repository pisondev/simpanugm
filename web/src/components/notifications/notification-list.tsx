"use client"

import { useInfiniteQuery } from "@tanstack/react-query"
import { BellOff } from "lucide-react"
import { useTranslations } from "next-intl"
import { useState } from "react"

import { EmptyState } from "@/components/layout/empty-state"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"
import { cn } from "@/lib/utils"

import { useMarkAllRead, useMarkRead } from "./notification-bell"
import { NotificationItem } from "./notification-item"
import { notificationKeys } from "./use-notification-stream"

export function NotificationList() {
  const t = useTranslations("notifications")
  const tc = useTranslations("common")
  const [unreadOnly, setUnreadOnly] = useState(false)
  const query = useInfiniteQuery({
    queryKey: notificationKeys.list(unreadOnly),
    queryFn: ({ pageParam }) => call(api.GET("/notifications", { params: { query: { unread: unreadOnly, limit: 20, cursor: pageParam } } })),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.next_cursor ?? undefined,
  })
  const markRead = useMarkRead()
  const markAll = useMarkAllRead()
  const items = query.data?.pages.flatMap((p) => p.items) ?? []
  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="inline-flex rounded-lg bg-muted p-1" role="group" aria-label={t("title")}>
          {[false, true].map((u) => (
            <button
              key={String(u)}
              type="button"
              aria-pressed={unreadOnly === u}
              onClick={() => setUnreadOnly(u)}
              className={cn(
                "inline-flex min-h-10 items-center rounded-md px-4 text-sm font-medium text-muted-foreground",
                unreadOnly === u && "bg-card text-foreground shadow-sm",
              )}
            >
              {u ? t("filterUnread") : t("filterAll")}
            </button>
          ))}
        </div>
        <Button variant="outline" onClick={() => markAll.mutate()} disabled={markAll.isPending}>
          {t("markAll")}
        </Button>
      </div>
      {query.isPending ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-20 w-full rounded-xl" />
          <Skeleton className="h-20 w-full rounded-xl" />
        </div>
      ) : items.length === 0 ? (
        <EmptyState icon={BellOff}>{unreadOnly ? t("emptyUnread") : t("empty")}</EmptyState>
      ) : (
        <ul className="flex flex-col gap-1 rounded-xl border bg-card p-1">
          {items.map((n) => (
            <li key={n.id}>
              <NotificationItem n={n} onOpen={(x) => !x.read_at && markRead.mutate(x.id)} />
            </li>
          ))}
        </ul>
      )}
      {query.hasNextPage ? (
        <Button variant="outline" className="self-center" onClick={() => void query.fetchNextPage()} disabled={query.isFetchingNextPage}>
          {tc("loadMore")}
        </Button>
      ) : null}
    </div>
  )
}
