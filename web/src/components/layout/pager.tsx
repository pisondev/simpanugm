import { useTranslations } from "next-intl"

import { ButtonLink } from "@/components/ui/button-link"
import { queryString } from "@/lib/query"

// keyset pages cannot go back one step, so the pager offers the first page and the next one
export function Pager({ path, filter, cursor, next }: { path: string; filter: Record<string, string | undefined>; cursor?: string; next?: string | null }) {
  const t = useTranslations("common")
  if (!cursor && !next) {
    return null
  }
  return (
    <div className="mt-6 flex flex-wrap justify-center gap-3">
      {cursor ? (
        <ButtonLink href={`${path}${queryString(filter)}`} variant="ghost">
          {t("firstPage")}
        </ButtonLink>
      ) : null}
      {next ? (
        <ButtonLink href={`${path}${queryString(filter, { cursor: next })}`} variant="outline">
          {t("nextPage")}
        </ButtonLink>
      ) : null}
    </div>
  )
}
