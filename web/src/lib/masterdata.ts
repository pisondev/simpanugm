import "server-only"

import { load, serverApi } from "./api/server"

export async function formData() {
  const api = await serverApi()
  const [categories, colours] = await Promise.all([load(api.GET("/categories")), load(api.GET("/colours"))])
  return { categories: categories.items, colours: colours.items }
}

export async function zonesAndCategories() {
  const api = await serverApi()
  const [categories, zones] = await Promise.all([load(api.GET("/categories")), load(api.GET("/zones"))])
  return { categories: categories.items, zones: zones.items }
}
