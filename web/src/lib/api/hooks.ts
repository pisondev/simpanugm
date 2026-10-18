"use client"

import { keepPreviousData, useQuery } from "@tanstack/react-query"

import { api } from "./client"
import { call } from "./problem"

export function useLocations(q: string) {
  const query = q.trim()
  return useQuery({
    queryKey: ["locations", query],
    queryFn: () => call(api.GET("/locations", { params: { query: { q: query || undefined, limit: 10 } } })),
    placeholderData: keepPreviousData,
    staleTime: 5 * 60_000,
  })
}

export function usePosts(nearLocationId?: string) {
  return useQuery({
    queryKey: ["posts", nearLocationId ?? null],
    queryFn: () => call(api.GET("/posts", { params: { query: { near_location_id: nearLocationId } } })),
    staleTime: 5 * 60_000,
  })
}
