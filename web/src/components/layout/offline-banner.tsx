"use client"

import { WifiOff } from "lucide-react"
import { useTranslations } from "next-intl"
import { useSyncExternalStore } from "react"

function subscribe(onChange: () => void) {
  window.addEventListener("online", onChange)
  window.addEventListener("offline", onChange)
  return () => {
    window.removeEventListener("online", onChange)
    window.removeEventListener("offline", onChange)
  }
}

export function OfflineBanner() {
  const t = useTranslations("errors")
  const online = useSyncExternalStore(
    subscribe,
    () => navigator.onLine,
    () => true,
  )
  if (online) {
    return null
  }
  return (
    <div role="status" className="flex items-center justify-center gap-2 bg-warning px-4 py-2 text-sm text-white">
      <WifiOff className="size-4" aria-hidden />
      {t("offline")}
    </div>
  )
}
