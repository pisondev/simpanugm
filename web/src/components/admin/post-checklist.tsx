"use client"

import { Checkbox } from "@/components/ui/checkbox"
import type { Post } from "@/lib/api/types"

export function PostChecklist({
  posts,
  value,
  onChange,
  describedBy,
}: {
  posts: Post[]
  value: string[]
  onChange: (v: string[]) => void
  describedBy?: string
}) {
  return (
    <div className="grid max-h-64 gap-1 overflow-y-auto rounded-lg border p-2" aria-describedby={describedBy}>
      {posts.map((p) => {
        const on = value.includes(p.id)
        return (
          <label key={p.id} className="flex min-h-11 cursor-pointer items-center gap-3 rounded-md px-2 hover:bg-muted">
            <Checkbox checked={on} onCheckedChange={(v) => onChange(v ? [...value, p.id] : value.filter((x) => x !== p.id))} />
            <span className="flex flex-col">
              <span className="text-sm font-medium">{p.name}</span>
              <span className="text-xs text-muted-foreground">{p.zone_name}</span>
            </span>
          </label>
        )
      })}
    </div>
  )
}
