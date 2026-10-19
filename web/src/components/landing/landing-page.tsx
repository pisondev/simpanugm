import { ArrowRight, Backpack, HandHeart, Headphones, KeyRound, PackageCheck, Search, ShieldCheck, Sparkles, Wallet } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import type { CSSProperties } from "react"

import { PageTransition, Reveal } from "@/components/motion/primitives"
import { ButtonLink } from "@/components/ui/button-link"

export default function LandingPage() {
  const t = useTranslations("landing")
  const steps = [
    { icon: Search, title: t("step1Title"), body: t("step1Body") },
    { icon: Sparkles, title: t("step2Title"), body: t("step2Body") },
    { icon: HandHeart, title: t("step3Title"), body: t("step3Body") },
  ]
  return (
    <PageTransition className="flex flex-col gap-12 md:gap-16">
      <section className="hero-wash relative isolate grid items-center gap-8 overflow-hidden border px-5 py-8 sm:p-9 lg:grid-cols-[1.25fr_1fr] lg:gap-10 lg:py-12">
        <Reveal className="flex flex-col items-start gap-5">
          <p className="flex items-center gap-2 text-xs font-semibold tracking-[0.12em] text-primary uppercase">
            <span aria-hidden className="size-1.5 rounded-full bg-primary" />
            {t("eyebrow")}
          </p>
          <h1 className="text-4xl leading-[1.08] font-semibold tracking-[-0.04em] text-balance sm:text-5xl">
            {t("title")}
            <br />
            <span className="text-primary">{t("titleAccent")}</span>
          </h1>
          <p className="max-w-lg text-base leading-relaxed text-muted-foreground">{t("lead")}</p>
          <div className="mt-1 flex w-full flex-col gap-3 sm:w-auto sm:flex-row sm:flex-wrap">
            <ButtonLink href="/daftar" size="lg">
              {t("start")}
              <ArrowRight aria-hidden />
            </ButtonLink>
            <ButtonLink href="/lapor/temuan" size="lg" variant="outline" className="bg-card">
              {t("foundAction")}
            </ButtonLink>
          </div>
          <Link href="/masuk" className="inline-flex min-h-11 items-center text-sm font-medium text-primary underline-offset-4 hover:underline">
            {t("signIn")}
          </Link>
        </Reveal>
        <Reveal
          variant="fade"
          className="relative flex min-h-64 flex-col items-center justify-center gap-5 overflow-hidden rounded-2xl border border-primary/10 bg-card/60 px-4 py-8 sm:min-h-80"
        >
          <div aria-hidden className="pointer-events-none absolute inset-5 rounded-full border border-dashed border-primary/15" />
          <div aria-hidden className="pointer-events-none absolute inset-12 rounded-full border border-primary/10" />
          <div aria-hidden className="relative flex w-full justify-around">
            <span className="flex size-14 -rotate-6 items-center justify-center rounded-2xl border bg-card text-primary shadow-surface">
              <Wallet className="size-7" />
            </span>
            <span className="mt-3 flex size-12 rotate-6 items-center justify-center rounded-2xl border bg-card text-muted-foreground shadow-surface">
              <Headphones className="size-6" />
            </span>
          </div>
          <span
            aria-hidden
            className="relative flex size-24 items-center justify-center rounded-3xl border border-primary/10 bg-primary text-primary-foreground shadow-raised"
          >
            <PackageCheck className="size-12" strokeWidth={1.4} />
          </span>
          <div aria-hidden className="relative -mt-2 flex w-full justify-around">
            <span className="flex size-11 rotate-6 items-center justify-center rounded-xl border bg-card text-muted-foreground shadow-surface">
              <KeyRound className="size-5" />
            </span>
            <span className="-mt-3 flex size-14 -rotate-6 items-center justify-center rounded-2xl border bg-card text-primary shadow-surface">
              <Backpack className="size-7" />
            </span>
          </div>
          <p className="relative text-center text-sm font-medium text-muted-foreground">{t("community")}</p>
        </Reveal>
      </section>
      <section aria-labelledby="cara-kerja" className="flex flex-col gap-6">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <h2 id="cara-kerja" className="text-2xl font-semibold tracking-tight">
            {t("howTitle")}
          </h2>
          <p className="text-sm text-muted-foreground">{t("howLead")}</p>
        </div>
        <ol className="motion-stagger grid gap-4 md:grid-cols-3">
          {steps.map((s, i) => (
            <li key={s.title} style={{ "--motion-index": i } as CSSProperties} className="surface flex flex-col gap-3 p-5">
              <div className="mb-1 flex items-center justify-between">
                <span className="flex size-11 items-center justify-center rounded-xl bg-surface-inset text-primary">
                  <s.icon className="size-5" aria-hidden />
                </span>
                <span aria-hidden className="text-sm font-medium text-muted-foreground tabular-nums">
                  0{i + 1}
                </span>
              </div>
              <h3 className="text-lg font-semibold">{s.title}</h3>
              <p className="text-sm leading-relaxed text-muted-foreground">{s.body}</p>
            </li>
          ))}
        </ol>
      </section>
      <section className="flex items-start gap-4 rounded-xl border bg-surface-inset p-5 sm:p-6">
        <ShieldCheck className="mt-0.5 size-6 shrink-0 text-success-text" aria-hidden />
        <div className="flex flex-col gap-1">
          <h2 className="font-semibold">{t("privacyTitle")}</h2>
          <p className="text-sm leading-relaxed text-muted-foreground">{t("privacyBody")}</p>
        </div>
      </section>
    </PageTransition>
  )
}
