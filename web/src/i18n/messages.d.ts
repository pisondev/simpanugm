import type messages from "../messages/id.json"

// a missing or misspelt message key fails the type check instead of rendering the raw key
declare module "next-intl" {
  interface AppConfig {
    Locale: "id"
    Messages: typeof messages
  }
}
