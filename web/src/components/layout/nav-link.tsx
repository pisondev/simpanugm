"use client"

import type { LucideIcon } from "lucide-react"
import Link from "next/link"
import { usePathname } from "next/navigation"

import { cn } from "@/lib/utils"

export type NavItem = { href: string; label: string; icon: LucideIcon; exact?: boolean }

export function useActive() {
  const path = usePathname()
  return (item: Pick<NavItem, "href" | "exact">) => (item.exact ? path === item.href : path === item.href || path.startsWith(`${item.href}/`))
}

export function SideLink({ item }: { item: NavItem }) {
  const active = useActive()(item)
  const Icon = item.icon
  return (
    <Link
      href={item.href}
      aria-current={active ? "page" : undefined}
      className={cn(
        "flex min-h-11 items-center gap-3 rounded-lg px-3 text-sm font-medium text-muted-foreground transition-colors hover:bg-muted hover:text-foreground",
        active && "bg-primary-soft text-primary hover:bg-primary-soft hover:text-primary",
      )}
    >
      <Icon className="size-5 shrink-0" aria-hidden />
      {item.label}
    </Link>
  )
}

export function TopLink({ item }: { item: NavItem }) {
  const active = useActive()(item)
  return (
    <Link
      href={item.href}
      aria-current={active ? "page" : undefined}
      className={cn(
        "inline-flex min-h-11 items-center rounded-lg px-3 text-sm font-medium text-muted-foreground transition-colors hover:text-foreground",
        active && "text-primary hover:text-primary",
      )}
    >
      {item.label}
    </Link>
  )
}

export function BottomLink({ item }: { item: NavItem }) {
  const active = useActive()(item)
  const Icon = item.icon
  return (
    <Link
      href={item.href}
      aria-current={active ? "page" : undefined}
      className={cn("flex min-h-14 flex-1 flex-col items-center justify-center gap-0.5 text-xs font-medium text-muted-foreground", active && "text-primary")}
    >
      <Icon className="size-5" aria-hidden />
      {item.label}
    </Link>
  )
}
