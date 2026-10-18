import { CircleCheck, TriangleAlert } from "lucide-react"
import type { ReactNode } from "react"

import { Alert, AlertDescription } from "@/components/ui/alert"

export function FormAlert({ tone = "error", children }: { tone?: "error" | "success"; children: ReactNode }) {
  if (!children) {
    return null
  }
  return (
    <Alert variant={tone === "error" ? "destructive" : "default"} className={tone === "success" ? "border-success/30 bg-success/5" : undefined}>
      {tone === "error" ? <TriangleAlert aria-hidden /> : <CircleCheck className="text-success" aria-hidden />}
      <AlertDescription className={tone === "success" ? "text-foreground" : undefined}>{children}</AlertDescription>
    </Alert>
  )
}
