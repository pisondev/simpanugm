import "server-only"

import { connection } from "next/server"

// the time of the request, read after connection() so the page is never prerendered with a stale clock
export async function requestTime(): Promise<Date> {
  await connection()
  return new Date()
}
