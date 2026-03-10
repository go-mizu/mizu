import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { GraphQLClient } from '../graphql'
import { DB } from '../cache'
import { parseUserResult, parseFollowList } from '../parse'
import { renderLayout, renderFollowPage, renderError } from '../html'
import {
  gqlUserByScreenName, gqlFollowers, gqlFollowing,
  userFieldToggles, CACHE_PROFILE, CACHE_FOLLOW,
} from '../config'

const app = new Hono<HonoEnv>()

async function handleFollow(c: any, type: 'followers' | 'following') {
  const username = c.req.param('username')
  const cursor = c.req.query('cursor') || ''
  const gql = new GraphQLClient(c.env.X_AUTH_TOKEN, c.env.X_CT0, c.env.X_BEARER_TOKEN)
  const db = new DB(c.env.DB)

  try {
    let profile = await db.getProfile<ReturnType<typeof parseUserResult>>(username)
    if (!profile) {
      const data = await gql.doGraphQL(gqlUserByScreenName, {
        screen_name: username,
        withSafetyModeUserFields: true,
        withSuperFollowsUserFields: true,
      }, userFieldToggles)
      profile = parseUserResult(data)
      if (profile) await db.setProfile(username, profile, CACHE_PROFILE)
    }

    if (!profile) {
      return c.html(renderError('User not found', `@${username} doesn't exist or may have been suspended.`), 404)
    }

    const endpoint = type === 'followers' ? gqlFollowers : gqlFollowing
    let followData = await db.getFollow<{ users: unknown[]; cursor: string }>(username, type, cursor)

    if (!followData) {
      const vars: Record<string, unknown> = {
        userId: profile.id,
        count: 50,
        includePromotedContent: false,
      }
      if (cursor) vars.cursor = cursor
      const data = await gql.doGraphQL(endpoint, vars, '')
      const result = parseFollowList(data)
      followData = { users: result.users, cursor: result.cursor }
      await db.setFollow(username, type, cursor, followData, CACHE_FOLLOW)
    }

    const users = (followData.users || []) as Parameters<typeof renderFollowPage>[1]
    const nextCursor = followData.cursor as string

    const content = renderFollowPage(profile, users, type, nextCursor)
    const title = type === 'followers'
      ? `People following @${profile.username}`
      : `People @${profile.username} follows`
    return c.html(renderLayout(title, content))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (msg.includes('rate limited')) return c.html(renderError('Rate Limited', 'Too many requests. Please try again later.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
}

app.get('/:username/followers', (c) => handleFollow(c, 'followers'))
app.get('/:username/following', (c) => handleFollow(c, 'following'))

export default app
