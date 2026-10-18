"use client"

import { useMutation } from "@tanstack/react-query"
import { CircleCheck, LoaderCircle, MailX } from "lucide-react"
import { useTranslations } from "next-intl"
import { useEffect, useRef } from "react"

import { ButtonLink } from "@/components/ui/button-link"
import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"

// the token is posted once from the browser, so a mail scanner fetching the link does not consume it
export function VerifyEmail({ token }: { token?: string }) {
  const t = useTranslations("auth.verify")
  const sent = useRef(false)
  const verify = useMutation({ mutationFn: (value: string) => call(api.POST("/auth/verify-email", { body: { token: value } })) })
  const { mutate } = verify
  useEffect(() => {
    if (token && !sent.current) {
      sent.current = true
      mutate(token)
    }
  }, [mutate, token])

  if (!token) {
    return <p className="text-muted-foreground">{t("missing")}</p>
  }
  if (verify.isSuccess) {
    return (
      <div className="flex flex-col items-center gap-3 text-center">
        <CircleCheck className="size-10 text-success" aria-hidden />
        <h2 className="text-lg font-semibold">{t("successTitle")}</h2>
        <p className="text-muted-foreground">{t("successBody")}</p>
        <ButtonLink href="/beranda">{t("toHome")}</ButtonLink>
      </div>
    )
  }
  if (verify.isError) {
    return (
      <div className="flex flex-col items-center gap-3 text-center">
        <MailX className="size-10 text-warning" aria-hidden />
        <h2 className="text-lg font-semibold">{t("failedTitle")}</h2>
        <p className="text-muted-foreground">{t("failedBody")}</p>
        <ButtonLink href="/beranda" variant="outline">
          {t("toLogin")}
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
