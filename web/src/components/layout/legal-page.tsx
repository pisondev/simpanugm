export function LegalPage({ title, version, sections }: { title: string; version?: string; sections: { h: string; p: string }[] }) {
  return (
    <article className="mx-auto flex max-w-2xl flex-col gap-6">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        {version ? <p className="mt-1 text-sm text-muted-foreground">{version}</p> : null}
      </header>
      {sections.map((s) => (
        <section key={s.h} className="flex flex-col gap-2">
          <h2 className="text-lg font-semibold">{s.h}</h2>
          <p className="text-muted-foreground">{s.p}</p>
        </section>
      ))}
    </article>
  )
}
