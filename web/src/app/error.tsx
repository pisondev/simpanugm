"use client"

import { TriangleAlert } from "lucide-react"
import { useTranslations } from "next-intl"

import { Button } from "@/components/ui/button"

export default function ErrorPage({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  const t = useTranslations()
  return (
    <main id="konten" className="mx-auto flex min-h-[60dvh] max-w-md flex-col items-center justify-center gap-4 px-4 text-center">
      <TriangleAlert className="size-10 text-warning" aria-hidden />
      <h1 className="text-xl font-semibold">{t("errors.pageTitle")}</h1>
      <p className="text-muted-foreground">{t("errors.pageBody")}</p>
      <Button onClick={reset}>{t("common.retry")}</Button>
      {error.digest ? <p className="text-xs text-muted-foreground">{t("common.requestId", { id: error.digest })}</p> : null}
    </main>
  )
}
