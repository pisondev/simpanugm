import { getRequestConfig } from "next-intl/server"

export default getRequestConfig(async () => ({
  locale: "id",
  timeZone: "Asia/Jakarta",
  messages: (await import("../messages/id.json")).default,
}))
