import { MemberShell } from "@/components/layout/member-shell"
import { StaffShell } from "@/components/layout/staff-shell"
import { VerifyBanner } from "@/components/layout/verify-banner"
import { requireMe } from "@/lib/session"

// staff reach their personal reports and profile here too, inside their own navigation
export default async function MemberLayout({ children }: LayoutProps<"/">) {
  const me = await requireMe()
  const user = { full_name: me.full_name, email: me.email, role: me.role }
  const body = (
    <>
      {me.email_verified ? null : <VerifyBanner />}
      {children}
    </>
  )
  return me.role === "member" ? <MemberShell me={user}>{body}</MemberShell> : <StaffShell me={user}>{body}</StaffShell>
}
