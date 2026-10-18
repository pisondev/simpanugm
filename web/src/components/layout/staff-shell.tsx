"use client"

import { Database, Ellipsis, FileSearch, FolderOpen, HandHeart, House, LayoutDashboard, PackageOpen, Search, UserRound, Users, Warehouse } from "lucide-react"
import { useTranslations } from "next-intl"
import type { ReactNode } from "react"

import { NotificationBell } from "@/components/notifications/notification-bell"
import { Button } from "@/components/ui/button"
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet"
import type { Me } from "@/lib/api/types"

import { BottomLink, type NavItem, SideLink } from "./nav-link"
import { OfflineBanner } from "./offline-banner"
import { UserMenu } from "./user-menu"
import { Wordmark } from "./wordmark"

function Section({ title, items }: { title: string; items: NavItem[] }) {
  return (
    <div className="flex flex-col gap-1">
      <p className="px-3 pt-4 pb-1 text-xs font-semibold tracking-wide text-muted-foreground uppercase">{title}</p>
      {items.map((item) => (
        <SideLink key={item.href} item={item} />
      ))}
    </div>
  )
}

export function StaffShell({ me, children }: { me: Pick<Me, "full_name" | "email" | "role">; children: ReactNode }) {
  const t = useTranslations("nav")
  const home: NavItem = { href: "/petugas", label: t("officerHome"), icon: House, exact: true }
  const receive: NavItem = { href: "/petugas/terima", label: t("receive"), icon: PackageOpen }
  const inventory: NavItem = { href: "/petugas/inventaris", label: t("inventory"), icon: Warehouse }
  const search: NavItem = { href: "/petugas/cari", label: t("search"), icon: Search }
  const work: NavItem[] = [
    home,
    receive,
    inventory,
    search,
    { href: "/petugas/lapor-hilang", label: t("proxyLost"), icon: FileSearch },
    { href: "/petugas/lapor-temuan", label: t("proxyFound"), icon: HandHeart },
  ]
  const admin: NavItem[] =
    me.role === "admin"
      ? [
          { href: "/admin", label: t("adminHome"), icon: LayoutDashboard, exact: true },
          { href: "/admin/pengguna", label: t("users"), icon: Users },
          { href: "/admin/master", label: t("masterData"), icon: Database },
        ]
      : []
  const account: NavItem[] = [
    { href: "/laporan", label: t("personal"), icon: FolderOpen },
    { href: "/profil", label: t("profile"), icon: UserRound },
  ]
  const sections = (
    <>
      <Section title={t("workspace")} items={work} />
      {admin.length > 0 ? <Section title={t("adminSection")} items={admin} /> : null}
      <Section title={t("account")} items={account} />
    </>
  )
  return (
    <div className="flex min-h-dvh flex-col lg:flex-row">
      <aside className="sticky top-0 hidden h-dvh w-64 shrink-0 flex-col border-r bg-card px-3 py-4 lg:flex">
        <Wordmark href={me.role === "admin" ? "/admin" : "/petugas"} className="px-3" />
        <nav aria-label={t("main")} className="mt-2 flex-1 overflow-y-auto">
          {sections}
        </nav>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <OfflineBanner />
        <header className="sticky top-0 z-40 border-b bg-card/95 backdrop-blur">
          <div className="flex h-16 items-center justify-between gap-4 px-4 md:px-6">
            <Wordmark href={me.role === "admin" ? "/admin" : "/petugas"} className="lg:invisible" />
            <div className="flex items-center gap-1">
              <NotificationBell />
              <UserMenu me={me} />
            </div>
          </div>
        </header>
        <main id="konten" className="w-full max-w-7xl flex-1 px-4 pt-6 pb-28 md:px-6 lg:pb-12">
          {children}
        </main>
      </div>
      <nav aria-label={t("main")} className="fixed inset-x-0 bottom-0 z-40 border-t bg-card pb-[env(safe-area-inset-bottom)] lg:hidden">
        <div className="flex items-stretch">
          <BottomLink item={home} />
          <BottomLink item={{ ...receive, label: t("receiveShort") }} />
          <BottomLink item={search} />
          <BottomLink item={inventory} />
          <Sheet>
            <SheetTrigger
              render={<Button variant="ghost" className="h-auto min-h-14 flex-1 flex-col gap-0.5 rounded-none text-xs font-medium text-muted-foreground" />}
            >
              <Ellipsis className="size-5" aria-hidden />
              {t("more")}
            </SheetTrigger>
            <SheetContent side="bottom" className="max-h-[80dvh] overflow-y-auto px-3 pb-6">
              <SheetHeader className="px-3">
                <SheetTitle>{t("more")}</SheetTitle>
              </SheetHeader>
              <nav aria-label={t("more")}>{sections}</nav>
            </SheetContent>
          </Sheet>
        </div>
      </nav>
    </div>
  )
}
