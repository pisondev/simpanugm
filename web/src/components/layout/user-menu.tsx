"use client"

import { Briefcase, LogOut, UserRound } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"

import { useSignOut } from "@/components/auth/use-sign-out"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuLinkItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import type { Me } from "@/lib/api/types"

function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() ?? "")
    .join("")
}

export function UserMenu({ me }: { me: Pick<Me, "full_name" | "email" | "role"> }) {
  const t = useTranslations("nav")
  const signOut = useSignOut()
  const staff = me.role !== "member"
  return (
    <DropdownMenu>
      {/* the initials are decoration, the accessible name is the full name so it never mismatches what is seen */}
      <DropdownMenuTrigger
        render={<Button variant="ghost" size="icon" className="rounded-full bg-primary-soft font-semibold text-primary hover:bg-primary-soft" />}
      >
        <span aria-hidden>{initials(me.full_name)}</span>
        <span className="sr-only">{t("accountOf", { name: me.full_name })}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-64">
        <DropdownMenuGroup>
          <DropdownMenuLabel className="text-sm">
            <span className="block truncate font-semibold text-foreground">{me.full_name}</span>
            <span className="block truncate font-normal">{me.email}</span>
          </DropdownMenuLabel>
        </DropdownMenuGroup>
        <DropdownMenuSeparator />
        <DropdownMenuLinkItem render={<Link href="/profil" />}>
          <UserRound aria-hidden />
          {t("profile")}
        </DropdownMenuLinkItem>
        {staff ? (
          <DropdownMenuLinkItem render={<Link href={me.role === "admin" ? "/admin" : "/petugas"} />}>
            <Briefcase aria-hidden />
            {t("workspace")}
          </DropdownMenuLinkItem>
        ) : null}
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => signOut.mutate(false)} disabled={signOut.isPending}>
          <LogOut aria-hidden />
          {t("signOut")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
