import { execFileSync } from "node:child_process"

import AxeBuilder from "@axe-core/playwright"
import { expect, type Page } from "@playwright/test"

const mailpit = process.env.E2E_MAILPIT_URL ?? "http://localhost:8025"

// NFR-USE-03: WCAG 2.1 AA on every page a journey visits
export async function expectAccessible(page: Page) {
  const result = await new AxeBuilder({ page }).withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"]).analyze()
  expect(result.violations.map((v) => `${v.id}: ${v.nodes.map((n) => n.target.join(" ")).join(", ")}`)).toEqual([])
}

type Summary = { ID: string }

// every email sent by the local stack lands in Mailpit, the link is read from the plain text part
export async function latestLink(to: string, path: string): Promise<string> {
  const pattern = new RegExp(`https?://[^\\s"]+${path}\\?token=[A-Za-z0-9_-]+`)
  for (let attempt = 0; attempt < 60; attempt++) {
    const list = (await (await fetch(`${mailpit}/api/v1/search?query=${encodeURIComponent(`to:"${to}"`)}`)).json()) as { messages: Summary[] }
    for (const m of list.messages) {
      const message = (await (await fetch(`${mailpit}/api/v1/message/${m.ID}`)).json()) as { Text: string }
      const match = pattern.exec(message.Text)
      if (match) {
        return match[0]
      }
    }
    await new Promise((r) => setTimeout(r, 500))
  }
  throw new Error(`no email with ${path} for ${to}`)
}

export function pathOf(link: string): string {
  const url = new URL(link)
  return url.pathname + url.search
}

export function uniqueEmail(prefix: string, domain = "mail.ugm.ac.id"): string {
  return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1e6)}@${domain}`
}

const compose = (
  process.env.E2E_COMPOSE ?? "docker compose -f ../deploy/compose.yaml -f ../deploy/compose.dev.yaml --env-file ../deploy/env/compose.env"
).split(" ")

// the first administrator comes from the command line, exactly as on a fresh server (FR-AUTH-09)
export function bootstrapAdmin(email: string, name: string): string {
  const [bin = "docker", ...args] = compose
  const out = execFileSync(bin, [...args, "run", "--rm", "-T", "api", "admin", "create", "--email", email, "--name", name], { encoding: "utf8" })
  const link = /https?:\/\/\S+\/undangan\?token=\S+/.exec(out)
  if (!link) {
    throw new Error(`admin create printed no link: ${out}`)
  }
  return link[0]
}

export async function registerMember(page: Page, name: string, email: string, password: string) {
  await page.goto("/daftar")
  await page.getByLabel("Nama lengkap").fill(name)
  await page.getByLabel("Email UGM").fill(email)
  await page.getByLabel("Kata sandi", { exact: true }).fill(password)
  await page.getByRole("checkbox").check()
  await page.getByRole("button", { name: "Buat akun" }).click()
  await expect(page.getByRole("heading", { name: "Periksa emailmu" })).toBeVisible()
  await page.goto(pathOf(await latestLink(email, "/verifikasi")))
  await expect(page.getByRole("heading", { name: "Email terverifikasi" })).toBeVisible()
}

export async function signIn(page: Page, email: string, password: string) {
  await page.goto("/masuk")
  await page.getByLabel("Email UGM").fill(email)
  await page.getByLabel("Kata sandi", { exact: true }).fill(password)
  await page.getByRole("main").getByRole("button", { name: "Masuk", exact: true }).click()
}

export async function setPassword(page: Page, link: string, password: string, submit: string) {
  await page.goto(pathOf(link))
  await page.getByLabel("Kata sandi baru", { exact: true }).fill(password)
  await page.getByLabel("Ulangi kata sandi baru").fill(password)
  await page.getByRole("button", { name: submit }).click()
}

// a plain photo drawn in the browser, large enough for the 320 px rule of BR-MED-01
export async function photo(page: Page): Promise<{ name: string; mimeType: string; buffer: Buffer }> {
  const data = await page.evaluate(() => {
    const canvas = document.createElement("canvas")
    canvas.width = 800
    canvas.height = 600
    const ctx = canvas.getContext("2d")
    if (!ctx) {
      throw new Error("no canvas")
    }
    ctx.fillStyle = "#1E3A8A"
    ctx.fillRect(0, 0, 800, 600)
    ctx.fillStyle = "#DBEAFE"
    ctx.fillRect(300, 120, 200, 360)
    return canvas.toDataURL("image/jpeg", 0.9)
  })
  return { name: "tumbler.jpg", mimeType: "image/jpeg", buffer: Buffer.from(data.split(",")[1] ?? "", "base64") }
}
