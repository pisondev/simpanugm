"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Laptop, Smartphone } from "lucide-react"
import { useTranslations } from "next-intl"
import { toast } from "sonner"

import { FormAlert } from "@/components/auth/form-alert"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import { api } from "@/lib/api/client"
import { call, messageFor } from "@/lib/api/problem"
import { formatDateTime, formatRelative } from "@/lib/format"

// US-05 criterion 2: every session except the current one can be revoked from the list
export function SessionList() {
  const t = useTranslations("profile")
  const te = useTranslations("errors")
  const client = useQueryClient()
  const sessions = useQuery({ queryKey: ["my-sessions"], queryFn: () => call(api.GET("/me/sessions")) })
  const revoke = useMutation({
    mutationFn: (id: string) => call(api.DELETE("/me/sessions/{id}", { params: { path: { id } } })),
    onSuccess: () => toast.success(t("revoked")),
    onSettled: () => client.invalidateQueries({ queryKey: ["my-sessions"] }),
  })
  if (sessions.isPending) {
    return <Skeleton className="h-32 w-full rounded-xl" />
  }
  if (sessions.error) {
    return <FormAlert>{messageFor(sessions.error, te)}</FormAlert>
  }
  const items = sessions.data.items
  return (
    <div className="flex flex-col gap-3">
      <FormAlert>{revoke.error ? messageFor(revoke.error, te) : null}</FormAlert>
      <ul className="flex flex-col divide-y rounded-xl border bg-card">
        {items.map((s) => {
          const Icon = /Android|iOS/.test(s.device) ? Smartphone : Laptop
          return (
            <li key={s.id} className="flex flex-wrap items-center gap-3 p-4">
              <Icon className="size-6 shrink-0 text-muted-foreground" aria-hidden />
              <div className="min-w-0 flex-1">
                <p className="font-medium">
                  {s.device}
                  {s.current ? <span className="ml-2 rounded-full bg-primary-soft px-2 py-0.5 text-xs text-primary">{t("thisDevice")}</span> : null}
                </p>
                <p className="text-sm text-muted-foreground">
                  {t("lastSeen", { when: formatRelative(s.last_seen_at) })} | {t("signedInAt", { when: formatDateTime(s.created_at) })}
                  {s.ip_hint ? ` | ${t("network", { ip: s.ip_hint })}` : ""}
                </p>
              </div>
              {s.current ? null : (
                <Button variant="outline" size="sm" onClick={() => revoke.mutate(s.id)} disabled={revoke.isPending}>
                  {t("revoke")}
                </Button>
              )}
            </li>
          )
        })}
      </ul>
      {items.length <= 1 ? <p className="text-sm text-muted-foreground">{t("noOtherSessions")}</p> : null}
    </div>
  )
}
