import type { Metadata } from "next"

import { SettingsForm } from "@/components/admin/settings-form"
import { PageHeader } from "@/components/layout/page-header"
import { load, serverApi } from "@/lib/api/server"

export async function generateMetadata(): Promise<Metadata> {
  return { title: "Pengaturan Sistem" }
}

export default async function SettingsPage() {
  const api = await serverApi()
  const settings = await load(api.GET("/admin/settings"))

  return (
    <div className="mx-auto flex max-w-5xl flex-col gap-6">
      <PageHeader title="Pengaturan Sistem" lead="Kelola konfigurasi runtime, batas waktu operasional, dan aturan bisnis (US-43)." />
      <SettingsForm initialItems={settings.items} />
    </div>
  )
}
