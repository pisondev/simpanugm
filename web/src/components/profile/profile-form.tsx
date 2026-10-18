"use client"

import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation } from "@tanstack/react-query"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useMemo } from "react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import { z } from "zod"

import { FormAlert } from "@/components/auth/form-alert"
import { Field } from "@/components/report/form/field"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { api } from "@/lib/api/client"
import { call, fieldErrors, messageFor } from "@/lib/api/problem"
import type { Me } from "@/lib/api/types"

const fields = ["full_name", "phone", "nim_nip"] as const

// US-05 criterion 1, the email changes through its own confirmation flow
export function ProfileForm({ me }: { me: Pick<Me, "full_name" | "phone" | "nim_nip"> }) {
  const t = useTranslations("profile")
  const ta = useTranslations("auth")
  const te = useTranslations("errors")
  const router = useRouter()
  const schema = useMemo(
    () =>
      z.object({
        full_name: z
          .string()
          .trim()
          .min(2, { error: te("required") })
          .max(100, { error: te("tooLong", { max: 100 }) }),
        phone: z
          .string()
          .trim()
          .max(20, { error: te("tooLong", { max: 20 }) }),
        nim_nip: z
          .string()
          .trim()
          .max(30, { error: te("tooLong", { max: 30 }) }),
      }),
    [te],
  )
  const form = useForm<z.input<typeof schema>>({
    resolver: zodResolver(schema),
    defaultValues: { full_name: me.full_name, phone: me.phone ?? "", nim_nip: me.nim_nip ?? "" },
  })
  const save = useMutation({
    mutationFn: (v: z.output<typeof schema>) => call(api.PATCH("/me", { body: v })),
    onSuccess: (updated) => {
      form.reset({ full_name: updated.full_name, phone: updated.phone ?? "", nim_nip: updated.nim_nip ?? "" })
      toast.success(t("saved"))
      router.refresh()
    },
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
  const general = save.error && Object.keys(fieldErrors(save.error)).length === 0 ? messageFor(save.error, te) : null
  return (
    <form noValidate onSubmit={form.handleSubmit((v) => save.mutate(v))} className="flex flex-col gap-5">
      <FormAlert>{general}</FormAlert>
      <Field id="full-name" label={t("name")} error={errors.full_name?.message}>
        {(aria) => <Input {...aria} autoComplete="name" {...form.register("full_name")} />}
      </Field>
      <Field id="phone" label={t("phone")} hint={t("phoneHint")} error={errors.phone?.message}>
        {(aria) => <Input {...aria} type="tel" inputMode="tel" autoComplete="tel" {...form.register("phone")} />}
      </Field>
      <Field id="nim-nip" label={t("nimNip")} error={errors.nim_nip?.message}>
        {(aria) => <Input {...aria} placeholder={ta("register.nimNipPlaceholder")} {...form.register("nim_nip")} />}
      </Field>
      <Button type="submit" className="self-start" disabled={save.isPending || !form.formState.isDirty}>
        {t("save")}
      </Button>
    </form>
  )
}
