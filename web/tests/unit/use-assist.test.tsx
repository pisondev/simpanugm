import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { act, renderHook, waitFor } from "@testing-library/react"
import { http, HttpResponse } from "msw"
import type { ReactNode } from "react"
import { describe, expect, it } from "vitest"

import { type AssistFields, useAssist } from "@/components/report/form/use-assist"

import { apiBase, server } from "./msw"

const tumbler = { id: "0190f1a2-0000-7000-8000-0000000000a1", slug: "tumbler", name: "Tumbler", parent_name: "Botol", sensitivity: "normal" }
const payung = { ...tumbler, id: "0190f1a2-0000-7000-8000-0000000000a2", slug: "payung", name: "Payung" }

const suggestions = {
  categories: [
    { category: tumbler, score: 0.7 },
    { category: payung, score: 0.2 },
  ],
  colours: ["biru"],
  brand: "Corkcicle",
}

function setup(initial: AssistFields, drafts = false) {
  const state = { ...initial }
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={client}>{children}</QueryClientProvider>
  const read = () => state
  const write = (patch: Partial<AssistFields>) => Object.assign(state, patch)
  const view = renderHook(() => useAssist({ enabled: true, reportType: "found", mediaId: "m1", read, write, drafts }), { wrapper })
  return { state, view }
}

const empty: AssistFields = { category_id: "", colours: [], brand: "", description: "" }

// US-23: suggestions are editable defaults that never overwrite what the reporter already typed
describe("useAssist", () => {
  it("fills empty fields and offers the other categories as alternatives", async () => {
    server.use(http.post(`${apiBase}/assist/analyze-image`, () => HttpResponse.json(suggestions)))
    const { state, view } = setup(empty)
    await waitFor(() => expect(view.result.current.status).toBe("ready"))
    expect(state).toMatchObject({ category_id: tumbler.id, colours: ["biru"], brand: "Corkcicle" })
    expect(view.result.current.suggested("brand", "Corkcicle")).toBe(true)
    expect(view.result.current.suggested("brand", "Lain")).toBe(false)
    view.rerender()
    expect(view.result.current.alternatives.map((a) => a.category.slug)).toEqual(["payung"])
  })

  it("keeps what the reporter already chose", async () => {
    server.use(http.post(`${apiBase}/assist/analyze-image`, () => HttpResponse.json(suggestions)))
    const { state, view } = setup({ ...empty, category_id: payung.id, brand: "Eiger" })
    await waitFor(() => expect(view.result.current.status).toBe("ready"))
    expect(state).toMatchObject({ category_id: payung.id, brand: "Eiger", colours: ["biru"] })
    expect(view.result.current.suggested("category", payung.id)).toBe(false)
  })

  // FR-ASSIST-02
  it("reports accepted, edited and ignored per field", async () => {
    server.use(http.post(`${apiBase}/assist/analyze-image`, () => HttpResponse.json(suggestions)))
    const { view } = setup(empty)
    await waitFor(() => expect(view.result.current.status).toBe("ready"))
    expect(view.result.current.outcome({ category_id: tumbler.id, colours: ["hijau"], brand: "", description: "" })).toEqual({
      category: "accepted",
      colours: "edited",
      brand: "ignored",
    })
  })

  it("drafts a description for an empty field only when asked to", async () => {
    server.use(
      http.post(`${apiBase}/assist/analyze-image`, () => HttpResponse.json(suggestions)),
      http.post(`${apiBase}/assist/describe`, () => HttpResponse.json({ description: "Tumbler berwarna biru, merek Corkcicle." })),
    )
    const { state, view } = setup(empty, true)
    await waitFor(() => expect(state.description).toBe("Tumbler berwarna biru, merek Corkcicle."))
    await waitFor(() => expect(view.result.current.outcome({ ...state })).toMatchObject({ description: "accepted" }))
  })

  // FR-ASSIST-04
  it("leaves the form alone when the assistant is unavailable", async () => {
    server.use(
      http.post(`${apiBase}/assist/analyze-image`, () =>
        HttpResponse.json({ code: "assist_unavailable", title: "x", status: 503, type: "x" }, { status: 503 }),
      ),
    )
    const { state, view } = setup(empty)
    await waitFor(() => expect(view.result.current.status).toBe("unavailable"))
    expect(state).toEqual(empty)
    expect(view.result.current.outcome(state)).toBeUndefined()
  })

  it("recovers from an unavailable model without overwriting manual entry", async () => {
    server.use(http.post(`${apiBase}/assist/analyze-image`, () => HttpResponse.json({}, { status: 503 })))
    const { state, view } = setup(empty)
    await waitFor(() => expect(view.result.current.status).toBe("unavailable"))
    state.brand = "Pilihan saya"
    server.use(http.post(`${apiBase}/assist/analyze-image`, () => HttpResponse.json(suggestions)))
    act(() => view.result.current.retry())
    await waitFor(() => expect(state.category_id).toBe(tumbler.id))
    expect(state.brand).toBe("Pilihan saya")
    expect(state.colours).toEqual(["biru"])
  })
})
