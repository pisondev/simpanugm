// readiness for traffic routing, checks that api /healthz is reachable
export async function GET() {
  const apiBase = process.env.API_BASE_URL || "http://api:8090"
  try {
    const res = await fetch(`${apiBase}/healthz`, {
      cache: "no-store",
      signal: AbortSignal.timeout(3000),
    })
    if (res.ok) {
      return new Response("ready", {
        status: 200,
        headers: { "Cache-Control": "no-store" },
      })
    }
    return new Response("api unhealthy", {
      status: 503,
      headers: { "Cache-Control": "no-store" },
    })
  } catch {
    return new Response("api unreachable", {
      status: 503,
      headers: { "Cache-Control": "no-store" },
    })
  }
}

export const dynamic = "force-dynamic"
