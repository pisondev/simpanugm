import { PageTransition, Reveal } from "@/components/motion/primitives"

export function LegalPage({ title, version, sections }: { title: string; version?: string; sections: { h: string; p: string }[] }) {
  return (
    <PageTransition>
      <article className="mx-auto flex max-w-3xl flex-col gap-6">
        <Reveal>
          <header className="flex flex-col gap-1">
            <h1 className="text-3xl font-semibold tracking-tight text-balance">{title}</h1>
            {version ? <p className="text-sm text-muted-foreground">{version}</p> : null}
          </header>
        </Reveal>
        <div className="grid gap-3 md:grid-cols-2">
          {sections.map((s) => (
            <section key={s.h} className="surface flex flex-col gap-2 p-5">
              <h2 className="text-base font-semibold">{s.h}</h2>
              <p className="text-sm leading-6 text-muted-foreground">{s.p}</p>
            </section>
          ))}
        </div>
      </article>
    </PageTransition>
  )
}
