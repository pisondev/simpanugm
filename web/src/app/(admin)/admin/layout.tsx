import { StaffShell } from "@/components/layout/staff-shell"
import { requireAdmin } from "@/lib/session"

export default async function AdminLayout({ children }: LayoutProps<"/admin">) {
  const me = await requireAdmin()
  return <StaffShell me={{ full_name: me.full_name, email: me.email, role: me.role }}>{children}</StaffShell>
}
