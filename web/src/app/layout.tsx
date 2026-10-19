import type { Metadata, Viewport } from "next"
import localFont from "next/font/local"
import { NextIntlClientProvider } from "next-intl"
import { getTranslations } from "next-intl/server"

import { Providers } from "./providers"
import "./globals.css"

const inter = localFont({ src: "./fonts/inter-latin-variable.woff2", weight: "100 900", variable: "--font-inter", display: "swap" })

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("meta")
  return {
    title: { default: t("title"), template: `%s | ${t("title")}` },
    description: t("description"),
    applicationName: t("title"),
  }
}

export const viewport: Viewport = {
  themeColor: "#1E40AF",
  width: "device-width",
  initialScale: 1,
}

export default async function RootLayout({ children }: LayoutProps<"/">) {
  const t = await getTranslations("common")
  return (
    <html lang="id" className={inter.variable}>
      <body className="min-h-dvh antialiased">
        <a
          href="#konten"
          className="sr-only z-50 rounded-lg bg-primary px-4 py-2 text-primary-foreground focus:not-sr-only focus:fixed focus:top-2 focus:left-2"
        >
          {t("skipToContent")}
        </a>
        <NextIntlClientProvider>
          <Providers>{children}</Providers>
        </NextIntlClientProvider>
      </body>
    </html>
  )
}
