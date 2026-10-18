"use client"

import { MapPin, Plus, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { useDeferredValue, useState } from "react"

import { Button } from "@/components/ui/button"
import { Command, CommandEmpty, CommandInput, CommandItem, CommandList } from "@/components/ui/command"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { useLocations } from "@/lib/api/hooks"

export type PickedPlace = { location_id: string; name: string; zone_name?: string }

type Props = {
  id: string
  value: PickedPlace[]
  onChange: (places: PickedPlace[]) => void
  max: number
  describedBy?: string
  invalid?: boolean
  disabled?: boolean
}

// FR-MD-04: server search by name and alias, the zone shown under each name
export function LocationPicker({ id, value, onChange, max, describedBy, invalid, disabled }: Props) {
  const t = useTranslations("form.places")
  const [open, setOpen] = useState(false)
  const [q, setQ] = useState("")
  const deferred = useDeferredValue(q)
  const locations = useLocations(open ? deferred : "")
  const chosen = new Set(value.map((p) => p.location_id))
  const results = (locations.data?.items ?? []).filter((l) => !chosen.has(l.id))
  const single = max === 1

  function pick(place: PickedPlace) {
    onChange(single ? [place] : [...value, place].slice(0, max))
    setOpen(false)
    setQ("")
  }

  return (
    <div className="flex flex-col gap-2">
      {value.length > 0 ? (
        <ul className="flex flex-col gap-2">
          {value.map((p) => (
            <li key={p.location_id} className="flex items-center gap-3 rounded-lg border bg-card px-3 py-2">
              <MapPin className="size-5 shrink-0 text-primary" aria-hidden />
              <span className="min-w-0 flex-1">
                <span className="block truncate font-medium">{p.name}</span>
                {p.zone_name ? <span className="block text-sm text-muted-foreground">{t("zone", { name: p.zone_name })}</span> : null}
              </span>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                onClick={() => onChange(value.filter((x) => x.location_id !== p.location_id))}
                aria-label={t("remove", { name: p.name })}
                disabled={disabled}
              >
                <X aria-hidden />
              </Button>
            </li>
          ))}
        </ul>
      ) : null}
      {value.length < max || single ? (
        <Popover open={open} onOpenChange={setOpen}>
          <PopoverTrigger
            render={
              <Button
                id={id}
                type="button"
                variant="outline"
                aria-describedby={describedBy}
                aria-invalid={invalid}
                disabled={disabled}
                className="w-full justify-start font-normal text-muted-foreground"
              />
            }
          >
            <Plus aria-hidden />
            {single && value.length === 1 ? t("search") : value.length === 0 ? t("search") : t("add")}
          </PopoverTrigger>
          <PopoverContent className="w-(--anchor-width) min-w-72 p-0" align="start">
            <Command shouldFilter={false}>
              <CommandInput placeholder={t("search")} value={q} onValueChange={setQ} />
              <CommandList>
                {locations.isFetching && results.length === 0 ? null : <CommandEmpty>{t("empty")}</CommandEmpty>}
                {results.map((l) => (
                  <CommandItem key={l.id} value={l.id} onSelect={() => pick({ location_id: l.id, name: l.name, zone_name: l.zone_name })}>
                    <span className="flex min-w-0 flex-col">
                      <span className="truncate">{l.name}</span>
                      <span className="truncate text-sm text-muted-foreground">
                        {t("zone", { name: l.zone_name })}
                        {l.aliases.length > 0 ? ` | ${l.aliases.slice(0, 3).join(", ")}` : ""}
                      </span>
                    </span>
                  </CommandItem>
                ))}
              </CommandList>
            </Command>
          </PopoverContent>
        </Popover>
      ) : null}
    </div>
  )
}
