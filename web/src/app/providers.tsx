"use client"

import { MutationCache, QueryCache, QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { useTranslations } from "next-intl"
import { useState, type ReactNode } from "react"

import { Toaster } from "@/components/ui/sonner"
import { ApiError } from "@/lib/api/problem"

// an expired session sends the user to sign in and the action is not retried (US-02 criterion 4)
function onExpired(err: unknown) {
  if (err instanceof ApiError && err.status === 401 && !window.location.pathname.startsWith("/masuk")) {
    const next = encodeURIComponent(window.location.pathname + window.location.search)
    // a full navigation drops every cached query of the expired session
    // eslint-disable-next-line @next/next/no-location-assign-relative-destination
    window.location.assign(`/masuk?next=${next}&expired=1`)
  }
}

function retry(count: number, err: unknown): boolean {
  if (err instanceof ApiError && err.status < 500 && err.status !== 429) {
    return false
  }
  return count < 2
}

export function Providers({ children }: { children: ReactNode }) {
  const t = useTranslations("common")
  const [client] = useState(
    () =>
      new QueryClient({
        queryCache: new QueryCache({ onError: onExpired }),
        mutationCache: new MutationCache({ onError: onExpired }),
        defaultOptions: {
          queries: { staleTime: 30_000, retry, refetchOnWindowFocus: false },
          mutations: { retry: false },
        },
      }),
  )
  return (
    <QueryClientProvider client={client}>
      {children}
      <Toaster position="top-center" closeButton containerAriaLabel={t("notifications")} />
    </QueryClientProvider>
  )
}
