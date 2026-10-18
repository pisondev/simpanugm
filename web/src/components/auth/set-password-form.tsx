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
import { ButtonLink } from "@/components/ui/button-link"
import { api } from "@/lib/api/client"
import { call, fieldErrors, messageFor } from "@/lib/api/problem"
import { homeFor } from "@/lib/paths"

import { FormAlert } from "./form-alert"
import { PasswordInput } from "./password-input"

// one form for a password reset and for accepting an invitation, both trade a token for a new password
export function SetPasswordForm({ token, purpose }: { token: string; purpose: "reset" | "invite" }) {
  const t = useTranslations("auth")
  const te = useTranslations("errors")
  const router = useRouter()
  const schema = useMemo(
    () =>
      z
        .object({
          password: z
            .string()
            .min(10, { error: (iss) => te("tooShort", { min: 10, count: String(iss.input ?? "").length }) })
            .max(128, { error: te("tooLong", { max: 128 }) }),
          confirm: z.string(),
        })
        .refine((v) => v.password === v.confirm, { path: ["confirm"], error: t("reset.mismatch") }),
    [t, te],
  )
  const form = useForm<z.input<typeof schema>>({ resolver: zodResolver(schema), defaultValues: { password: "", confirm: "" } })
  const save = useMutation({
    mutationFn: async (password: string) => {
      if (purpose === "invite") {
        return call(api.POST("/auth/invitations/accept", { body: { token, password } }))
      }
      await call(api.POST("/auth/password/reset", { body: { token, password } }))
      return null
    },
    onSuccess: (me) => {
      if (me) {
        router.replace(homeFor(me.role))
        router.refresh()
      }
    },
    onError: (err) => {
      const message = fieldErrors(err).password
      if (message) {
        form.setError("password", { message })
      }
    },
  })
  const errors = form.formState.errors
  if (save.isSuccess && purpose === "reset") {
    return (
      <div className="flex flex-col gap-4">
        <FormAlert tone="success">{t("reset.done")}</FormAlert>
        <ButtonLink href="/masuk">{t("login.submit")}</ButtonLink>
      </div>
    )
  }
  const tokenProblem = save.error && !fieldErrors(save.error).password
  return (
    <form noValidate onSubmit={form.handleSubmit((v) => save.mutate(v.password))} className="flex flex-col gap-5">
      {tokenProblem ? (
        <FormAlert>
          {messageFor(save.error, te)}{" "}
          {purpose === "reset" ? (
            <Link href="/lupa-sandi" className="underline underline-offset-4">
              {t("reset.requestNew")}
            </Link>
          ) : null}
        </FormAlert>
      ) : null}
      <Field id="password" label={t("reset.newPassword")} hint={t("passwordHint")} error={errors.password?.message}>
        {(aria) => <PasswordInput {...aria} autoComplete="new-password" {...form.register("password")} />}
      </Field>
      <Field id="confirm" label={t("reset.confirm")} error={errors.confirm?.message}>
        {(aria) => <PasswordInput {...aria} autoComplete="new-password" {...form.register("confirm")} />}
      </Field>
      <Button type="submit" size="lg" disabled={save.isPending || save.isSuccess}>
        {purpose === "invite" ? t("invite.submit") : t("reset.submit")}
      </Button>
    </form>
  )
}
