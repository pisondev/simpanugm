import { PackageCheck } from "lucide-react"
import Link from "next/link"

import { cn } from "@/lib/utils"

export function Wordmark({ href = "/", className }: { href?: string; className?: string }) {
  return (
    <Link href={href} className={cn("inline-flex items-center gap-2 text-lg font-semibold tracking-tight text-foreground", className)}>
      <PackageCheck className="size-6 text-primary" aria-hidden />
      <span>
        Simpan<span className="text-primary">UGM</span>
      </span>
    </Link>
  )
}
