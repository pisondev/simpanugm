import { SearchX } from "lucide-react"
import { useTranslations } from "next-intl"

import { ButtonLink } from "@/components/ui/button-link"

export default function NotFound() {
  const t = useTranslations("errors")
  return (
    <main id="konten" className="mx-auto flex min-h-dvh max-w-md flex-col items-center justify-center gap-4 px-4 text-center">
      <SearchX className="size-10 text-muted-foreground" aria-hidden />
      <h1 className="text-xl font-semibold">{t("notFoundTitle")}</h1>
      <p className="text-muted-foreground">{t("notFoundBody")}</p>
      <ButtonLink href="/">{t("toHome")}</ButtonLink>
    </main>
  )
}
