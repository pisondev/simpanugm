"use client"

import { useMutation } from "@tanstack/react-query"
import { UserPlus } from "lucide-react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import { FormAlert } from "@/components/auth/form-alert"
import { Field } from "@/components/report/form/field"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { NativeSelect } from "@/components/ui/native-select"
import { api } from "@/lib/api/client"
import { call, fieldErrors, messageFor } from "@/lib/api/problem"
import type { Post } from "@/lib/api/types"

import { PostChecklist } from "./post-checklist"

// US-04 criterion 1: an invitation valid 72 hours, the campus email domain rule does not apply
export function InviteDialog({ posts }: { posts: Post[] }) {
  const t = useTranslations("admin")
  const ts = useTranslations("status.role")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const [email, setEmail] = useState("")
  const [name, setName] = useState("")
  const [role, setRole] = useState<"officer" | "admin">("officer")
  const [postIds, setPostIds] = useState<string[]>([])
  const [errors, setErrors] = useState<Record<string, string>>({})
  const invite = useMutation({
    mutationFn: () => call(api.POST("/admin/users/invitations", { body: { email: email.trim(), full_name: name.trim(), role, post_ids: postIds } })),
    onSuccess: (user) => {
      toast.success(t("invited", { email: user.email }))
      setOpen(false)
      setEmail("")
      setName("")
      setPostIds([])
      router.refresh()
    },
    onError: (err) => setErrors(fieldErrors(err)),
  })
  function submit() {
    const local: Record<string, string> = {}
    if (!email.trim()) local.email = te("required")
    if (name.trim().length < 2) local.full_name = te("required")
    if (role === "officer" && postIds.length === 0) local.post_ids = t("postsRequired")
    setErrors(local)
    if (Object.keys(local).length === 0) {
      invite.mutate()
    }
  }
  const general = invite.error && Object.keys(fieldErrors(invite.error)).length === 0 ? messageFor(invite.error, te) : null
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger render={<Button />}>
        <UserPlus aria-hidden />
        {t("invite")}
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("inviteTitle")}</DialogTitle>
          <DialogDescription>{t("inviteLead")}</DialogDescription>
        </DialogHeader>
        <form
          id="invite-form"
          noValidate
          className="flex flex-col gap-4"
          onSubmit={(e) => {
            e.preventDefault()
            submit()
          }}
        >
          <FormAlert>{general}</FormAlert>
          <Field id="invite-email" label={t("inviteEmail")} error={errors.email}>
            {(aria) => <Input {...aria} type="email" value={email} onChange={(e) => setEmail(e.target.value)} autoComplete="off" />}
          </Field>
          <Field id="invite-name" label={t("inviteName")} error={errors.full_name}>
            {(aria) => <Input {...aria} value={name} onChange={(e) => setName(e.target.value)} autoComplete="off" />}
          </Field>
          <Field id="invite-role" label={t("role")} error={errors.role}>
            {(aria) => (
              <NativeSelect {...aria} value={role} onChange={(e) => setRole(e.target.value === "admin" ? "admin" : "officer")}>
                <option value="officer">{ts("officer")}</option>
                <option value="admin">{ts("admin")}</option>
              </NativeSelect>
            )}
          </Field>
          <Field
            id="invite-posts"
            as="fieldset"
            label={t("posts")}
            hint={t("postsHint")}
            optionalLabel={role === "admin" ? tc("optional") : undefined}
            error={errors.post_ids}
          >
            {(aria) => <PostChecklist posts={posts} value={postIds} onChange={setPostIds} describedBy={aria["aria-describedby"]} />}
          </Field>
        </form>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            {tc("cancel")}
          </Button>
          <Button type="submit" form="invite-form" disabled={invite.isPending}>
            {t("sendInvite")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
