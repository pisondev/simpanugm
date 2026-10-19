import type { LucideIcon } from "lucide-react"
import type { ReactNode } from "react"

// information-architecture 4: an icon, one sentence on why it is empty, one action
export function EmptyState({ icon: Icon, children, action }: { icon: LucideIcon; children: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex flex-col items-center gap-3 rounded-xl border border-dashed bg-card px-4 py-10 text-center">
      <span className="relative my-2 flex size-16 items-center justify-center rounded-2xl border bg-surface-inset text-primary">
        <span aria-hidden className="absolute -top-2 -right-2 size-5 rounded-lg border bg-card" />
        <Icon className="size-7" aria-hidden />
        <span aria-hidden className="absolute -bottom-1 -left-3 size-3 rounded-full bg-primary-soft" />
      </span>
      <p className="max-w-sm text-muted-foreground">{children}</p>
      {action}
    </div>
  )
}
