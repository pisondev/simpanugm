"use client"

import { useMutation } from "@tanstack/react-query"

import { api } from "@/lib/api/client"
import { call } from "@/lib/api/problem"

// a full navigation drops every cached query of the old session
export function useSignOut() {
  return useMutation({
    mutationFn: (all: boolean) => call(api.POST("/auth/logout", { params: { query: { all } } })),
    // eslint-disable-next-line @next/next/no-location-assign-relative-destination
    onSettled: () => window.location.assign("/masuk"),
  })
}
