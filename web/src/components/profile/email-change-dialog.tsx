"use client"

import { useMutation } from "@tanstack/react-query"
import { useTranslations } from "next-intl"
import { useState } from "react"

import { FormAlert } from "@/components/auth/form-alert"
import { PasswordInput } from "@/components/auth/password-input"
import { Field } from "@/components/report/form/field"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api/client"
import { call, fieldErrors, messageFor } from "@/lib/api/problem"

export function EmailChangeDialog() {
  const t = useTranslations("profile")
  const te = useTranslations("errors")
  const tc = useTranslations("common")
  const [open, setOpen] = useState(false)
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const start = useMutation({ mutationFn: () => call(api.POST("/me/email", { body: { email: email.trim(), password } })) })
  const errors = start.error ? fieldErrors(start.error) : {}
  const general = start.error && Object.keys(errors).length === 0 ? messageFor(start.error, te) : null
  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        setOpen(v)
        if (v) {
          start.reset()
          setEmail("")
          setPassword("")
        }
      }}
    >
      <DialogTrigger render={<Button variant="outline" size="sm" />}>{t("changeEmail")}</DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("changeEmailTitle")}</DialogTitle>
          <DialogDescription>{t("changeEmailLead")}</DialogDescription>
        </DialogHeader>
        {start.isSuccess ? (
          <FormAlert tone="success">{t("emailChangeSent")}</FormAlert>
        ) : (
          <form
            id="email-change"
            noValidate
            className="flex flex-col gap-4"
            onSubmit={(e) => {
              e.preventDefault()
              start.mutate()
            }}
          >
            <FormAlert>{general}</FormAlert>
            <Field id="new-email" label={t("newEmail")} error={errors.email}>
              {(aria) => <Input {...aria} type="email" autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} />}
            </Field>
            <Field id="email-password" label={t("currentPassword")} error={errors.current_password}>
              {(aria) => <PasswordInput {...aria} autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />}
            </Field>
          </form>
        )}
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            {start.isSuccess ? tc("close") : tc("cancel")}
          </Button>
          {start.isSuccess ? null : (
            <Button type="submit" form="email-change" disabled={start.isPending || !email || !password}>
              {t("sendConfirmation")}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
