"use client"

import { HandHeart, Plus, SearchX } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import type { ReactElement } from "react"

import { DropdownMenu, DropdownMenuContent, DropdownMenuLinkItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"

// the Lapor button of information-architecture 1, opening "Saya kehilangan" or "Saya menemukan"
export function ReportMenu({ trigger, side = "bottom" }: { trigger: ReactElement; side?: "top" | "bottom" }) {
  const t = useTranslations("nav")
  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={trigger} aria-label={t("report")}>
        <Plus aria-hidden />
        <span className={side === "top" ? "sr-only" : undefined}>{t("report")}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent side={side} align="center" className="w-72">
        <DropdownMenuLinkItem render={<Link href="/lapor/hilang" />} className="items-start">
          <SearchX className="mt-0.5 size-5 text-primary" aria-hidden />
          <span>
            <span className="block font-medium">{t("reportLost")}</span>
            <span className="block text-sm text-muted-foreground">{t("reportLostHint")}</span>
          </span>
        </DropdownMenuLinkItem>
        <DropdownMenuLinkItem render={<Link href="/lapor/temuan" />} className="items-start">
          <HandHeart className="mt-0.5 size-5 text-primary" aria-hidden />
          <span>
            <span className="block font-medium">{t("reportFound")}</span>
            <span className="block text-sm text-muted-foreground">{t("reportFoundHint")}</span>
          </span>
        </DropdownMenuLinkItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
