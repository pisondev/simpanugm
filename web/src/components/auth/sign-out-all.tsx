"use client"

import { useTranslations } from "next-intl"

import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"

import { useSignOut } from "./use-sign-out"

// US-02 criterion 5: every session of the user is revoked
export function SignOutAll() {
  const t = useTranslations("profile")
  const tn = useTranslations("nav")
  const tc = useTranslations("common")
  const signOut = useSignOut()
  return (
    <AlertDialog>
      <AlertDialogTrigger render={<Button variant="destructive" className="w-fit" />}>{tn("signOutAll")}</AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("signOutAllTitle")}</AlertDialogTitle>
          <AlertDialogDescription>{t("signOutAllHint")}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{tc("cancel")}</AlertDialogCancel>
          <Button variant="destructive" onClick={() => signOut.mutate(true)} disabled={signOut.isPending}>
            {t("signOutAllConfirm")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
