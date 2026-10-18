"use client"

import { Bell, FolderOpen, House, PackageSearch, UserRound } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import type { ReactNode } from "react"

import { NotificationBell, useUnreadCount } from "@/components/notifications/notification-bell"
import { Button } from "@/components/ui/button"
import type { Me } from "@/lib/api/types"

import { BottomLink, type NavItem, TopLink, useActive } from "./nav-link"
import { OfflineBanner } from "./offline-banner"
import { ReportMenu } from "./report-menu"
import { UserMenu } from "./user-menu"
import { Wordmark } from "./wordmark"

export function MemberShell({ me, children }: { me: Pick<Me, "full_name" | "email" | "role">; children: ReactNode }) {
  const t = useTranslations("nav")
  const items: NavItem[] = [
    { href: "/beranda", label: t("home"), icon: House },
    { href: "/laporan", label: t("myReports"), icon: FolderOpen },
    { href: "/barang-temuan", label: t("foundItems"), icon: PackageSearch },
    { href: "/profil", label: t("profile"), icon: UserRound },
  ]
  const [home, reports, found, profile] = items as [NavItem, NavItem, NavItem, NavItem]
  return (
    <div className="flex min-h-dvh flex-col">
      <OfflineBanner />
      <header className="sticky top-0 z-40 border-b bg-card/95 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-5xl items-center justify-between gap-4 px-4 md:px-6">
          <Wordmark href="/beranda" />
          <nav aria-label={t("main")} className="hidden items-center gap-1 md:flex">
            <TopLink item={home} />
            <TopLink item={reports} />
            <TopLink item={found} />
            <ReportMenu trigger={<Button className="ml-2" />} />
          </nav>
          <div className="flex items-center gap-1">
            <div className="hidden md:block">
              <NotificationBell />
            </div>
            <UserMenu me={me} />
          </div>
        </div>
      </header>
      <main id="konten" className="mx-auto w-full max-w-5xl flex-1 px-4 pt-6 pb-28 md:px-6 md:pb-12">
        {children}
      </main>
      <nav aria-label={t("main")} className="fixed inset-x-0 bottom-0 z-40 border-t bg-card pb-[env(safe-area-inset-bottom)] md:hidden">
        <div className="flex items-stretch">
          <BottomLink item={home} />
          <BottomLink item={reports} />
          <div className="flex flex-1 items-center justify-center">
            <ReportMenu side="top" trigger={<Button size="icon-lg" className="-mt-6 size-14 rounded-full shadow-lg [&_svg:not([class*='size-'])]:size-6" />} />
          </div>
          <NotificationTab label={t("notifications")} />
          <BottomLink item={profile} />
        </div>
      </nav>
    </div>
  )
}

function NotificationTab({ label }: { label: string }) {
  const tn = useTranslations("notifications")
  const active = useActive()({ href: "/notifikasi" })
  const count = useUnreadCount().data?.count ?? 0
  return (
    <Link
      href="/notifikasi"
      aria-current={active ? "page" : undefined}
      aria-label={tn("bellUnread", { count })}
      className={`relative flex min-h-14 flex-1 flex-col items-center justify-center gap-0.5 text-xs font-medium ${active ? "text-primary" : "text-muted-foreground"}`}
    >
      <Bell className="size-5" aria-hidden />
      <span aria-hidden>{label}</span>
      {count > 0 ? (
        <span
          className="absolute top-1.5 left-1/2 ml-1 flex min-w-5 items-center justify-center rounded-full bg-destructive px-1 text-[0.7rem] leading-5 font-semibold text-white"
          aria-hidden
        >
          {count > 99 ? "99+" : count}
        </span>
      ) : null}
    </Link>
  )
}
