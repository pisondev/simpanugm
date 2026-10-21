import Link from "next/link"
import { getTranslations } from "next-intl/server"

import { DemoBanner } from "@/components/layout/demo-banner"
import { Wordmark } from "@/components/layout/wordmark"
import { ButtonLink } from "@/components/ui/button-link"
import { homeFor } from "@/lib/paths"
import { getMe } from "@/lib/session"

async function optionalMe() {
  try {
    return await getMe()
  } catch {
    return null
  }
}

export default async function PublicLayout({ children }: LayoutProps<"/">) {
  const [t, tl, me] = await Promise.all([getTranslations("nav"), getTranslations("landing"), optionalMe()])
  return (
    <div className="flex min-h-dvh flex-col">
      <DemoBanner />
      <header className="border-b bg-background/95">
        <div className="mx-auto flex h-16 w-full items-center justify-between gap-4 px-6 md:px-10">
          <Wordmark />
          <nav aria-label={t("main")} className="flex items-center gap-2">
            {me ? (
              <ButtonLink href={homeFor(me.role)} size="sm">
                {me.role === "member" ? t("home") : t("workspace")}
              </ButtonLink>
            ) : (
              <>
                <ButtonLink href="/masuk" variant="ghost" size="sm">
                  {t("signIn")}
                </ButtonLink>
                <ButtonLink href="/daftar" size="sm">
                  {t("register")}
                </ButtonLink>
              </>
            )}
          </nav>
        </div>
      </header>
      <main id="konten" className="mx-auto w-full flex-1 px-0 py-8 md:py-12">
        {children}
      </main>
      <footer className="border-t bg-card">
        <div className="mx-auto flex w-full flex-col gap-3 px-6 py-6 text-sm text-muted-foreground md:flex-row md:items-center md:justify-between md:px-10">
          <p>{tl("footer")}</p>
          <nav aria-label={t("help")} className="flex flex-wrap gap-x-4 gap-y-2">
            <Link href="/bantuan" className="hover:text-foreground">
              {t("help")}
            </Link>
            <Link href="/privasi" className="hover:text-foreground">
              {t("privacy")}
            </Link>
            <Link href="/ketentuan" className="hover:text-foreground">
              {t("terms")}
            </Link>
          </nav>
        </div>
      </footer>
    </div>
  )
}
