import type { CSSProperties, ReactNode } from "react"

import { cn } from "@/lib/utils"

export type FieldAria = { id: string; "aria-describedby"?: string; "aria-invalid"?: boolean }

type Props = {
  id: string
  label: ReactNode
  hint?: ReactNode
  error?: string
  optionalLabel?: string
  className?: string
  suggested?: boolean
  motionIndex?: number
  as?: "div" | "fieldset"
  children: (aria: FieldAria) => ReactNode
}

// label, hint and error wired with aria-describedby, errors announced politely (ui-guidelines 7)
export function Field({ id, label, hint, error, optionalLabel, className, suggested = false, motionIndex = 0, as = "div", children }: Props) {
  const hintId = hint ? `${id}-hint` : undefined
  const errorId = error ? `${id}-error` : undefined
  const describedBy = [hintId, errorId].filter(Boolean).join(" ") || undefined
  const Wrapper = as
  const heading = (
    <>
      {label}
      {optionalLabel ? <span className="ml-1 font-normal text-muted-foreground">({optionalLabel})</span> : null}
    </>
  )
  return (
    <Wrapper
      className={cn("flex flex-col gap-2", suggested && "suggested-field isolate", className)}
      style={{ "--motion-index": motionIndex } as CSSProperties}
      aria-describedby={as === "fieldset" ? describedBy : undefined}
    >
      {as === "fieldset" ? (
        <legend className="mb-2 text-sm font-medium">{heading}</legend>
      ) : (
        <label htmlFor={id} className="text-sm font-medium">
          {heading}
        </label>
      )}
      {children({ id, "aria-describedby": describedBy, "aria-invalid": error ? true : undefined })}
      {hint ? (
        <p id={hintId} className="text-sm text-muted-foreground">
          {hint}
        </p>
      ) : null}
      <p id={errorId} aria-live="polite" className={cn("text-sm text-destructive", !error && "sr-only")}>
        {error ?? ""}
      </p>
    </Wrapper>
  )
}
