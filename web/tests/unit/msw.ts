import { setupServer } from "msw/node"

export const server = setupServer()

export const apiBase = "http://localhost:3000/api/v1"
