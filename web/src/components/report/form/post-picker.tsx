"use client"

import { useTranslations } from "next-intl"

import type { Post } from "@/lib/api/types"
import { formatDistance } from "@/lib/format"
import { cn } from "@/lib/utils"

import { OpenNow, TodayHours } from "../post-hours"

type Props = { name: string; posts: Post[]; value: string; onChange: (id: string) => void; describedBy?: string; showNearest?: boolean }

// information-architecture 3.2: posts sorted by distance with opening hours and the open now state
export function PostPicker({ name, posts, value, onChange, describedBy, showNearest = true }: Props) {
  const t = useTranslations("form.custody")
  return (
    <div aria-describedby={describedBy} className="grid gap-2">
      {posts.map((post, i) => {
        const checked = post.id === value
        return (
          <label
            key={post.id}
            className={cn(
              "flex cursor-pointer items-start gap-3 rounded-xl border bg-card p-3 transition-colors hover:border-primary/40 has-[:focus-visible]:ring-3 has-[:focus-visible]:ring-ring/50",
              checked && "border-primary bg-primary-soft/40",
            )}
          >
            <input type="radio" name={name} value={post.id} checked={checked} onChange={() => onChange(post.id)} className="mt-1 size-4 accent-primary" />
            <span className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="flex flex-wrap items-center gap-2 font-medium">
                {post.name}
                {showNearest && i === 0 && post.distance_m != null ? (
                  <span className="rounded-full bg-primary px-2 py-0.5 text-xs text-primary-foreground">{t("nearest")}</span>
                ) : null}
              </span>
              <span className="text-sm text-muted-foreground">
                {post.location_name} | {post.zone_name}
              </span>
              {post.distance_m != null ? (
                <span className="text-sm text-muted-foreground">{t("distance", { distance: formatDistance(post.distance_m) })}</span>
              ) : null}
              <span className="flex flex-wrap items-center gap-x-3 gap-y-1">
                <OpenNow hours={post.opening_hours} />
                <TodayHours hours={post.opening_hours} />
              </span>
            </span>
          </label>
        )
      })}
    </div>
  )
}
