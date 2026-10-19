import type { ReactNode } from "react"

export function ComparisonPanel({ left, right, leftTitle, rightTitle }: { left: ReactNode; right: ReactNode; leftTitle: string; rightTitle: string }) {
  return (
    <div className="grid items-start gap-4 lg:grid-cols-2">
      {[
        { title: leftTitle, content: left },
        { title: rightTitle, content: right },
      ].map((side) => (
        <section key={side.title} aria-label={side.title} className="surface min-w-0 overflow-hidden">
          <h2 className="border-b bg-surface-inset px-5 py-3 text-sm font-semibold">{side.title}</h2>
          <div className="flex flex-col gap-4 p-5">{side.content}</div>
        </section>
      ))}
    </div>
  )
}
