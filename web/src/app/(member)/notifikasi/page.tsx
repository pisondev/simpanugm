import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { PageHeader } from "@/components/layout/page-header"
import { NotificationList } from "@/components/notifications/notification-list"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("notifications")
  return { title: t("title") }
}

// US-35 and FR-NOTIF-06, every item opens the page where the user acts
export default async function NotificationsPage() {
  const t = await getTranslations("notifications")
  return (
    <div className="mx-auto max-w-2xl">
      <PageHeader title={t("title")} />
      <NotificationList />
    </div>
  )
}
