"use client"

import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation } from "@tanstack/react-query"
import { MailCheck } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"
import { useMemo } from "react"
import { Controller, useForm } from "react-hook-form"
import { z } from "zod"

import { Field } from "@/components/report/form/field"
import { Button } from "@/components/ui/button"
import { ButtonLink } from "@/components/ui/button-link"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api/client"
import { call, fieldErrors, messageFor } from "@/lib/api/problem"
import { privacyVersion } from "@/lib/paths"

import { FormAlert } from "./form-alert"
import { PasswordInput } from "./password-input"

const fields = ["full_name", "email", "password", "nim_nip", "phone", "accept_privacy"] as const

export function RegisterForm() {
  const t = useTranslations("auth")
  const te = useTranslations("errors")
  const tc = useTranslations("common")
  const schema = useMemo(
    () =>
      z.object({
        full_name: z
          .string()
          .trim()
          .min(2, { error: te("required") })
          .max(100, { error: te("tooLong", { max: 100 }) }),
        email: z
          .string()
          .trim()
          .min(1, { error: te("required") })
          .max(254),
        password: z
          .string()
          .min(10, { error: (iss) => te("tooShort", { min: 10, count: String(iss.input ?? "").length }) })
          .max(128, { error: te("tooLong", { max: 128 }) }),
        nim_nip: z
          .string()
          .trim()
          .max(30, { error: te("tooLong", { max: 30 }) }),
        phone: z
          .string()
          .trim()
          .max(20, { error: te("tooLong", { max: 20 }) }),
        accept_privacy: z.boolean().refine((v) => v, { error: te("required") }),
      }),
    [te],
  )
  type Values = z.input<typeof schema>
  const form = useForm<Values>({
    resolver: zodResolver(schema),
    defaultValues: { full_name: "", email: "", password: "", nim_nip: "", phone: "", accept_privacy: false },
  })
  const register = useMutation({
    mutationFn: (v: z.output<typeof schema>) =>
      call(
        api.POST("/auth/register", {
          body: {
            full_name: v.full_name,
            email: v.email,
            password: v.password,
            nim_nip: v.nim_nip || undefined,
            phone: v.phone || undefined,
            privacy_version: privacyVersion,
            accept_privacy: v.accept_privacy,
          },
        }),
      ),
    onError: (err) => {
      for (const [name, message] of Object.entries(fieldErrors(err))) {
        const target = fields.find((f) => f === name)
        if (target) {
          form.setError(target, { message })
        }
      }
    },
  })
  const errors = form.formState.errors
  if (register.isSuccess) {
    return (
      <div className="flex flex-col items-center gap-3 text-center">
        <MailCheck className="size-10 text-success" aria-hidden />
        <h2 className="text-lg font-semibold">{t("register.doneTitle")}</h2>
        <p>{t("register.doneBody", { email: form.getValues("email") })}</p>
        <p className="text-sm text-muted-foreground">{t("register.doneHint")}</p>
        <ButtonLink href="/masuk" variant="outline">
          {t("login.submit")}
        </ButtonLink>
      </div>
    )
  }
  const hasFieldErrors = register.error ? Object.keys(fieldErrors(register.error)).length > 0 : false
  return (
    <form noValidate onSubmit={form.handleSubmit((v) => register.mutate(v))} className="flex flex-col gap-5">
      <FormAlert>{register.error && !hasFieldErrors ? messageFor(register.error, te) : null}</FormAlert>
      <Field id="full-name" label={t("register.fullName")} error={errors.full_name?.message}>
        {(aria) => <Input {...aria} autoComplete="name" {...form.register("full_name")} />}
      </Field>
      <Field id="email" label={t("email")} error={errors.email?.message}>
        {(aria) => <Input {...aria} type="email" autoComplete="email" inputMode="email" placeholder={t("emailPlaceholder")} {...form.register("email")} />}
      </Field>
      <Field id="password" label={t("password")} hint={t("passwordHint")} error={errors.password?.message}>
        {(aria) => <PasswordInput {...aria} autoComplete="new-password" {...form.register("password")} />}
      </Field>
      <Field id="nim-nip" label={t("register.nimNip")} optionalLabel={tc("optional")} error={errors.nim_nip?.message}>
        {(aria) => <Input {...aria} placeholder={t("register.nimNipPlaceholder")} {...form.register("nim_nip")} />}
      </Field>
      <Field id="phone" label={t("register.phone")} hint={t("register.phoneHint")} optionalLabel={tc("optional")} error={errors.phone?.message}>
        {(aria) => <Input {...aria} type="tel" inputMode="tel" autoComplete="tel" {...form.register("phone")} />}
      </Field>
      <Controller
        control={form.control}
        name="accept_privacy"
        render={({ field }) => (
          <div className="flex flex-col gap-1">
            <label className="flex min-h-11 items-start gap-3">
              <Checkbox
                className="mt-1"
                checked={field.value}
                onCheckedChange={(v) => field.onChange(Boolean(v))}
                aria-invalid={errors.accept_privacy ? true : undefined}
              />
              <span>
                {t.rich("register.accept", {
                  privacy: (chunks) => (
                    <Link href="/privasi" target="_blank" className="text-primary underline underline-offset-4">
                      {chunks}
                    </Link>
                  ),
                })}
              </span>
            </label>
            {errors.accept_privacy ? <p className="text-sm text-destructive">{errors.accept_privacy.message}</p> : null}
          </div>
        )}
      />
      <Button type="submit" size="lg" disabled={register.isPending}>
        {t("register.submit")}
      </Button>
    </form>
  )
}
