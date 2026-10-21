"use client"

import { ExternalLink, Megaphone } from "lucide-react"

export function DemoBanner() {
  // Shown in development and when demo mode is enabled
  const isDemo = process.env.NEXT_PUBLIC_DEMO_MODE === "true"
  if (!isDemo) {
    return null
  }

  const mailpitUrl = process.env.NEXT_PUBLIC_MAILPIT_URL || "http://localhost:8025"

  return (
    <aside aria-label="Informasi Mode Demo" className="border-b bg-primary px-4 py-1.5 text-xs text-white">
      <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-1.5 font-medium">
          <Megaphone className="size-3.5 shrink-0 text-amber-300" aria-hidden />
          <span>Mode Demo: Sistem berjalan dengan model AI nyata di laptop.</span>
        </div>
        <div className="flex items-center gap-3">
          <a
            href={mailpitUrl}
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-1 font-medium underline underline-offset-2 hover:text-amber-200 focus-visible:rounded focus-visible:ring-2 focus-visible:ring-white focus-visible:outline-none"
            title="Buka Mailpit untuk melihat email verifikasi dan notifikasi demo"
          >
            <span>Kotak Masuk Email (Mailpit)</span>
            <ExternalLink className="size-3" aria-hidden />
          </a>
        </div>
      </div>
    </aside>
  )
}
