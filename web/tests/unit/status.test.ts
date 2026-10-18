import { readFileSync } from "node:fs"
import { resolve } from "node:path"

import { describe, expect, it } from "vitest"

import messages from "@/messages/id.json"
import { custodyStatus, foundStatus, lostStatus } from "@/lib/status"

// vitest runs from web/, the contract lives next door in api/
const spec = readFileSync(resolve(process.cwd(), "../api/openapi.yaml"), "utf8")

function specEnum(name: string): string[] {
  const match = new RegExp(`\\n    ${name}:\\s*\\n\\s*type: string\\s*\\n\\s*enum: \\[([^\\]]+)\\]`).exec(spec)
  if (!match?.[1]) {
    throw new Error(`enum ${name} not found in api/openapi.yaml`)
  }
  return match[1].split(",").map((s) => s.trim())
}

// ui-guidelines section 4: every enum value has an icon, a tone and a label, checked against the contract itself
describe("status vocabulary", () => {
  const cases = [
    { name: "LostStatus", map: lostStatus, labels: messages.status.lost },
    { name: "FoundStatus", map: foundStatus, labels: messages.status.found },
    { name: "CustodyStatus", map: custodyStatus, labels: messages.status.custody },
  ]
  for (const c of cases) {
    it(`covers every ${c.name} value`, () => {
      const values = specEnum(c.name)
      expect(values.length).toBeGreaterThan(3)
      expect(Object.keys(c.map).sort()).toEqual([...values].sort())
      expect(Object.keys(c.labels).sort()).toEqual([...values].sort())
    })
  }

  it("labels every sensitivity, role and account state", () => {
    expect(Object.keys(messages.status.sensitivity).sort()).toEqual(specEnum("Sensitivity").sort())
    expect(Object.keys(messages.status.role).sort()).toEqual(specEnum("Role").sort())
    expect(Object.keys(messages.status.account).sort()).toEqual(specEnum("AccountStatus").sort())
  })
})
