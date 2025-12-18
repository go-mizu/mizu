import { WordPress } from "@acme/wordpress"

const wp = new WordPress({
  baseUrl: "https://example.com",
  authToken: process.env.WP_TOKEN!,
})

// list posts with query params
const posts = await wp.posts.list({ search: "contract", per_page: 10 })

// retrieve by id
const post = await wp.posts.retrieve({ id: posts[0].id })

// create a post
const created = await wp.posts.create({
  title: "Hello",
  content: "This is generated from a contract-driven SDK",
  status: "draft",
})

// update a post
const updated = await wp.posts.update({
  id: created.id,
  status: "publish",
})

// delete a post
await wp.posts.delete({ id: created.id, force: true })

// pages
const pages = await wp.pages.list({ per_page: 5 })

// media
const media = await wp.media.list({ per_page: 5 })

// current user
const me = await wp.users.me()
console.log(me.id, me.name)
