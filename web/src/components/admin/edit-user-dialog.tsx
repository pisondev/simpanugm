"use client"

import { useMutation } from "@tanstack/react-query"
import { Pencil } from "lucide-react"
import { useRouter } from "next/navigation"
import { useTranslations } from "next-intl"
import { useState } from "react"
import { toast } from "sonner"

import { FormAlert } from "@/components/auth/form-alert"
import { Field } from "@/components/report/form/field"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { NativeSelect } from "@/components/ui/native-select"
import { api } from "@/lib/api/client"
import { call, fieldErrors, messageFor } from "@/lib/api/problem"
import type { AdminUser, Post, Role } from "@/lib/api/types"

import { PostChecklist } from "./post-checklist"

const roles: Role[] = ["member", "officer", "admin"]

// US-04 criterion 3: a changed role or post list takes effect on the officer's next request
export function EditUserDialog({ user, posts }: { user: AdminUser; posts: Post[] }) {
  const t = useTranslations("admin")
  const ts = useTranslations("status.role")
  const tc = useTranslations("common")
  const te = useTranslations("errors")
  const router = useRouter()
  const [open, setOpen] = useState(false)
  const [role, setRole] = useState<Role>(user.role)
  const [postIds, setPostIds] = useState<string[]>(user.post_ids)
  const [confirmRole, setConfirmRole] = useState(false)
  const save = useMutation({
    mutationFn: () => call(api.PATCH("/admin/users/{id}", { params: { path: { id: user.id } }, body: { role, post_ids: role === "member" ? [] : postIds } })),
    onSuccess: () => {
      toast.success(t("saved"))
      setOpen(false)
      setConfirmRole(false)
      router.refresh()
    },
  })
  const roleChanged = role !== user.role
  const postsMissing = role === "officer" && postIds.length === 0
  const errors = save.error ? fieldErrors(save.error) : {}
  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        setOpen(v)
        if (v) {
          setRole(user.role)
          setPostIds(user.post_ids)
          setConfirmRole(false)
          save.reset()
        }
      }}
    >
      <DialogTrigger render={<Button variant="ghost" size="sm" aria-label={t("editTitle", { name: user.full_name })} />}>
        <Pencil aria-hidden />
        {t("edit")}
      </DialogTrigger>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("editTitle", { name: user.full_name })}</DialogTitle>
          <DialogDescription>{user.email}</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4">
          <FormAlert>{save.error && Object.keys(errors).length === 0 ? messageFor(save.error, te) : null}</FormAlert>
          <Field id={`role-${user.id}`} label={t("role")} error={errors.role}>
            {(aria) => (
              <NativeSelect
                {...aria}
                value={role}
                onChange={(e) => {
                  setRole(e.target.value as Role)
                  setConfirmRole(false)
                }}
              >
                {roles.map((r) => (
                  <option key={r} value={r}>
                    {ts(r)}
                  </option>
                ))}
              </NativeSelect>
            )}
          </Field>
          {role !== "member" ? (
            <Field id={`posts-${user.id}`} as="fieldset" label={t("posts")} hint={t("postsHint")} error={postsMissing ? t("postsRequired") : errors.post_ids}>
              {(aria) => <PostChecklist posts={posts} value={postIds} onChange={setPostIds} describedBy={aria["aria-describedby"]} />}
            </Field>
          ) : null}
          {confirmRole ? (
            <div className="rounded-lg border border-warning/30 bg-warning/5 p-3 text-sm">
              <p className="font-medium">{t("confirmRoleTitle", { name: user.full_name })}</p>
              <p className="text-muted-foreground">{t("confirmRoleBody")}</p>
            </div>
          ) : null}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)}>
            {tc("cancel")}
          </Button>
          <Button
            disabled={save.isPending || postsMissing}
            onClick={() => {
              if (roleChanged && !confirmRole) {
                setConfirmRole(true)
                return
              }
              save.mutate()
            }}
          >
            {roleChanged && confirmRole ? t("confirmRole") : tc("save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
