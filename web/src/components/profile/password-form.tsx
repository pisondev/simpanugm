"use client"

import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { useTranslations } from "next-intl"
import { useMemo } from "react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import { z } from "zod"

import { FormAlert } from "@/components/auth/form-alert"
import { PasswordInput } from "@/components/auth/password-input"
import { Field } from "@/components/report/form/field"
import { Button } from "@/components/ui/button"
import { api } from "@/lib/api/client"
import { call, fieldErrors, messageFor } from "@/lib/api/problem"

export function PasswordForm() {
  const t = useTranslations("profile")
  const ta = useTranslations("auth")
  const te = useTranslations("errors")
  const client = useQueryClient()
  const schema = useMemo(
    () =>
      z
        .object({
          current_password: z.string().min(1, { error: te("required") }),
          new_password: z
            .string()
            .min(10, { error: (iss) => te("tooShort", { min: 10, count: String(iss.input ?? "").length }) })
            .max(128, { error: te("tooLong", { max: 128 }) }),
          confirm: z.string(),
        })
        .refine((v) => v.new_password === v.confirm, { path: ["confirm"], error: ta("reset.mismatch") }),
    [ta, te],
  )
  const form = useForm<z.input<typeof schema>>({ resolver: zodResolver(schema), defaultValues: { current_password: "", new_password: "", confirm: "" } })
  const change = useMutation({
    mutationFn: (v: z.output<typeof schema>) => call(api.PUT("/me/password", { body: { current_password: v.current_password, new_password: v.new_password } })),
    onSuccess: () => {
      form.reset()
      toast.success(t("passwordChanged"))
      void client.invalidateQueries({ queryKey: ["my-sessions"] })
    },
    onError: (err) => {
      const f = fieldErrors(err)
      if (f.current_password) form.setError("current_password", { message: f.current_password })
      if (f.new_password) form.setError("new_password", { message: f.new_password })
    },
  })
  const errors = form.formState.errors
  const general = change.error && Object.keys(fieldErrors(change.error)).length === 0 ? messageFor(change.error, te) : null
  return (
    <form noValidate onSubmit={form.handleSubmit((v) => change.mutate(v))} className="flex flex-col gap-5">
      <FormAlert>{general}</FormAlert>
      <Field id="current-password" label={t("currentPassword")} error={errors.current_password?.message}>
        {(aria) => <PasswordInput {...aria} autoComplete="current-password" {...form.register("current_password")} />}
      </Field>
      <Field id="new-password" label={t("newPassword")} hint={ta("passwordHint")} error={errors.new_password?.message}>
        {(aria) => <PasswordInput {...aria} autoComplete="new-password" {...form.register("new_password")} />}
      </Field>
      <Field id="confirm-password" label={t("confirmPassword")} error={errors.confirm?.message}>
        {(aria) => <PasswordInput {...aria} autoComplete="new-password" {...form.register("confirm")} />}
      </Field>
      <Button type="submit" className="self-start" disabled={change.isPending}>
        {t("changePassword")}
      </Button>
    </form>
  )
}
