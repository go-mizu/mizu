import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { InstagramClient } from '../instagram'
import { Cache } from '../cache'
import { parseProfileResponse, parsePostsResult, parseHighlights, parseTaggedPosts } from '../parse'
import { renderLayout, renderProfileHeader, renderPostGrid, renderHighlights, renderPagination, renderPrivateMessage, renderError } from '../html'
import { CACHE_PROFILE, CACHE_POSTS, CACHE_HIGHLIGHTS, docIdProfilePosts, docIdProfilePostsAnon, qhHighlights, qhTagged } from '../config'
import type { Profile, ProfileWithPosts } from '../types'

const app = new Hono<HonoEnv>()

function ig(c: { env: { INSTA_SESSION_ID: string; INSTA_CSRF_TOKEN: string; INSTA_DS_USER_ID: string; INSTA_MID: string; INSTA_IG_DID: string } }) {
  return new InstagramClient(c.env.INSTA_SESSION_ID, c.env.INSTA_CSRF_TOKEN, c.env.INSTA_DS_USER_ID, c.env.INSTA_MID, c.env.INSTA_IG_DID)
}

// Reels tab
app.get('/:username/reels', async (c) => {
  const username = c.req.param('username')
  const cursor = c.req.query('cursor') || ''
  const client = ig(c)
  const cache = new Cache(c.env.KV)

  try {
    const { profile } = await getProfile(client, cache, username)
    if (!profile) return c.html(renderError('User not found', `@${username} doesn't exist.`), 404)

    const { renderReelsGrid } = await import('./reels')
    const { docIdReels, CACHE_REELS } = await import('../config')
    const { parseReels } = await import('../parse')

    const cacheKey = `reels:${username.toLowerCase()}:${cursor}`
    let reelsData = await cache.get<{ reels: unknown[]; cursor: string }>(cacheKey)
    if (!reelsData) {
      const vars: Record<string, unknown> = { data: { target_user_id: profile.id, page_size: 12 } }
      if (cursor) (vars.data as Record<string, unknown>).max_id = cursor
      const data = await client.graphqlPost(docIdReels, vars)
      const result = parseReels(data)
      reelsData = { reels: result.reels, cursor: result.cursor }
      await cache.set(cacheKey, reelsData, CACHE_REELS)
    }

    const tabs = renderTabs(username, 'reels')
    const grid = renderReelsGrid(reelsData.reels as any)
    const pagination = renderPagination(reelsData.cursor as string, `/${username}/reels`)

    let content = renderProfileHeader(profile) + tabs + grid + pagination
    return c.html(renderLayout(`@${profile.username}`, content))
  } catch (e) {
    return handleError(c, e)
  }
})

// Tagged tab
app.get('/:username/tagged', async (c) => {
  const username = c.req.param('username')
  const cursor = c.req.query('cursor') || ''
  const client = ig(c)
  const cache = new Cache(c.env.KV)

  try {
    const { profile } = await getProfile(client, cache, username)
    if (!profile) return c.html(renderError('User not found', `@${username} doesn't exist.`), 404)

    const cacheKey = `tagged:${username.toLowerCase()}:${cursor}`
    let taggedData = await cache.get<{ posts: unknown[]; cursor: string }>(cacheKey)
    if (!taggedData) {
      const vars: Record<string, unknown> = { id: profile.id, first: 12 }
      if (cursor) vars.after = cursor
      const data = await client.graphqlQuery(qhTagged, vars)
      const result = parseTaggedPosts(data)
      taggedData = { posts: result.posts, cursor: result.cursor }
      await cache.set(cacheKey, taggedData, CACHE_POSTS)
    }

    const tabs = renderTabs(username, 'tagged')
    const grid = renderPostGrid(taggedData.posts as any)
    const pagination = renderPagination(taggedData.cursor as string, `/${username}/tagged`)

    let content = renderProfileHeader(profile) + tabs + grid + pagination
    return c.html(renderLayout(`@${profile.username}`, content))
  } catch (e) {
    return handleError(c, e)
  }
})

// Main profile (posts grid)
app.get('/:username', async (c) => {
  const username = c.req.param('username')
  if (username === 'favicon.ico' || username === 'robots.txt' || username === 's' || username === 'api' || username === 'search' || username === 'explore' || username === 'p' || username === 'stories' || username === 'reels') return c.notFound()

  const cursor = c.req.query('cursor') || ''
  const client = ig(c)
  const cache = new Cache(c.env.KV)

  try {
    const { profile, initialPosts, initialCursor, initialHasMore } = await getProfile(client, cache, username)
    if (!profile) return c.html(renderError('User not found', `@${username} doesn't exist or may have been suspended.`), 404)

    if (profile.isPrivate) {
      const content = renderProfileHeader(profile) + renderPrivateMessage()
      return c.html(renderLayout(`@${profile.username}`, content))
    }

    // Fetch highlights
    let highlights: any[] = []
    const hlKey = `highlights:${username.toLowerCase()}`
    let hlData = await cache.get<any[]>(hlKey)
    if (!hlData) {
      try {
        const hlResp = await client.graphqlQuery(qhHighlights, { user_id: profile.id, include_highlight_reels: true })
        hlData = parseHighlights(hlResp)
        await cache.set(hlKey, hlData, CACHE_HIGHLIGHTS)
      } catch { hlData = [] }
    }
    highlights = hlData || []

    let posts: any[]
    let nextCursor: string

    if (cursor) {
      // Paginated: fetch next page
      const cacheKey = `posts:${username.toLowerCase()}:${cursor}`
      let postsData = await cache.get<{ posts: unknown[]; cursor: string }>(cacheKey)
      if (!postsData) {
        const vars: Record<string, unknown> = { data: { count: 12, include_relationship_info: false, latest_besties_reel_media: true, latest_reel_media: true }, username }
        ;(vars.data as Record<string, unknown>).max_id = cursor
        const data = await client.graphqlPost(docIdProfilePosts, vars)
        const result = parsePostsResult(data, profile.username, profile.profilePicUrl)
        postsData = { posts: result.posts, cursor: result.cursor }
        await cache.set(cacheKey, postsData, CACHE_POSTS)
      }
      posts = postsData.posts as any[]
      nextCursor = postsData.cursor as string
    } else {
      // First page: use data from profile response
      posts = initialPosts || []
      nextCursor = initialCursor || ''
    }

    const tabs = renderTabs(username, 'posts')
    const grid = renderPostGrid(posts)
    const pagination = renderPagination(nextCursor, `/${username}`)

    let content = ''
    if (!cursor) {
      content = renderProfileHeader(profile) + renderHighlights(highlights) + tabs + grid + pagination
    } else {
      content = grid + pagination
    }

    return c.html(renderLayout(`@${profile.username}`, content))
  } catch (e) {
    return handleError(c, e)
  }
})

// ── Helpers ──

async function getProfile(client: InstagramClient, cache: Cache, username: string): Promise<{ profile: Profile | null; initialPosts?: any[]; initialCursor?: string; initialHasMore?: boolean }> {
  const profileKey = `profile:${username.toLowerCase()}`
  const cached = await cache.get<ProfileWithPosts>(profileKey)
  if (cached) return { profile: cached.profile, initialPosts: cached.posts, initialCursor: cached.cursor, initialHasMore: cached.hasMore }

  const data = await client.getProfileInfo(username)
  const result = parseProfileResponse(data)
  if (!result) return { profile: null }

  await cache.set(profileKey, result, CACHE_PROFILE)
  return { profile: result.profile, initialPosts: result.posts, initialCursor: result.cursor, initialHasMore: result.hasMore }
}

function renderTabs(username: string, active: 'posts' | 'reels' | 'tagged'): string {
  const gridIcon = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></svg>'
  const reelsIcon = '<svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2.982c2.937 0 3.285.011 4.445.064a6.087 6.087 0 012.042.379 3.408 3.408 0 011.265.823c.371.371.655.803.823 1.265a6.087 6.087 0 01.379 2.042c.053 1.16.064 1.508.064 4.445s-.011 3.285-.064 4.445a6.087 6.087 0 01-.379 2.042 3.643 3.643 0 01-2.088 2.088 6.087 6.087 0 01-2.042.379c-1.16.053-1.508.064-4.445.064s-3.285-.011-4.445-.064a6.087 6.087 0 01-2.042-.379 3.643 3.643 0 01-2.088-2.088 6.087 6.087 0 01-.379-2.042c-.053-1.16-.064-1.508-.064-4.445s.011-3.285.064-4.445a6.087 6.087 0 01.379-2.042 3.408 3.408 0 01.823-1.265 3.408 3.408 0 011.265-.823 6.087 6.087 0 012.042-.379c1.16-.053 1.508-.064 4.445-.064M12 1c-2.987 0-3.362.013-4.535.066a8.074 8.074 0 00-2.67.511 5.392 5.392 0 00-1.949 1.27 5.392 5.392 0 00-1.269 1.948 8.074 8.074 0 00-.51 2.67C1.012 8.638 1 9.013 1 12s.013 3.362.066 4.535a8.074 8.074 0 00.511 2.67 5.625 5.625 0 003.218 3.218 8.074 8.074 0 002.67.51C8.638 22.988 9.013 23 12 23s3.362-.013 4.535-.066a8.074 8.074 0 002.67-.511 5.625 5.625 0 003.218-3.218 8.074 8.074 0 00.51-2.67C22.988 15.362 23 14.987 23 12s-.013-3.362-.066-4.535a8.074 8.074 0 00-.511-2.67 5.392 5.392 0 00-1.27-1.949 5.392 5.392 0 00-1.948-1.269 8.074 8.074 0 00-2.67-.51C15.362 1.012 14.987 1 12 1z"/><path d="M10 7.757l6 4.243-6 4.243z"/></svg>'
  const taggedIcon = '<svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round"><path d="M20.59 13.41l-7.17 7.17a2 2 0 01-2.83 0L2 12V2h10l8.59 8.59a2 2 0 010 2.82z"/><line x1="7" y1="7" x2="7.01" y2="7"/></svg>'

  return `<div class="tabs"><a href="/${username}" class="${active === 'posts' ? 'active' : ''}">${gridIcon} POSTS</a><a href="/${username}/reels" class="${active === 'reels' ? 'active' : ''}">${reelsIcon} REELS</a><a href="/${username}/tagged" class="${active === 'tagged' ? 'active' : ''}">${taggedIcon} TAGGED</a></div>`
}

function handleError(c: any, e: unknown): Response {
  const msg = e instanceof Error ? e.message : String(e)
  if (msg.includes('rate limited')) return c.html(renderError('Rate Limited', 'Too many requests. Please try again later.'), 429)
  if (msg.includes('Session locked') || msg.includes('Session expired')) return c.html(renderError('Session Error', msg), 503)
  if (msg.includes('not found')) return c.html(renderError('Not Found', msg), 404)
  return c.html(renderError('Error', msg), 500)
}

export default app
export { getProfile, renderTabs }
function proxyImg(url: string): string {
  if (!url) return ''
  if (url.includes('.cdninstagram.com/') || url.includes('.fbcdn.net/')) {
    return '/img/' + encodeURIComponent(url)
  }
  return url
}

export function renderReelsGrid(reels: any[]): string {
  let h = '<div class="post-grid">'
  for (const r of reels) {
    const url = `/reels/${r.shortcode || r.id}`
    h += `<a href="${url}" class="post-grid-item"><img src="${proxyImg(r.displayUrl || '')}" alt="" loading="lazy"><div class="post-grid-overlay"><span class="post-grid-stat"><svg viewBox="0 0 24 24" fill="#fff" width="19" height="19"><path d="M8 5v14l11-7z"/></svg> ${r.playCount ? fmtNum(r.playCount) : fmtNum(r.viewCount || 0)}</span></div><span class="post-grid-badge"><svg viewBox="0 0 24 24" fill="#fff" width="20" height="20"><path d="M10 7.757l6 4.243-6 4.243z"/></svg></span></a>`
  }
  h += '</div>'
  return h
}

function fmtNum(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1).replace(/\.0$/, '') + 'M'
  if (n >= 10_000) return (n / 1_000).toFixed(0) + 'K'
  if (n >= 1_000) return (n / 1_000).toFixed(1).replace(/\.0$/, '') + 'K'
  return n.toLocaleString()
}
