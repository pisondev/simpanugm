import { HandHeart, Search, ShieldCheck, Sparkles } from "lucide-react"
import { useTranslations } from "next-intl"

import { ButtonLink } from "@/components/ui/button-link"

export default function LandingPage() {
  const t = useTranslations("landing")
  const steps = [
    { icon: Search, title: t("step1Title"), body: t("step1Body") },
    { icon: Sparkles, title: t("step2Title"), body: t("step2Body") },
    { icon: HandHeart, title: t("step3Title"), body: t("step3Body") },
  ]
  return (
    <div className="flex flex-col gap-14">
      <section className="flex max-w-2xl flex-col gap-5">
        <h1 className="text-3xl font-semibold tracking-tight text-balance md:text-4xl">{t("title")}</h1>
        <p className="text-lg text-muted-foreground">{t("lead")}</p>
        <div className="flex flex-col gap-3 sm:flex-row">
          <ButtonLink href="/daftar" size="lg">
            {t("start")}
          </ButtonLink>
          <ButtonLink href="/masuk" size="lg" variant="outline">
            {t("signIn")}
          </ButtonLink>
        </div>
      </section>
      <section aria-labelledby="cara-kerja" className="flex flex-col gap-5">
        <h2 id="cara-kerja" className="text-xl font-semibold">
          {t("howTitle")}
        </h2>
        <ol className="grid gap-4 md:grid-cols-3">
          {steps.map((s, i) => (
            <li key={s.title} className="flex flex-col gap-3 rounded-xl border bg-card p-5 shadow-sm">
              <span className="flex items-center gap-3">
                <span className="flex size-10 items-center justify-center rounded-full bg-primary-soft text-primary">
                  <s.icon className="size-5" aria-hidden />
                </span>
                <span className="text-sm font-medium text-muted-foreground">{i + 1}</span>
              </span>
              <h3 className="text-lg font-semibold">{s.title}</h3>
              <p className="text-muted-foreground">{s.body}</p>
            </li>
          ))}
        </ol>
      </section>
      <section className="flex items-start gap-4 rounded-xl border bg-card p-5">
        <ShieldCheck className="mt-0.5 size-6 shrink-0 text-success" aria-hidden />
        <div>
          <h2 className="font-semibold">{t("privacyTitle")}</h2>
          <p className="text-muted-foreground">{t("privacyBody")}</p>
        </div>
      </section>
    </div>
  )
}
