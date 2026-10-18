"use client"

import { useMutation } from "@tanstack/react-query"
import { CircleCheck, LoaderCircle, MailX } from "lucide-react"
import { useTranslations } from "next-intl"
import { useEffect, useRef } from "react"

import { ButtonLink } from "@/components/ui/button-link"
import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"

// posted once from the browser, like the verification link, so a mail scanner does not consume it
export function ConfirmEmail({ token }: { token?: string }) {
  const t = useTranslations("auth.confirmEmail")
  const sent = useRef(false)
  const confirm = useMutation({ mutationFn: (value: string) => call(api.POST("/auth/email/confirm", { body: { token: value } })) })
  const { mutate } = confirm
  useEffect(() => {
    if (token && !sent.current) {
      sent.current = true
      mutate(token)
    }
  }, [mutate, token])
  if (!token) {
    return <p className="text-muted-foreground">{t("missing")}</p>
  }
  if (confirm.isSuccess || confirm.isError) {
    const ok = confirm.isSuccess
    const Icon = ok ? CircleCheck : MailX
    return (
      <div className="flex flex-col items-center gap-3 text-center">
        <Icon className={ok ? "size-10 text-success" : "size-10 text-warning"} aria-hidden />
        <p>{ok ? t("done") : t("failed")}</p>
        <ButtonLink href="/profil" variant={ok ? "default" : "outline"}>
          {t("toProfile")}
        </ButtonLink>
      </div>
    )
  }
  return (
    <p role="status" className="flex items-center justify-center gap-2 text-muted-foreground">
      <LoaderCircle className="size-5 animate-spin" aria-hidden />
      {t("working")}
    </p>
  )
}
