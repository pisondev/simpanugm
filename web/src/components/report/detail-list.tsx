import type { ReactNode } from "react"

export function DetailList({ children }: { children: ReactNode }) {
  return <dl className="grid gap-x-6 gap-y-4 sm:grid-cols-[minmax(8rem,auto)_1fr]">{children}</dl>
}

export function Detail({ label, children }: { label: ReactNode; children: ReactNode }) {
  return (
    <>
      <dt className="text-sm text-muted-foreground">{label}</dt>
      <dd className="-mt-3 break-words whitespace-pre-line sm:mt-0">{children}</dd>
    </>
  )
}
