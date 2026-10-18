"use client"

import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation } from "@tanstack/react-query"
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

export function LoginForm({ next }: { next?: string }) {
  const t = useTranslations("auth")
  const te = useTranslations("errors")
  const router = useRouter()
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
    mutationFn: (v: z.output<typeof schema>) => call(api.POST("/auth/login", { body: v })),
    onSuccess: (me) => {
      router.replace(safeNext(next, homeFor(me.role)))
      router.refresh()
    },
  })
  const errors = form.formState.errors
  return (
    <form noValidate onSubmit={form.handleSubmit((v) => login.mutate(v))} className="flex flex-col gap-5">
      <FormAlert>{login.error ? messageFor(login.error, te) : null}</FormAlert>
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
