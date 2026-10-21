"use client"

import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation } from "@tanstack/react-query"
import { ArrowUpRight } from "lucide-react"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useMemo } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod"

import { Field } from "@/components/report/form/field"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api/client"
import { call, messageFor } from "@/lib/api/problem"
import { homeFor, safeNext } from "@/lib/paths"

import { FormAlert } from "./form-alert"
import { PasswordInput } from "./password-input"

const DEMO_ACTORS = [
  {
    roleLabel: "Penemu (Mahasiswa)",
    name: "Budi Santoso",
    email: "anggota.penemu@demo.simpanugm.my.id",
    dashboardMode: "penemu",
    image: "/illustrations/finder-student.webp",
    tone: "owner-amber",
  },
  {
    roleLabel: "Pemilik (Mahasiswa)",
    name: "Siti Rahma",
    email: "anggota.pemilik@demo.simpanugm.my.id",
    image: "/illustrations/owner-student.webp",
    tone: "owner-mint",
  },
  {
    roleLabel: "Petugas (Pos Perpus)",
    name: "Agus Prasetyo",
    email: "petugas.pos1@demo.simpanugm.my.id",
    image: "/illustrations/officer-student.webp",
    tone: "owner-sky",
  },
  {
    roleLabel: "Administrator",
    name: "Admin SimpanUGM",
    email: "admin@demo.simpanugm.my.id",
    image: "/illustrations/admin-student.webp",
    tone: "owner-violet",
  },
] as const

export function LoginForm({ next }: { next?: string }) {
  const t = useTranslations("auth")
  const te = useTranslations("errors")
  const router = useRouter()
  const isDemo = process.env.NEXT_PUBLIC_DEMO_MODE === "true"
  const schema = useMemo(
    () =>
      z.object({
        email: z
          .string()
          .trim()
          .min(1, { error: te("required") }),
        password: z.string().min(1, { error: te("required") }),
      }),
    [te],
  )
  const form = useForm<z.input<typeof schema>>({ resolver: zodResolver(schema), defaultValues: { email: "", password: "" } })
  const login = useMutation({
    mutationFn: (v: z.output<typeof schema> & { dashboardMode?: "penemu" }) =>
      call(api.POST("/auth/login", { body: { email: v.email, password: v.password } })),
    onSuccess: (me, values) => {
      const home = me.role === "member" && values.dashboardMode === "penemu" ? "/beranda?mode=penemu" : homeFor(me.role)
      router.replace(safeNext(next, home))
      router.refresh()
    },
  })
  const errors = form.formState.errors
  return (
    <form noValidate onSubmit={form.handleSubmit((v) => login.mutate(v))} className="flex flex-col gap-5">
      <FormAlert>{login.error ? messageFor(login.error, te) : null}</FormAlert>

      {isDemo ? (
        <section aria-labelledby="demo-quick-login" className="flex flex-col gap-3">
          <div>
            <h2 id="demo-quick-login" className="text-sm font-bold text-[#101f50]">
              {t("login.demoTitle")}
            </h2>
            <p className="mt-1 text-xs leading-relaxed text-muted-foreground">{t("login.demoLead")}</p>
          </div>
          <div className="grid grid-cols-1 gap-2 min-[380px]:grid-cols-2 sm:gap-3">
            {DEMO_ACTORS.map((act) => (
              <button
                key={act.email}
                type="button"
                disabled={login.isPending || login.isSuccess}
                onClick={() => {
                  form.setValue("email", act.email)
                  form.setValue("password", "kata sandi panjang 2026")
                  login.mutate({ email: act.email, password: "kata sandi panjang 2026", dashboardMode: "dashboardMode" in act ? act.dashboardMode : undefined })
                }}
                className={`owner-feature group relative flex min-h-24 min-w-0 items-center gap-2 overflow-hidden rounded-2xl border p-2.5 text-left transition-colors hover:border-primary focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:outline-none disabled:cursor-wait disabled:opacity-60 sm:gap-3 ${act.tone}`}
              >
                <span className="relative size-14 shrink-0 overflow-hidden rounded-xl border border-white/80 bg-white/70 sm:size-16" aria-hidden>
                  <img src={act.image} width={96} height={96} alt="" className="size-full origin-[50%_35%] scale-[1.55] object-cover object-[40%_30%]" />
                </span>
                <span className="min-w-0 pr-2">
                  <span className="block text-xs leading-snug font-bold text-[#101f50] sm:text-[13px]">{act.roleLabel}</span>
                  <span className="mt-1 block text-[11px] leading-snug text-slate-600">{act.name}</span>
                </span>
                <ArrowUpRight className="absolute top-2 right-2 size-3.5 opacity-60" aria-hidden />
              </button>
            ))}
          </div>
        </section>
      ) : null}

      {isDemo ? (
        <div className="flex items-center gap-3 text-[11px] text-muted-foreground">
          <span className="h-px flex-1 bg-border" aria-hidden />
          {t("login.manualSeparator")}
          <span className="h-px flex-1 bg-border" aria-hidden />
        </div>
      ) : null}

      <Field id="email" label={t("email")} error={errors.email?.message}>
        {(aria) => <Input {...aria} type="email" autoComplete="username" inputMode="email" placeholder={t("emailPlaceholder")} {...form.register("email")} />}
      </Field>
      <Field id="password" label={t("password")} error={errors.password?.message}>
        {(aria) => <PasswordInput {...aria} autoComplete="current-password" {...form.register("password")} />}
      </Field>
      <div className="-mt-2 text-right">
        <Link href="/lupa-sandi" className="text-sm text-primary underline-offset-4 hover:underline">
          {t("login.forgot")}
        </Link>
      </div>
      <Button type="submit" size="lg" disabled={login.isPending || login.isSuccess}>
        {t("login.submit")}
      </Button>
    </form>
  )
}
