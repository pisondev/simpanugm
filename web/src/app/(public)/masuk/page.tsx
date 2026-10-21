import type { Metadata } from "next"
import Link from "next/link"
import { redirect } from "next/navigation"
import { getTranslations } from "next-intl/server"
import { ArrowRight, Check, ShieldCheck } from "lucide-react"

import { FormAlert } from "@/components/auth/form-alert"
import { LoginForm } from "@/components/auth/login-form"
import { PageTransition } from "@/components/motion/primitives"
import { homeFor, safeNext } from "@/lib/paths"
import { getMe } from "@/lib/session"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("auth.login")
  return { title: t("title") }
}

export default async function SignInPage({ searchParams }: PageProps<"/masuk">) {
  const params = await searchParams
  const next = typeof params.next === "string" ? params.next : undefined
  const me = await getMe().catch(() => null)
  if (me) {
    redirect(safeNext(next, homeFor(me.role)))
  }
  const t = await getTranslations("auth.login")
  return (
    <PageTransition className="mx-auto grid w-full max-w-[1280px] gap-5 px-4 md:px-6 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)] lg:gap-7">
      <aside
        aria-label={t("introLabel")}
        className="relative isolate hidden overflow-hidden rounded-3xl border border-blue-100 bg-[#eef5ff] p-8 lg:flex lg:flex-col xl:p-10"
      >
        <p className="relative z-10 self-start rounded-full border border-blue-200 bg-white/90 px-3 py-1 text-xs font-bold tracking-wide text-blue-800">
          SIMPANUGM
        </p>
        <p className="relative z-10 mt-7 text-[clamp(2rem,3.5vw,3.2rem)] leading-[1.08] font-extrabold tracking-tight text-[#101f50]">
          {t("introStart")}
          <br />
          <span className="text-[#0759e8]">{t("introAccent")}</span>
        </p>
        <p className="relative z-10 mt-4 max-w-sm text-sm leading-relaxed text-slate-600">{t("introLead")}</p>
        <div className="relative mt-5 min-h-52 flex-1">
          <img
            src="/illustrations/campus-entrance.webp"
            width={1280}
            height={720}
            alt=""
            aria-hidden
            className="absolute inset-0 size-full rounded-2xl object-cover object-center"
          />
        </div>
        <div className="relative z-10 mt-5 rounded-2xl border border-white bg-white/85 p-4">
          <p className="flex items-center gap-2 text-sm font-bold text-[#101f50]">
            <ShieldCheck className="size-4 text-blue-700" aria-hidden />
            {t("introHumanTitle")}
          </p>
          <p className="mt-2 text-xs leading-relaxed text-slate-600">{t("introHumanLead")}</p>
          <ol className="mt-4 flex flex-wrap items-center gap-2 text-[11px] font-semibold text-blue-800">
            {[t("introReport"), t("introMatch"), t("introVerify")].map((label, index) => (
              <li key={label} className="flex items-center gap-2">
                <span className="flex items-center gap-1.5 rounded-full bg-blue-50 px-2 py-1">
                  <Check className="size-3" aria-hidden />
                  {label}
                </span>
                {index < 2 ? <ArrowRight className="size-3 text-blue-400" aria-hidden /> : null}
              </li>
            ))}
          </ol>
        </div>
      </aside>
      <div className="min-w-0 rounded-3xl border border-blue-100 bg-white p-5 shadow-surface sm:p-8 lg:p-9">
        <header>
          <p className="mb-2 text-xs font-bold tracking-wide text-blue-700">{t("eyebrow")}</p>
          <h1 className="text-3xl font-extrabold tracking-tight text-[#101f50]">{t("title")}</h1>
          <p className="mt-2 text-sm leading-relaxed text-muted-foreground">{t("lead")}</p>
        </header>
        <div className="mt-6 flex flex-col gap-5">
          {params.expired ? <FormAlert>{t("expired")}</FormAlert> : null}
          <LoginForm next={next} />
        </div>
        <p className="mt-6 text-center text-sm text-muted-foreground">
          {t("noAccount")}{" "}
          <Link href="/daftar" className="font-semibold text-primary underline-offset-4 hover:underline">
            {t("register")}
          </Link>
        </p>
      </div>
    </PageTransition>
  )
}
