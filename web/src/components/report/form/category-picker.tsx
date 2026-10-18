"use client"

import { ChevronsUpDown } from "lucide-react"
import { useTranslations } from "next-intl"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import type { Category } from "@/lib/api/types"
import { cn } from "@/lib/utils"

export type Leaf = Category & { parentName: string }

export function leaves(tree: Category[]): Leaf[] {
  return tree.flatMap((parent) => (parent.children ?? []).map((leaf) => ({ ...leaf, parentName: parent.name })))
}

type Props = {
  tree: Category[]
  value: string
  onChange: (id: string) => void
  id: string
  describedBy?: string
  invalid?: boolean
}

export function CategoryPicker({ tree, value, onChange, id, describedBy, invalid }: Props) {
  const t = useTranslations("form.category")
  const [open, setOpen] = useState(false)
  const selected = leaves(tree).find((l) => l.id === value)
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            id={id}
            variant="outline"
            role="combobox"
            aria-expanded={open}
            aria-describedby={describedBy}
            aria-invalid={invalid}
            className={cn("w-full justify-between font-normal", !selected && "text-muted-foreground")}
          />
        }
      >
        <span className="truncate">{selected ? `${selected.name} (${selected.parentName})` : t("placeholder")}</span>
        <ChevronsUpDown className="opacity-50" aria-hidden />
      </PopoverTrigger>
      <PopoverContent className="w-(--anchor-width) min-w-72 p-0" align="start">
        <Command>
          <CommandInput placeholder={t("search")} />
          <CommandList>
            <CommandEmpty>{t("empty")}</CommandEmpty>
            {tree.map((parent) => (
              <CommandGroup key={parent.id} heading={parent.name}>
                {(parent.children ?? []).map((leaf) => (
                  <CommandItem
                    key={leaf.id}
                    value={leaf.id}
                    keywords={[leaf.name, parent.name, leaf.slug]}
                    data-checked={leaf.id === value}
                    onSelect={() => {
                      onChange(leaf.id)
                      setOpen(false)
                    }}
                  >
                    {leaf.name}
                  </CommandItem>
                ))}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}
