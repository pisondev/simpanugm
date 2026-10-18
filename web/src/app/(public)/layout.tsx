import Link from "next/link"
import { getTranslations } from "next-intl/server"

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
      <header className="border-b bg-card">
        <div className="mx-auto flex h-16 max-w-5xl items-center justify-between gap-4 px-4 md:px-6">
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
      <main id="konten" className="mx-auto w-full max-w-5xl flex-1 px-4 py-8 md:px-6 md:py-12">
        {children}
      </main>
      <footer className="border-t bg-card">
        <div className="mx-auto flex max-w-5xl flex-col gap-3 px-4 py-6 text-sm text-muted-foreground md:flex-row md:items-center md:justify-between md:px-6">
          <p>{tl("footer")}</p>
          <nav aria-label={t("help")} className="flex gap-4">
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
