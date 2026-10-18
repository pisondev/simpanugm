"use client"

import { useMutation, useQuery } from "@tanstack/react-query"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import { FormAlert } from "@/components/auth/form-alert"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Skeleton } from "@/components/ui/skeleton"
import { api } from "@/lib/api/client"
import { call, messageFor } from "@/lib/api/problem"
import type { NotificationPreference } from "@/lib/api/types"

// FR-NOTIF-04, account and security messages are mandatory and therefore not listed
export function PreferencesForm() {
  const te = useTranslations("errors")
  const current = useQuery({ queryKey: ["notification-preferences"], queryFn: () => call(api.GET("/me/notification-preferences")) })
  if (current.isPending) {
    return <Skeleton className="h-48 w-full rounded-xl" />
  }
  if (current.error || !current.data) {
    return <FormAlert>{messageFor(current.error, te)}</FormAlert>
  }
  return <PreferencesFields initial={current.data.items} />
}

function PreferencesFields({ initial }: { initial: NotificationPreference[] }) {
  const t = useTranslations("notifications")
  const te = useTranslations("errors")
  const tc = useTranslations("common")
  const [items, setItems] = useState(initial)
  const save = useMutation({
    mutationFn: () => call(api.PUT("/me/notification-preferences", { body: { items } })),
    onSuccess: (data) => {
      setItems(data.items)
      toast.success(t("prefsSaved"))
    },
  })
  function toggle(group: NotificationPreference["group"], channel: "app" | "email", value: boolean) {
    setItems((prev) => prev.map((p) => (p.group === group ? { ...p, [channel]: value } : p)))
  }
  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={(e) => {
        e.preventDefault()
        save.mutate()
      }}
    >
      <FormAlert>{save.error ? messageFor(save.error, te) : null}</FormAlert>
      <div className="flex flex-col divide-y rounded-xl border bg-card">
        {items.map((p) => (
          <fieldset key={p.group} className="flex flex-col gap-3 p-4 sm:flex-row sm:items-start sm:justify-between">
            <legend className="sr-only">{t(`groups.${p.group}`)}</legend>
            <div className="max-w-md">
              <p className="font-medium">{t(`groups.${p.group}`)}</p>
              <p className="text-sm text-muted-foreground">{t(`groupHints.${p.group}`)}</p>
            </div>
            <div className="flex gap-6">
              {(["app", "email"] as const).map((channel) => (
                <label key={channel} className="flex min-h-11 items-center gap-2 text-sm">
                  <Checkbox checked={p[channel]} onCheckedChange={(v) => toggle(p.group, channel, Boolean(v))} />
                  {channel === "app" ? t("channelApp") : t("channelEmail")}
                </label>
              ))}
            </div>
          </fieldset>
        ))}
      </div>
      <Button type="submit" className="self-start" disabled={save.isPending}>
        {save.isPending ? tc("saving") : tc("save")}
      </Button>
    </form>
  )
}
