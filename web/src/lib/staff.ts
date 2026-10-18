import "server-only"

import { load, serverApi } from "./api/server"
import type { Me, Post } from "./api/types"

// officers work on their assigned posts, an admin without assignments works on every post
export async function staffPosts(me: Me): Promise<Post[]> {
  const api = await serverApi()
  const all = (await load(api.GET("/posts"))).items
  if (me.role === "admin" && me.post_ids.length === 0) {
    return all
  }
  return all.filter((p) => me.post_ids.includes(p.id))
}
