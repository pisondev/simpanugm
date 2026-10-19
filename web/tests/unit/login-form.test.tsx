import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { http, HttpResponse } from "msw"
import { beforeEach, describe, expect, it, vi } from "vitest"

import { LoginForm } from "@/components/auth/login-form"

import { apiBase, server } from "./msw"
import { renderUi } from "./render"

const replace = vi.fn()
const refresh = vi.fn()

vi.mock("next/navigation", () => ({ useRouter: () => ({ replace, refresh, push: vi.fn() }) }))

const me = {
  id: "0190f1a2-0000-7000-8000-000000000001",
  email: "rani@mail.ugm.ac.id",
  full_name: "Rani",
  role: "officer",
  status: "active",
  email_verified: true,
  claim_restricted: false,
  post_ids: [],
  permissions: [],
}

describe("LoginForm, US-02", () => {
  beforeEach(() => {
    replace.mockReset()
    refresh.mockReset()
  })

  it("asks for both fields before calling api", async () => {
    renderUi(<LoginForm />)
    await userEvent.click(screen.getByRole("button", { name: "Masuk" }))
    expect(await screen.findAllByText("Wajib diisi")).toHaveLength(2)
  })

  it("shows the answer of api for a wrong password", async () => {
    server.use(
      http.post(`${apiBase}/auth/login`, () =>
        HttpResponse.json(
          { type: "about:blank", title: "Gagal masuk", status: 401, code: "invalid_credentials", detail: "Email atau kata sandi salah" },
          { status: 401, headers: { "Content-Type": "application/problem+json" } },
        ),
      ),
    )
    renderUi(<LoginForm />)
    await userEvent.type(screen.getByLabelText("Email UGM"), "rani@mail.ugm.ac.id")
    await userEvent.type(screen.getByLabelText("Kata sandi"), "salah sekali")
    await userEvent.click(screen.getByRole("button", { name: "Masuk" }))
    expect(await screen.findByText("Email atau kata sandi salah")).toBeInTheDocument()
    expect(replace).not.toHaveBeenCalled()
  })

  it("lands on the home of the role, or on a safe next path", async () => {
    server.use(http.post(`${apiBase}/auth/login`, () => HttpResponse.json(me)))
    renderUi(<LoginForm next="//evil.example" />)
    await userEvent.type(screen.getByLabelText("Email UGM"), "rani@mail.ugm.ac.id")
    await userEvent.type(screen.getByLabelText("Kata sandi"), "benar sekali")
    await userEvent.click(screen.getByRole("button", { name: "Masuk" }))
    await waitFor(() => expect(replace).toHaveBeenCalledWith("/petugas"))
  })

  it("allows quick login for demo actors only in demo mode", async () => {
    vi.stubEnv("NEXT_PUBLIC_DEMO_MODE", "true")
    server.use(http.post(`${apiBase}/auth/login`, () => HttpResponse.json(me)))
    renderUi(<LoginForm />)
    const budiButton = screen.getByRole("button", { name: /Penemu \(Mahasiswa\)/ })
    await userEvent.click(budiButton)
    await waitFor(() => expect(replace).toHaveBeenCalled())
    vi.unstubAllEnvs()
  })

  it("shows no demo accounts unless demo mode is on", () => {
    renderUi(<LoginForm />)
    expect(screen.queryByRole("button", { name: /Penemu \(Mahasiswa\)/ })).toBeNull()
  })
})
