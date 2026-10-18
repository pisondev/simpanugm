"use client"

import { useMutation } from "@tanstack/react-query"
import Link from "next/link"
import { useTranslations } from "next-intl"
import { useState } from "react"

import { Field } from "@/components/report/form/field"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api/client"
import { call, isApiError, messageFor } from "@/lib/api/problem"

import { FormAlert } from "./form-alert"

// US-03 criterion 1: the same answer for every address, only transport errors are shown
export function ForgotForm() {
  const t = useTranslations("auth")
  const te = useTranslations("errors")
  const [email, setEmail] = useState("")
  const [required, setRequired] = useState(false)
  const forgot = useMutation({ mutationFn: () => call(api.POST("/auth/password/forgot", { body: { email: email.trim() } })) })
  const done = forgot.isSuccess || (forgot.isError && isApiError(forgot.error) && forgot.error.status === 422)
  if (done) {
    return (
      <div className="flex flex-col gap-4">
        <FormAlert tone="success">{t("forgot.done")}</FormAlert>
        <Link href="/masuk" className="text-sm text-primary underline-offset-4 hover:underline">
          {t("forgot.backToLogin")}
        </Link>
      </div>
    )
  }
  return (
    <form
      noValidate
      className="flex flex-col gap-5"
      onSubmit={(e) => {
        e.preventDefault()
        if (!email.trim()) {
          setRequired(true)
          return
        }
        setRequired(false)
        forgot.mutate()
      }}
    >
      <FormAlert>{forgot.error ? messageFor(forgot.error, te) : null}</FormAlert>
      <Field id="email" label={t("email")} error={required ? te("required") : undefined}>
        {(aria) => (
          <Input
            {...aria}
            type="email"
            autoComplete="email"
            inputMode="email"
            placeholder={t("emailPlaceholder")}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
          />
        )}
      </Field>
      <Button type="submit" size="lg" disabled={forgot.isPending}>
        {t("forgot.submit")}
      </Button>
    </form>
  )
}
