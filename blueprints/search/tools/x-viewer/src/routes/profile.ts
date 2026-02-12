import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { Cache } from '../cache'
import { parseUserResult } from '../parse'
import { renderLayout, renderProfileHeader, renderTweetCard, renderMediaGrid, renderPagination, renderError } from '../html'
import {
  CACHE_PROFILE, CACHE_TIMELINE,
} from '../config'
import { fetchProfileWithFallback, fetchUserTimelineWithFallback } from '../fallback-fetch'
import { isRateLimitedError } from '../rate-limit'

const app = new Hono<HonoEnv>()

app.get('/:username', async (c) => {
  const username = c.req.param('username')
  if (username === 'favicon.ico' || username === 'robots.txt' || username === 's') return c.notFound()

  const cursor = c.req.query('cursor') || ''
  const tab = c.req.query('tab') || 'tweets'
  const cache = new Cache(c.env.KV)

  try {
    const profileKey = `profile:${username.toLowerCase()}`
    let profile = await cache.get<ReturnType<typeof parseUserResult>>(profileKey)
    if (!profile) {
      profile = await fetchProfileWithFallback(c.env, username)
      if (profile) await cache.set(profileKey, profile, CACHE_PROFILE)
    }

    if (!profile) {
      return c.html(renderError('User not found', `@${username} doesn't exist or may have been suspended.`), 404)
    }

    const cacheKey = `tweets:${username.toLowerCase()}:${tab}:${cursor}`
    let timelineData = await cache.get<{ tweets: unknown[]; cursor: string }>(cacheKey)

    if (!timelineData) {
      const result = await fetchUserTimelineWithFallback(c.env, username, tab, cursor, profile.id || '')
      timelineData = { tweets: result.tweets, cursor: result.cursor }
      await cache.set(cacheKey, timelineData, CACHE_TIMELINE)
    }

    const tweets = (timelineData.tweets || []) as Parameters<typeof renderTweetCard>[0][]
    const nextCursor = timelineData.cursor as string

    // Mark pinned tweets on the Posts tab (first page only)
    if (tab === 'tweets' && !cursor && profile.pinnedTweetIDs.length > 0) {
      const pinnedSet = new Set(profile.pinnedTweetIDs)
      for (const tweet of tweets) {
        if (pinnedSet.has(tweet.id)) tweet.isPin = true
      }
      // Sort pinned tweets to the top
      tweets.sort((a, b) => (a.isPin === b.isPin ? 0 : a.isPin ? -1 : 1))
    }

    let content = `<div class="sh"><h2>${profile.name}</h2></div>`
    content += renderProfileHeader(profile)

    const base = `/${profile.username}`
    content += `<div class="tabs"><a href="${base}" class="${tab === 'tweets' ? 'active' : ''}">Posts</a><a href="${base}?tab=replies" class="${tab === 'replies' ? 'active' : ''}">Replies</a><a href="${base}?tab=media" class="${tab === 'media' ? 'active' : ''}">Media</a></div>`

    if (tab === 'media') {
      content += renderMediaGrid(tweets)
    } else {
      for (const tweet of tweets) content += renderTweetCard(tweet)
    }
    content += renderPagination(nextCursor, `/${username}?tab=${tab}`)

    return c.html(renderLayout(`@${profile.username}`, content))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (isRateLimitedError(e)) return c.html(renderError('Rate Limited', 'Too many requests. Please try again later.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
})

export default app
