import { Phone } from "lucide-react"
import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"

import { OpenNow } from "@/components/report/post-hours"
import { serverApi } from "@/lib/api/server"
import type { Post } from "@/lib/api/types"
import { formatRanges, rangesFor, weekdays } from "@/lib/hours"

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations("help")
  return { title: t("title") }
}

async function posts(): Promise<Post[]> {
  try {
    const api = await serverApi()
    const { data } = await api.GET("/posts")
    return data?.items ?? []
  } catch {
    return []
  }
}

export default async function HelpPage() {
  const [t, items] = await Promise.all([getTranslations("help"), posts()])
  const faq = t.raw("faq") as { q: string; a: string }[]
  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-10">
      <header>
        <h1 className="text-2xl font-semibold tracking-tight">{t("title")}</h1>
        <p className="mt-1 text-muted-foreground">{t("lead")}</p>
      </header>
      <section aria-labelledby="pos" className="flex flex-col gap-4">
        <h2 id="pos" className="text-lg font-semibold">
          {t("postsTitle")}
        </h2>
        {items.length === 0 ? <p className="text-muted-foreground">{t("noPosts")}</p> : null}
        <ul className="grid gap-4 md:grid-cols-2">
          {items.map((post) => (
            <li key={post.id} className="flex flex-col gap-3 rounded-xl border bg-card p-4 shadow-sm">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h3 className="font-semibold">{post.name}</h3>
                  <p className="text-sm text-muted-foreground">
                    {post.location_name} | {post.zone_name}
                  </p>
                </div>
                <OpenNow hours={post.opening_hours} />
              </div>
              {post.phone ? (
                <a href={`tel:${post.phone}`} className="inline-flex min-h-11 items-center gap-2 text-sm text-primary">
                  <Phone className="size-4" aria-hidden />
                  {t("phone", { phone: post.phone })}
                </a>
              ) : null}
              <table className="w-full text-sm">
                <tbody>
                  {weekdays.map((d) => {
                    const ranges = rangesFor(post.opening_hours, d)
                    return (
                      <tr key={d} className="border-t first:border-t-0">
                        <th scope="row" className="py-1.5 pr-3 text-left font-normal text-muted-foreground">
                          {t(`weekdays.${d}`)}
                        </th>
                        <td className="py-1.5 text-right">{ranges.length > 0 ? formatRanges(ranges) : t("closed")}</td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </li>
          ))}
        </ul>
      </section>
      <section aria-labelledby="faq" className="flex flex-col gap-4">
        <h2 id="faq" className="text-lg font-semibold">
          {t("faqTitle")}
        </h2>
        <dl className="flex flex-col gap-4">
          {faq.map((f) => (
            <div key={f.q} className="rounded-xl border bg-card p-4">
              <dt className="font-medium">{f.q}</dt>
              <dd className="mt-1 text-muted-foreground">{f.a}</dd>
            </div>
          ))}
        </dl>
      </section>
    </div>
  )
}
