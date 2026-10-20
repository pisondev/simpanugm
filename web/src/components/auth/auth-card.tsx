import type { ReactNode } from "react"

import { PageTransition, Reveal } from "@/components/motion/primitives"

export function AuthCard({ title, lead, children, footer }: { title: string; lead?: ReactNode; children: ReactNode; footer?: ReactNode }) {
  return (
    <PageTransition className="mx-auto w-full max-w-md">
      <div className="auth-texture rounded-2xl p-3 sm:p-5">
        <div className="surface rounded-xl p-5 shadow-raised sm:p-7">
          <Reveal>
            <header className="flex flex-col gap-1">
              <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
              {lead ? <p className="text-sm leading-6 text-muted-foreground">{lead}</p> : null}
            </header>
          </Reveal>
          <div className="mt-6">{children}</div>
        </div>
      </div>
      {footer ? <div className="mt-4 text-center text-sm text-muted-foreground">{footer}</div> : null}
    </PageTransition>
  )
}
