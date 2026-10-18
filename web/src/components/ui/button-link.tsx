import type { VariantProps } from "class-variance-authority"
import Link from "next/link"
import type { ComponentProps } from "react"

import { buttonVariants } from "@/components/ui/button"
import { cn } from "@/lib/utils"

// navigation that looks like a button stays a link for assistive technology, never an anchor with role=button
function ButtonLink({ className, variant, size, ...props }: ComponentProps<typeof Link> & VariantProps<typeof buttonVariants>) {
  return <Link data-slot="button" className={cn(buttonVariants({ variant, size }), className)} {...props} />
}

export { ButtonLink }
