import { describe, expect, it } from "vitest"

import { isOpenNow } from "@/lib/hours"
import { claimPath, codePattern, homeFor, reportPath, safeNext } from "@/lib/paths"
import { pick, pickOne, queryString, uuidPattern } from "@/lib/query"

describe("safeNext", () => {
  it("keeps same origin paths", () => {
    expect(safeNext("/laporan/hilang/LST-2611-0007", "/beranda")).toBe("/laporan/hilang/LST-2611-0007")
  })

  it("refuses anything that could leave the site", () => {
    for (const bad of ["//evil.example", "https://evil.example", "/\\evil.example", "beranda", "", undefined]) {
      expect(safeNext(bad, "/beranda")).toBe("/beranda")
    }
  })
})

describe("codes and homes", () => {
  it("recognises report codes in any case", () => {
    expect(codePattern.test("FND-2610-0042")).toBe(true)
    expect(codePattern.test("lst-2611-0007")).toBe(true)
    expect(codePattern.test("dompet hitam")).toBe(false)
  })

  it("routes a code to the right record view", () => {
    expect(reportPath("fnd-2610-0042", true)).toBe("/petugas/laporan/temuan/FND-2610-0042")
    expect(reportPath("LST-2611-0007", false)).toBe("/laporan/hilang/LST-2611-0007")
  })

  it("routes a claim code to the right view", () => {
    expect(claimPath("CLM-2611-0001", true)).toBe("/petugas/klaim/CLM-2611-0001")
    expect(claimPath("CLM-2611-0001", false)).toBe("/klaim/CLM-2611-0001")
  })

  it("sends each role to its home", () => {
    expect(homeFor("member")).toBe("/beranda")
    expect(homeFor("officer")).toBe("/petugas")
    expect(homeFor("admin")).toBe("/admin")
  })
})

describe("query params", () => {
  it("passes only well formed values", () => {
    expect(pick("0190f1a2-0000-7000-8000-000000000000", uuidPattern)).toBeDefined()
    expect(pick("'; drop table", uuidPattern)).toBeUndefined()
    expect(pick(["a", "b"])).toBeUndefined()
    expect(pickOne("lost", ["lost", "found"] as const)).toBe("lost")
    expect(pickOne("other", ["lost", "found"] as const)).toBeUndefined()
  })

  it("drops empty values from a query string", () => {
    expect(queryString({ q: "dompet", type: undefined }, { cursor: "abc" })).toBe("?q=dompet&cursor=abc")
    expect(queryString({})).toBe("")
  })
})

describe("opening hours", () => {
  const hours = { mon: [["08:00", "16:00"]], sat: [] }

  it("is open inside a range in WIB", () => {
    expect(isOpenNow(hours, new Date("2026-11-02T02:00:00Z"))).toBe(true)
  })

  it("is closed at the end minute and on a day without ranges", () => {
    expect(isOpenNow(hours, new Date("2026-11-02T09:00:00Z"))).toBe(false)
    expect(isOpenNow(hours, new Date("2026-11-07T02:00:00Z"))).toBe(false)
  })
})
