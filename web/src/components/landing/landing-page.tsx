import Image from "next/image"
import { ArrowRight, Backpack, Headphones, KeyRound, Search, Settings2, ShieldCheck, UserRound, Wallet } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import type { CSSProperties } from "react"

import { PageTransition, Reveal } from "@/components/motion/primitives"
import { ButtonLink } from "@/components/ui/button-link"

export default function LandingPage() {
  const t = useTranslations("landing")
  const steps = [
    { title: t("step1Title"), body: t("step1Body") },
    { title: t("step2Title"), body: t("step2Body") },
    { title: t("step3Title"), body: t("step3Body") },
  ]
  const roles = [
    { icon: UserRound, image: "/landing/roles/owner.png", title: t("roleOwnerTitle"), body: t("roleOwnerBody"), tone: "gold" },
    { icon: Search, image: "/landing/roles/finder.png", title: t("roleFinderTitle"), body: t("roleFinderBody"), tone: "blue" },
    { icon: ShieldCheck, image: "/landing/roles/officer.png", title: t("roleOfficerTitle"), body: t("roleOfficerBody"), tone: "green" },
    { icon: Settings2, image: "/landing/roles/admin.png", title: t("roleAdminTitle"), body: t("roleAdminBody"), tone: "violet" },
  ]

  return (
    <PageTransition className="landing-page flex flex-col gap-12 md:gap-16">
      <section className="landing-hero relative isolate overflow-hidden">
        <div className="landing-hero__art" aria-hidden>
          <Image src="/landing/campus-entrance.png" alt="" fill priority sizes="(max-width: 768px) 100vw, 1152px" className="object-cover object-center" />
        </div>
        <div className="landing-hero__wash" aria-hidden />
        <div className="landing-floating-items" aria-hidden>
          <span className="landing-floating-item landing-floating-item--wallet">
            <Wallet />
          </span>
          <span className="landing-floating-item landing-floating-item--headphones">
            <Headphones />
          </span>
          <span className="landing-floating-item landing-floating-item--key">
            <KeyRound />
          </span>
          <span className="landing-floating-item landing-floating-item--backpack">
            <Backpack />
          </span>
        </div>
        <div className="landing-hero__inner">
          <Reveal className="landing-hero__copy">
            <p className="landing-hero__eyebrow">
              <span aria-hidden className="size-1.5 rounded-full bg-primary" />
              {t("eyebrow")}
            </p>
            <span className="landing-hero__accent" aria-hidden />
            <h1 className="landing-hero__title">
              {t("title")}
              <br />
              <span>{t("titleAccent")}</span>
            </h1>
            <p className="landing-hero__lead">{t("lead")}</p>
          </Reveal>
          <div className="landing-hero__action-row">
            <div className="landing-hero__cta">
              <div className="landing-hero__actions">
                <ButtonLink href="/daftar" size="lg">
                  {t("start")}
                  <ArrowRight aria-hidden />
                </ButtonLink>
                <ButtonLink href="/lapor/temuan" size="lg" variant="outline" className="bg-card/90">
                  {t("foundAction")}
                </ButtonLink>
              </div>
              <Link href="/masuk" className="landing-hero__sign-in">
                {t("signIn")}
              </Link>
            </div>
            <div className="landing-roles" aria-label="Peran di SimpanUGM">
              {roles.map((role) => (
                <article key={role.title} className={`landing-role-card landing-role-card--${role.tone}`}>
                  <div className="landing-role-card__portrait" aria-hidden>
                    <Image src={role.image} alt="" fill sizes="(max-width: 767px) 40vw, 180px" className="object-contain object-bottom" />
                  </div>
                  <div className="landing-role-card__body">
                    <span className="landing-role-card__icon">
                      <role.icon className="size-4" aria-hidden />
                    </span>
                    <div>
                      <h2>{role.title}</h2>
                      <p>{role.body}</p>
                    </div>
                  </div>
                </article>
              ))}
            </div>
          </div>
        </div>
      </section>

      <section aria-labelledby="cara-kerja" className="landing-process">
        <div className="landing-section-heading">
          <div>
            <h2 id="cara-kerja">{t("howTitle")}</h2>
            <span aria-hidden className="landing-section-heading__accent" />
          </div>
          <p>{t("howLead")}</p>
        </div>
        <ol className="landing-process__list">
          {steps.map((step, index) => (
            <li key={step.title} style={{ "--motion-index": index } as CSSProperties} className="landing-process-card">
              <div className="landing-process-card__visual" aria-hidden>
                <Image
                  src="/landing/process-triptych.png"
                  alt=""
                  width={2048}
                  height={683}
                  className="landing-process-card__sprite"
                  style={{ left: `${index * -100}%` }}
                />
              </div>
              <div className="landing-process-card__copy">
                <span className="landing-process-card__number">0{index + 1}</span>
                <h3>{step.title}</h3>
                <p>{step.body}</p>
              </div>
              {index < steps.length - 1 ? <ArrowRight className="landing-process-card__arrow" aria-hidden /> : null}
            </li>
          ))}
        </ol>
      </section>

      <section className="landing-privacy">
        <div className="landing-privacy__icon">
          <ShieldCheck className="size-7" aria-hidden />
        </div>
        <div>
          <h2>{t("privacyTitle")}</h2>
          <p>{t("privacyBody")}</p>
        </div>
        <div className="landing-privacy__art" aria-hidden>
          <ShieldCheck className="size-20" strokeWidth={1.2} />
        </div>
      </section>
    </PageTransition>
  )
}
