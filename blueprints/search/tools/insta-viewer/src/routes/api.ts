import { Hono } from 'hono'
import { cors } from 'hono/cors'
import type { HonoEnv } from '../types'
import { InstagramClient } from '../instagram'
import { SessionManager } from '../session'
import { Cache } from '../cache'
import {
  parseProfileResponse, parsePostsResult, parsePostDetail, parseComments, parseCommentReplies,
  parseSearchResults, parseHashtagPosts, parseLocationPosts, parseStories,
  parseReels, parseFollowList, parseHighlights,
} from '../parse'
import {
  CACHE_PROFILE, CACHE_POSTS, CACHE_POST, CACHE_COMMENTS, CACHE_SEARCH,
  CACHE_HASHTAG, CACHE_LOCATION, CACHE_STORIES, CACHE_REELS, CACHE_FOLLOW, CACHE_HIGHLIGHTS,
  docIdProfilePosts, docIdReels, qhComments, qhCommentReplies, qhHashtag, qhLocation,
  qhFollowers, qhFollowing, qhHighlights,
} from '../config'

const app = new Hono<HonoEnv>()
app.use('*', cors())

function sm(c: any) { return new SessionManager(c.env) }
async function ig(c: any) { return new SessionManager(c.env).getClient() }
function kv(c: any) { return new Cache(c.env.KV) }

// GET /api/profile/:username
app.get('/profile/:username', async (c) => {
  const username = c.req.param('username')
  const client = await ig(c)
  const cache = kv(c)

  const key = `profile:${username.toLowerCase()}`
  let result = await cache.get<any>(key)
  if (!result) {
    const data = await client.getProfileInfo(username)
    result = parseProfileResponse(data)
    if (result) await cache.set(key, result, CACHE_PROFILE)
  }
  if (!result) return c.json({ error: 'User not found' }, 404)
  return c.json(result)
})

// GET /api/posts/:username?cursor=
app.get('/posts/:username', async (c) => {
  const username = c.req.param('username')
  const cursor = c.req.query('cursor') || ''
  const client = await ig(c)
  const cache = kv(c)

  // Get profile for ID
  const profileKey = `profile:${username.toLowerCase()}`
  let profileData = await cache.get<any>(profileKey)
  if (!profileData) {
    const data = await client.getProfileInfo(username)
    profileData = parseProfileResponse(data)
    if (profileData) await cache.set(profileKey, profileData, CACHE_PROFILE)
  }
  if (!profileData?.profile) return c.json({ error: 'User not found' }, 404)

  if (!cursor && profileData.posts) {
    return c.json({ posts: profileData.posts, cursor: profileData.cursor, hasMore: profileData.hasMore })
  }

  const postsKey = `posts:${username.toLowerCase()}:${cursor}`
  let postsData = await cache.get<any>(postsKey)
  if (!postsData) {
    const vars: Record<string, unknown> = { data: { count: 12 }, username }
    if (cursor) (vars.data as any).max_id = cursor
    const data = await client.graphqlPost(docIdProfilePosts, vars)
    postsData = parsePostsResult(data, username)
    await cache.set(postsKey, postsData, CACHE_POSTS)
  }
  return c.json(postsData)
})

// GET /api/post/:shortcode
app.get('/post/:shortcode', async (c) => {
  const shortcode = c.req.param('shortcode')
  const client = await ig(c)
  const cache = kv(c)

  const key = `post:${shortcode}`
  let post = await cache.get<any>(key)
  if (!post) {
    const data = await client.getPostDetail(shortcode)
    post = parsePostDetail(data)
    if (post) await cache.set(key, post, CACHE_POST)
  }
  if (!post) return c.json({ error: 'Post not found' }, 404)
  return c.json({ post })
})

// GET /api/comments/:shortcode?cursor=
app.get('/comments/:shortcode', async (c) => {
  const shortcode = c.req.param('shortcode')
  const cursor = c.req.query('cursor') || ''
  const client = await ig(c)
  const cache = kv(c)

  const key = `comments:${shortcode}:${cursor}`
  let data = await cache.get<any>(key)
  if (!data) {
    const vars: Record<string, unknown> = { shortcode, first: 24 }
    if (cursor) vars.after = cursor
    const resp = await client.graphqlQuery(qhComments, vars)
    data = parseComments(resp)
    await cache.set(key, data, CACHE_COMMENTS)
  }
  return c.json(data)
})

// GET /api/comments/:shortcode/replies/:commentId
app.get('/comments/:shortcode/replies/:commentId', async (c) => {
  const commentId = c.req.param('commentId')
  const cursor = c.req.query('cursor') || ''
  const client = await ig(c)
  const cache = kv(c)

  const key = `replies:${commentId}:${cursor}`
  let data = await cache.get<any>(key)
  if (!data) {
    const vars: Record<string, unknown> = { comment_id: commentId, first: 12 }
    if (cursor) vars.after = cursor
    const resp = await client.graphqlQuery(qhCommentReplies, vars)
    data = parseCommentReplies(resp)
    await cache.set(key, data, CACHE_COMMENTS)
  }
  return c.json(data)
})

// GET /api/search?q=
app.get('/search', async (c) => {
  const query = c.req.query('q') || ''
  if (!query) return c.json({ error: 'Query required' }, 400)

  const client = await ig(c)
  const cache = kv(c)

  const key = `search:${query.toLowerCase()}`
  let result = await cache.get<any>(key)
  if (!result) {
    const data = await client.search(query)
    result = parseSearchResults(data)
    await cache.set(key, result, CACHE_SEARCH)
  }
  return c.json(result)
})

// GET /api/hashtag/:tag?cursor=
app.get('/hashtag/:tag', async (c) => {
  const tag = c.req.param('tag')
  const cursor = c.req.query('cursor') || ''
  const client = await ig(c)
  const cache = kv(c)

  const key = `hashtag:${tag.toLowerCase()}:${cursor}`
  let data = await cache.get<any>(key)
  if (!data) {
    const vars: Record<string, unknown> = { tag_name: tag, first: 12 }
    if (cursor) vars.after = cursor
    const resp = await client.graphqlQuery(qhHashtag, vars)
    data = parseHashtagPosts(resp)
    await cache.set(key, data, CACHE_HASHTAG)
  }
  return c.json(data)
})

// GET /api/location/:id?cursor=
app.get('/location/:id', async (c) => {
  const locationId = c.req.param('id')
  const cursor = c.req.query('cursor') || ''
  const client = await ig(c)
  const cache = kv(c)

  const key = `location:${locationId}:${cursor}`
  let data = await cache.get<any>(key)
  if (!data) {
    const vars: Record<string, unknown> = { id: locationId, first: 12 }
    if (cursor) vars.after = cursor
    const resp = await client.graphqlQuery(qhLocation, vars)
    data = parseLocationPosts(resp)
    await cache.set(key, data, CACHE_LOCATION)
  }
  return c.json(data)
})

// GET /api/stories/:username
app.get('/stories/:username', async (c) => {
  const username = c.req.param('username')
  const client = await ig(c)
  const cache = kv(c)

  // Get profile for user ID
  const profileKey = `profile:${username.toLowerCase()}`
  let profileData = await cache.get<any>(profileKey)
  if (!profileData) {
    const data = await client.getProfileInfo(username)
    profileData = parseProfileResponse(data)
    if (profileData) await cache.set(profileKey, profileData, CACHE_PROFILE)
  }
  if (!profileData?.profile) return c.json({ error: 'User not found' }, 404)

  const storiesKey = `stories:${username.toLowerCase()}`
  let items = await cache.get<any[]>(storiesKey)
  if (!items) {
    const data = await client.getStories(profileData.profile.id)
    items = parseStories(data)
    await cache.set(storiesKey, items, CACHE_STORIES)
  }
  return c.json({ username, items })
})

// GET /api/reels/:username?cursor=
app.get('/reels/:username', async (c) => {
  const username = c.req.param('username')
  const cursor = c.req.query('cursor') || ''
  const client = await ig(c)
  const cache = kv(c)

  const profileKey = `profile:${username.toLowerCase()}`
  let profileData = await cache.get<any>(profileKey)
  if (!profileData) {
    const data = await client.getProfileInfo(username)
    profileData = parseProfileResponse(data)
    if (profileData) await cache.set(profileKey, profileData, CACHE_PROFILE)
  }
  if (!profileData?.profile) return c.json({ error: 'User not found' }, 404)

  const key = `reels:${username.toLowerCase()}:${cursor}`
  let data = await cache.get<any>(key)
  if (!data) {
    const vars: Record<string, unknown> = { data: { target_user_id: profileData.profile.id, page_size: 12 } }
    if (cursor) (vars.data as any).max_id = cursor
    const resp = await client.graphqlPost(docIdReels, vars)
    data = parseReels(resp)
    await cache.set(key, data, CACHE_REELS)
  }
  return c.json(data)
})

// GET /api/followers/:username?cursor=
app.get('/followers/:username', async (c) => {
  return handleFollow(c, 'followers')
})

// GET /api/following/:username?cursor=
app.get('/following/:username', async (c) => {
  return handleFollow(c, 'following')
})

async function handleFollow(c: any, type: 'followers' | 'following') {
  const username = c.req.param('username')
  const cursor = c.req.query('cursor') || ''
  const client = await ig(c)
  const cache = kv(c)

  const profileKey = `profile:${username.toLowerCase()}`
  let profileData = await cache.get<any>(profileKey)
  if (!profileData) {
    const data = await client.getProfileInfo(username)
    profileData = parseProfileResponse(data)
    if (profileData) await cache.set(profileKey, profileData, CACHE_PROFILE)
  }
  if (!profileData?.profile) return c.json({ error: 'User not found' }, 404)

  const qh = type === 'followers' ? qhFollowers : qhFollowing
  const key = `${type}:${username.toLowerCase()}:${cursor}`
  let data = await cache.get<any>(key)
  if (!data) {
    const vars: Record<string, unknown> = { id: profileData.profile.id, first: 24 }
    if (cursor) vars.after = cursor
    const resp = await client.graphqlQuery(qh, vars)
    data = parseFollowList(resp)
    await cache.set(key, data, CACHE_FOLLOW)
  }
  return c.json(data)
}

// GET /api/highlights/:username
app.get('/highlights/:username', async (c) => {
  const username = c.req.param('username')
  const client = await ig(c)
  const cache = kv(c)

  const profileKey = `profile:${username.toLowerCase()}`
  let profileData = await cache.get<any>(profileKey)
  if (!profileData) {
    const data = await client.getProfileInfo(username)
    profileData = parseProfileResponse(data)
    if (profileData) await cache.set(profileKey, profileData, CACHE_PROFILE)
  }
  if (!profileData?.profile) return c.json({ error: 'User not found' }, 404)

  const key = `highlights:${username.toLowerCase()}`
  let data = await cache.get<any[]>(key)
  if (!data) {
    const resp = await client.graphqlQuery(qhHighlights, { user_id: profileData.profile.id, include_highlight_reels: true })
    data = parseHighlights(resp)
    await cache.set(key, data, CACHE_HIGHLIGHTS)
  }
  return c.json({ highlights: data })
})

// JSON error handler for all API routes
app.onError(async (err, c) => {
  const msg = err.message || String(err)
  if (msg.includes('rate limited')) return c.json({ error: 'Rate limited' }, 429)
  if (InstagramClient.isSessionError(err)) {
    // Try to refresh session for next request
    try { await sm(c).refreshSession() } catch { /* best effort */ }
    return c.json({ error: msg, sessionRefreshed: true }, 503)
  }
  if (msg.includes('not found')) return c.json({ error: msg }, 404)
  return c.json({ error: msg }, 500)
})

export default app
