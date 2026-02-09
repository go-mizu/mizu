import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { GraphQLClient } from '../graphql'
import { Cache } from '../cache'
import { parseUserResult, parseTimeline } from '../parse'
import { renderLayout, renderProfileHeader, renderTweetCard, renderPagination, renderError } from '../html'
import {
  gqlUserByScreenName, gqlUserTweetsV2, gqlUserMedia,
  userFieldToggles, userTweetsFieldToggles,
  CACHE_PROFILE, CACHE_TIMELINE,
} from '../config'

const app = new Hono<HonoEnv>()

app.get('/:username', async (c) => {
  const username = c.req.param('username')
  if (username === 'favicon.ico' || username === 'robots.txt' || username === 's') return c.notFound()

  const cursor = c.req.query('cursor') || ''
  const tab = c.req.query('tab') || 'tweets'
  const gql = new GraphQLClient(c.env.X_AUTH_TOKEN, c.env.X_CT0, c.env.X_BEARER_TOKEN)
  const cache = new Cache(c.env.KV)

  try {
    const profileKey = `profile:${username.toLowerCase()}`
    let profile = await cache.get<ReturnType<typeof parseUserResult>>(profileKey)
    if (!profile) {
      const data = await gql.doGraphQL(gqlUserByScreenName, {
        screen_name: username,
        withSafetyModeUserFields: true,
        withSuperFollowsUserFields: true,
      }, userFieldToggles)
      profile = parseUserResult(data)
      if (profile) await cache.set(profileKey, profile, CACHE_PROFILE)
    }

    if (!profile) {
      return c.html(renderError('User not found', `@${username} doesn't exist or may have been suspended.`), 404)
    }

    const endpoint = tab === 'media' ? gqlUserMedia : gqlUserTweetsV2
    const toggles = tab === 'media' ? '' : userTweetsFieldToggles
    const cacheKey = `tweets:${username.toLowerCase()}:${tab}:${cursor}`
    let timelineData = await cache.get<{ tweets: unknown[]; cursor: string }>(cacheKey)

    if (!timelineData) {
      const vars: Record<string, unknown> = {
        userId: profile.id,
        count: 40,
        includePromotedContent: false,
        withQuickPromoteEligibilityTweetFields: true,
        withVoice: true,
      }
      if (cursor) vars.cursor = cursor
      const data = await gql.doGraphQL(endpoint, vars, toggles)
      const result = parseTimeline(data)
      timelineData = { tweets: result.tweets, cursor: result.cursor }
      await cache.set(cacheKey, timelineData, CACHE_TIMELINE)
    }

    const tweets = (timelineData.tweets || []) as Parameters<typeof renderTweetCard>[0][]
    const nextCursor = timelineData.cursor as string

    let content = `<div class="sh"><h2>${profile.name}</h2><div class="sh-sub">${profile.tweetsCount.toLocaleString()} posts</div></div>`
    content += renderProfileHeader(profile)

    const base = `/${profile.username}`
    content += `<div class="tabs"><a href="${base}" class="${tab === 'tweets' ? 'active' : ''}">Posts</a><a href="${base}?tab=replies" class="${tab === 'replies' ? 'active' : ''}">Replies</a><a href="${base}?tab=media" class="${tab === 'media' ? 'active' : ''}">Media</a></div>`

    for (const tweet of tweets) content += renderTweetCard(tweet)
    content += renderPagination(nextCursor, `/${username}?tab=${tab}`)

    return c.html(renderLayout(`@${profile.username}`, content))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (msg.includes('rate limited')) return c.html(renderError('Rate Limited', 'Too many requests. Please try again later.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
})

export default app
