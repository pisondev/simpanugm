"use client"

import { Check, Copy } from "lucide-react"
import { useTranslations } from "next-intl"
import { useState } from "react"

import { Button } from "@/components/ui/button"

export function CodeChip({ code }: { code: string }) {
  const t = useTranslations("common")
  const [copied, setCopied] = useState(false)
  async function copy() {
    try {
      await navigator.clipboard.writeText(code)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 2000)
    } catch {
      setCopied(false)
    }
  }
  return (
    <span className="inline-flex items-center gap-1 rounded-lg border bg-card py-0.5 pr-0.5 pl-2.5">
      <code className="font-mono text-sm font-semibold tracking-wide">{code}</code>
      <Button variant="ghost" size="icon-sm" onClick={copy} aria-label={copied ? t("copied") : t("copy")}>
        {copied ? <Check className="text-success" aria-hidden /> : <Copy aria-hidden />}
      </Button>
      <span aria-live="polite" className="sr-only">
        {copied ? t("copied") : ""}
      </span>
    </span>
  )
}
