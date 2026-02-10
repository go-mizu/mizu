import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { InstagramClient } from '../instagram'
import { Cache } from '../cache'
import { parseProfileResponse, parseFollowList } from '../parse'
import { renderLayout, renderFollowPage, renderPagination, renderError } from '../html'
import { CACHE_PROFILE, CACHE_FOLLOW, qhFollowers, qhFollowing } from '../config'

const app = new Hono<HonoEnv>()

app.get('/:username/followers', async (c) => {
  return handleFollow(c, 'followers')
})

app.get('/:username/following', async (c) => {
  return handleFollow(c, 'following')
})

async function handleFollow(c: any, type: 'followers' | 'following') {
  const username = c.req.param('username')
  const cursor = c.req.query('cursor') || ''
  const client = new InstagramClient(c.env.INSTA_SESSION_ID, c.env.INSTA_CSRF_TOKEN, c.env.INSTA_DS_USER_ID, c.env.INSTA_MID, c.env.INSTA_IG_DID)
  const cache = new Cache(c.env.KV)

  try {
    // Get profile
    const profileKey = `profile:${username.toLowerCase()}`
    let profileData = await cache.get<any>(profileKey)
    if (!profileData) {
      const data = await client.getProfileInfo(username)
      profileData = parseProfileResponse(data)
      if (profileData) await cache.set(profileKey, profileData, CACHE_PROFILE)
    }
    if (!profileData?.profile) return c.html(renderError('User not found', `@${username} doesn't exist.`), 404)

    const profile = profileData.profile
    const qh = type === 'followers' ? qhFollowers : qhFollowing
    const cacheKey = `${type}:${username.toLowerCase()}:${cursor}`

    let followData = await cache.get<any>(cacheKey)
    if (!followData) {
      const vars: Record<string, unknown> = { id: profile.id, first: 24 }
      if (cursor) vars.after = cursor
      const data = await client.graphqlQuery(qh, vars)
      followData = parseFollowList(data)
      await cache.set(cacheKey, followData, CACHE_FOLLOW)
    }

    const content = renderFollowPage(username, followData.users || [], type)
      + renderPagination(followData.cursor || '', `/${username}/${type}`)

    return c.html(renderLayout(`${username}'s ${type}`, content))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (msg.includes('rate limited')) return c.html(renderError('Rate Limited', 'Too many requests.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
}

export default app
