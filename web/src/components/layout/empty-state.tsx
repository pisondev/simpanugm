import type { LucideIcon } from "lucide-react"
import type { ReactNode } from "react"

// information-architecture 4: an icon, one sentence on why it is empty, one action
export function EmptyState({ icon: Icon, children, action }: { icon: LucideIcon; children: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed bg-card px-4 py-10 text-center">
      <Icon className="size-8 text-muted-foreground" aria-hidden />
      <p className="max-w-sm text-muted-foreground">{children}</p>
      {action}
    </div>
  )
}
