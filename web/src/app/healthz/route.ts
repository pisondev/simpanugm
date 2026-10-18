// liveness for the container health check, it never calls api so a slow backend does not restart web
export function GET() {
  return new Response("ok", { headers: { "Cache-Control": "no-store" } })
}

export const dynamic = "force-dynamic"
