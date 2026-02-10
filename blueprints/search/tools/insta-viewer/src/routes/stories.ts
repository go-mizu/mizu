import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { InstagramClient } from '../instagram'
import { Cache } from '../cache'
import { parseProfileResponse, parseStories, parseHighlights } from '../parse'
import { renderLayout, renderStoriesViewer, renderError } from '../html'
import { CACHE_STORIES, CACHE_PROFILE } from '../config'

const app = new Hono<HonoEnv>()

// User stories
app.get('/:username', async (c) => {
  const username = c.req.param('username')
  if (username === 'highlights') return c.notFound()
  const client = new InstagramClient(c.env.INSTA_SESSION_ID, c.env.INSTA_CSRF_TOKEN, c.env.INSTA_DS_USER_ID, c.env.INSTA_MID, c.env.INSTA_IG_DID)
  const cache = new Cache(c.env.KV)

  try {
    // Get profile for user ID and avatar
    const profileKey = `profile:${username.toLowerCase()}`
    let profileData = await cache.get<any>(profileKey)
    if (!profileData) {
      const data = await client.getProfileInfo(username)
      profileData = parseProfileResponse(data)
      if (profileData) await cache.set(profileKey, profileData, CACHE_PROFILE)
    }
    if (!profileData?.profile) return c.html(renderError('User not found', `@${username} doesn't exist.`), 404)

    const profile = profileData.profile

    // Get stories
    const storiesKey = `stories:${username.toLowerCase()}`
    let items = await cache.get<any[]>(storiesKey)
    if (!items) {
      const data = await client.getStories(profile.id)
      items = parseStories(data)
      await cache.set(storiesKey, items, CACHE_STORIES)
    }

    const content = renderStoriesViewer(username, items || [], profile.profilePicUrl)
    return c.html(renderLayout(`${username}'s Story`, content, { isStory: true }))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (msg.includes('rate limited')) return c.html(renderError('Rate Limited', 'Too many requests.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
})

// Highlight stories
app.get('/highlights/:id', async (c) => {
  const highlightId = c.req.param('id')
  const client = new InstagramClient(c.env.INSTA_SESSION_ID, c.env.INSTA_CSRF_TOKEN, c.env.INSTA_DS_USER_ID, c.env.INSTA_MID, c.env.INSTA_IG_DID)
  const cache = new Cache(c.env.KV)

  try {
    const cacheKey = `highlight:${highlightId}`
    let items = await cache.get<any[]>(cacheKey)
    if (!items) {
      const fullId = highlightId.startsWith('highlight:') ? highlightId : `highlight:${highlightId}`
      const data = await client.getHighlightItems(fullId)
      items = parseStories(data)
      await cache.set(cacheKey, items, CACHE_STORIES)
    }

    const username = (items && items.length > 0) ? items[0].ownerUsername || '' : ''
    const content = renderStoriesViewer(username, items || [], '')
    return c.html(renderLayout('Highlight', content, { isStory: true }))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    return c.html(renderError('Error', msg), 500)
  }
})

export default app
