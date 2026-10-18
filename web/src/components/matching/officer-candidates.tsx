import Link from "next/link"
import { useTranslations } from "next-intl"

import { CodeChip } from "@/components/report/code-chip"
import { ButtonLink } from "@/components/ui/button-link"
import type { MatchCandidate, MatchScores } from "@/lib/api/types"

import { DismissDialog } from "./dismiss-dialog"
import { RematchButton } from "./match-actions"
import { percent, TierBadge } from "./tier-badge"

const columns = ["visual", "semantic", "attributes", "location", "time"] as const satisfies readonly (keyof MatchScores)[]

function num(v: number | null | undefined): string {
  return v === null || v === undefined ? "-" : v.toFixed(2)
}

type Props = {
  items: MatchCandidate[]
  // the side the officer is looking from, the table shows the other report
  side: "lost" | "found"
  reportId: string
  showLow: boolean
  toggleHref: string
}

// the review board view of FR-OFF-03: every component number, the counterpart and the officer actions
export function OfficerCandidates({ items, side, reportId, showLow, toggleHref }: Props) {
  const t = useTranslations("matching.officer")
  return (
    <section aria-labelledby="kandidat-petugas" className="flex flex-col gap-3 rounded-xl border bg-card p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 id="kandidat-petugas" className="font-semibold">
          {t("title")}
        </h2>
        <div className="flex flex-wrap gap-2">
          <ButtonLink href={toggleHref} variant="ghost" size="sm" scroll={false}>
            {showLow ? t("hideLow") : t("showLow")}
          </ButtonLink>
          <RematchButton type={side} id={reportId} />
        </div>
      </div>
      {items.length === 0 ? (
        <p className="text-muted-foreground">{t("empty")}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[44rem] text-sm">
            <thead>
              <tr className="border-b text-left text-muted-foreground">
                <th scope="col" className="py-2 pr-3 font-medium">
                  {side === "lost" ? t("found") : t("lost")}
                </th>
                <th scope="col" className="py-2 pr-3 font-medium">
                  {t("tier")}
                </th>
                <th scope="col" className="py-2 pr-3 text-right font-medium">
                  {t("overall")}
                </th>
                {columns.map((c) => (
                  <th key={c} scope="col" className="py-2 pr-3 text-right font-medium">
                    {t(`scores.${c}`)}
                  </th>
                ))}
                <th scope="col" className="py-2 pr-3 text-right font-medium">
                  {t("scores.category")}
                </th>
                <th scope="col" className="py-2 font-medium">
                  <span className="sr-only">{t("actions")}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {items.map((c) => {
                const other = side === "lost" ? c.found_report : c.lost_report
                const href = side === "lost" ? `/petugas/laporan/temuan/${other.code}` : `/petugas/laporan/hilang/${other.code}`
                return (
                  <tr key={c.id} className="border-b align-top last:border-0">
                    <td className="py-2 pr-3">
                      <Link href={href} className="flex flex-col gap-1 hover:underline">
                        <CodeChip code={other.code} />
                        <span>{other.item_name}</span>
                        {side === "found" && c.lost_report.owner_name ? <span className="text-muted-foreground">{c.lost_report.owner_name}</span> : null}
                      </Link>
                      {c.flags?.includes("name_hit") ? <span className="mt-1 inline-block rounded-full border px-2 text-xs">{t("nameHit")}</span> : null}
                    </td>
                    <td className="py-2 pr-3">
                      <TierBadge tier={c.tier} />
                    </td>
                    <td className="py-2 pr-3 text-right font-semibold tabular-nums">{percent(c.overall)}</td>
                    {columns.map((k) => (
                      <td key={k} className="py-2 pr-3 text-right tabular-nums">
                        {num(c.scores?.[k])}
                      </td>
                    ))}
                    <td className="py-2 pr-3 text-right tabular-nums">{num(c.scores?.category_factor)}</td>
                    <td className="py-2 text-right">{c.status === "active" ? <DismissDialog matchId={c.id} scope="officer" size="sm" /> : null}</td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}
