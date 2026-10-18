"use client"

import { useMutation } from "@tanstack/react-query"
import { MailWarning } from "lucide-react"
import { useTranslations } from "next-intl"
import { toast } from "sonner"

import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { api } from "@/lib/api/client"
import { call, messageFor } from "@/lib/api/problem"

export function VerifyBanner() {
  const t = useTranslations("auth.verify")
  const te = useTranslations("errors")
  const resend = useMutation({
    mutationFn: () => call(api.POST("/auth/resend-verification")),
    onSuccess: () => toast.success(t("resent")),
    onError: (err) => toast.error(messageFor(err, te)),
  })
  return (
    <Alert className="mb-6 border-warning/40 bg-warning/5">
      <MailWarning className="text-warning" aria-hidden />
      <AlertDescription className="flex flex-col gap-3 text-foreground sm:flex-row sm:items-center sm:justify-between">
        <span>{t("banner")}</span>
        <Button variant="outline" size="sm" onClick={() => resend.mutate()} disabled={resend.isPending || resend.isSuccess}>
          {t("bannerAction")}
        </Button>
      </AlertDescription>
    </Alert>
  )
}
