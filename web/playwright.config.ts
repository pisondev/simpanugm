import { defineConfig, devices } from "@playwright/test"

// journeys of testing-strategy section 4 against a running stack, make dev locally and the compose stack in CI
export default defineConfig({
  testDir: "./tests/e2e",
  timeout: 60_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://localhost:8000",
    locale: "id-ID",
    timezoneId: "Asia/Jakarta",
    trace: "retain-on-failure",
  },
  projects: [
    // the long journeys run once, on desktop, the member pages are covered on a phone by J1
    { name: "phone", use: { ...devices["Pixel 7"] }, testIgnore: /sprint1|claims/ },
    { name: "desktop", use: { ...devices["Desktop Chrome"] } },
  ],
})
