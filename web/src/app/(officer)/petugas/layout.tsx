import { StaffShell } from "@/components/layout/staff-shell"
import { requireStaff } from "@/lib/session"

export default async function OfficerLayout({ children }: LayoutProps<"/petugas">) {
  const me = await requireStaff()
  return <StaffShell me={{ full_name: me.full_name, email: me.email, role: me.role }}>{children}</StaffShell>
}
