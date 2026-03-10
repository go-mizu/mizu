import { Hono } from 'hono'
import { cors } from 'hono/cors'
import type { HonoEnv } from '../types'
import { GraphQLClient } from '../graphql'
import { DB } from '../cache'
import { fetchTweetConversation } from '../tweet-fetch'
import {
  fetchProfileWithFallback, fetchUserTimelineWithFallback,
  fetchSearchTweetsWithFallback, fetchSearchUsersWithFallback,
} from '../fallback-fetch'
import {
  parseUserResult, parseFollowList, parseGraphList, parseListTimeline,
  parseListMembers,
} from '../parse'
import {
  gqlUserByScreenName, gqlFollowers, gqlFollowing,
  gqlListById, gqlListTweets, gqlListMembers,
  userFieldToggles,
  CACHE_PROFILE, CACHE_TIMELINE, CACHE_TWEET, CACHE_SEARCH, CACHE_FOLLOW, CACHE_LIST,
  SearchPeople,
} from '../config'

const now = () => Math.floor(Date.now() / 1000)

const app = new Hono<HonoEnv>()

// CORS must run before auth to handle preflight OPTIONS correctly
app.use('*', cors())

// Auth middleware: if API_TOKEN is set, require Bearer token
app.use('*', async (c, next) => {
  const token = c.env.API_TOKEN
  if (token) {
    const auth = c.req.header('Authorization') || ''
    if (auth !== `Bearer ${token}`) {
      return c.json({ error: 'Unauthorized' }, 401)
    }
  }
  await next()
})

function gql(c: any) {
  return new GraphQLClient(c.env.X_AUTH_TOKEN, c.env.X_CT0, c.env.X_BEARER_TOKEN)
}

// GET /api/profile/:username
app.get('/profile/:username', async (c) => {
  const username = c.req.param('username')
  const reload = c.req.query('reload') === '1'
  const db = new DB(c.env.DB)

  const cached = reload ? null : await db.getProfileWithMeta<any>(username)

  if (cached) {
    const age = now() - cached.fetchedAt
    // Background refresh if stale (>24h) but still within TTL grace
    if (age > 86400) {
      const refreshPromise = fetchProfileWithFallback(c.env, username).then(p => {
        if (p) return db.setProfile(username, p, CACHE_PROFILE)
      }).catch(() => {})
      ;(c as any).executionCtx?.waitUntil(refreshPromise)
    }
    return c.json({
      profile: cached.data,
      meta: { fromCache: true, age, cachedAt: cached.fetchedAt },
    })
  }

  const start = Date.now()
  const profile = await fetchProfileWithFallback(c.env, username)
  if (!profile) return c.json({ error: 'User not found' }, 404)
  await db.setProfile(username, profile, CACHE_PROFILE)
  return c.json({ profile, meta: { fromCache: false, duration: Date.now() - start } })
})

// GET /api/tweets/:username?tab=tweets|replies|media&cursor=
app.get('/tweets/:username', async (c) => {
  const username = c.req.param('username')
  const tab = c.req.query('tab') || 'tweets'
  const cursor = c.req.query('cursor') || ''
  const reload = c.req.query('reload') === '1'
  const db = new DB(c.env.DB)

  // Fetch profile (with meta for cache tracking, but we don't expose profile meta here)
  let profile = await db.getProfile<any>(username)
  if (!profile) {
    profile = await fetchProfileWithFallback(c.env, username)
    if (profile) await db.setProfile(username, profile, CACHE_PROFILE)
  }
  if (!profile) return c.json({ error: 'User not found' }, 404)

  const cachedTimeline = reload ? null : await db.getTimelineWithMeta<any>(username, tab, cursor)

  if (cachedTimeline) {
    const age = now() - cachedTimeline.fetchedAt
    return c.json({
      ...cachedTimeline.data,
      meta: { fromCache: true, age, cachedAt: cachedTimeline.fetchedAt },
    })
  }

  const start = Date.now()
  const result = await fetchUserTimelineWithFallback(c.env, username, tab, cursor, profile.id || '')
  const timelineData = { tweets: result.tweets, cursor: result.cursor }
  await db.setTimeline(username, tab, cursor, timelineData, CACHE_TIMELINE)
  return c.json({ ...timelineData, meta: { fromCache: false, duration: Date.now() - start } })
})

// GET /api/tweet/:id
app.get('/tweet/:id', async (c) => {
  const tweetID = c.req.param('id')
  const cursor = c.req.query('cursor') || ''
  const reload = c.req.query('reload') === '1'
  const db = new DB(c.env.DB)

  const cachedRow = reload ? null : await db.getTweetWithMeta<any>(tweetID, cursor)

  if (cachedRow) {
    const age = now() - cachedRow.fetchedAt
    const cached = cachedRow.data
    if (!cached.mainTweet) return c.json({ error: 'Tweet not found' }, 404)
    return c.json({
      tweet: cached.mainTweet,
      replies: cached.replies,
      cursor: cached.cursor,
      meta: { fromCache: true, age, cachedAt: cachedRow.fetchedAt },
    })
  }

  const start = Date.now()
  const result = await fetchTweetConversation(c.env, tweetID, cursor, false)

  let cached: any = null
  if (cursor && !result.mainTweet) {
    const firstPage = await db.getTweet<any>(tweetID, '')
    if (firstPage?.mainTweet) {
      cached = { mainTweet: firstPage.mainTweet, replies: result.replies, cursor: result.cursor }
    }
  }
  if (!cached && result.mainTweet) {
    cached = { mainTweet: result.mainTweet, replies: result.replies, cursor: result.cursor }
  }
  if (cached) await db.setTweet(tweetID, cursor, cached, CACHE_TWEET)

  if (!cached || !cached.mainTweet) return c.json({ error: 'Tweet not found' }, 404)
  return c.json({
    tweet: cached.mainTweet,
    replies: cached.replies,
    cursor: cached.cursor,
    meta: { fromCache: false, duration: Date.now() - start },
  })
})

// GET /api/search?q=&mode=Top|Latest|People|Photos&cursor=
app.get('/search', async (c) => {
  const query = c.req.query('q') || ''
  const mode = c.req.query('mode') || 'Top'
  const cursor = c.req.query('cursor') || ''
  const reload = c.req.query('reload') === '1'
  if (!query) return c.json({ error: 'Query required' }, 400)

  const db = new DB(c.env.DB)

  if (mode === SearchPeople) {
    const cachedRow = reload ? null : await db.getSearchWithMeta<any>(query, mode, cursor)
    if (cachedRow) {
      const age = now() - cachedRow.fetchedAt
      return c.json({
        ...cachedRow.data,
        meta: { fromCache: true, age, cachedAt: cachedRow.fetchedAt },
      })
    }
    const start = Date.now()
    const result = await fetchSearchUsersWithFallback(c.env, query, cursor)
    const usersData = { users: result.users, cursor: result.cursor }
    await db.setSearch(query, mode, cursor, usersData, CACHE_SEARCH)
    return c.json({ ...usersData, meta: { fromCache: false, duration: Date.now() - start } })
  } else {
    const cachedRow = reload ? null : await db.getSearchWithMeta<any>(query, mode, cursor)
    if (cachedRow) {
      const age = now() - cachedRow.fetchedAt
      return c.json({
        ...cachedRow.data,
        meta: { fromCache: true, age, cachedAt: cachedRow.fetchedAt },
      })
    }
    const start = Date.now()
    const result = await fetchSearchTweetsWithFallback(c.env, query, mode, cursor)
    const searchData = { tweets: result.tweets, cursor: result.cursor }
    await db.setSearch(query, mode, cursor, searchData, CACHE_SEARCH)
    return c.json({ ...searchData, meta: { fromCache: false, duration: Date.now() - start } })
  }
})

// GET /api/followers/:username?cursor=
app.get('/followers/:username', async (c) => handleFollow(c, 'followers'))
app.get('/following/:username', async (c) => handleFollow(c, 'following'))

async function handleFollow(c: any, type: 'followers' | 'following') {
  const username = c.req.param('username')
  const cursor = c.req.query('cursor') || ''
  const reload = c.req.query('reload') === '1'
  const client = gql(c)
  const db = new DB(c.env.DB)

  let profile = await db.getProfile<any>(username)
  if (!profile) {
    const data = await client.doGraphQL(gqlUserByScreenName, {
      screen_name: username, withSafetyModeUserFields: true,
    }, userFieldToggles)
    profile = parseUserResult(data)
    if (profile) await db.setProfile(username, profile, CACHE_PROFILE)
  }
  if (!profile) return c.json({ error: 'User not found' }, 404)

  const endpoint = type === 'followers' ? gqlFollowers : gqlFollowing
  const cachedRow = reload ? null : await db.getFollowWithMeta<any>(username, type, cursor)

  if (cachedRow) {
    const age = now() - cachedRow.fetchedAt
    return c.json({
      ...cachedRow.data,
      meta: { fromCache: true, age, cachedAt: cachedRow.fetchedAt },
    })
  }

  const start = Date.now()
  const vars: Record<string, unknown> = {
    userId: profile.id, count: 50, includePromotedContent: false,
  }
  if (cursor) vars.cursor = cursor
  const data = await client.doGraphQL(endpoint, vars, '')
  const result = parseFollowList(data)
  const followData = { users: result.users, cursor: result.cursor }
  await db.setFollow(username, type, cursor, followData, CACHE_FOLLOW)
  return c.json({ ...followData, meta: { fromCache: false, duration: Date.now() - start } })
}

// GET /api/list/:id?tab=tweets|members&cursor=
app.get('/list/:id', async (c) => {
  const listID = c.req.param('id')
  const tab = c.req.query('tab') || 'tweets'
  const cursor = c.req.query('cursor') || ''
  const reload = c.req.query('reload') === '1'
  const client = gql(c)
  const db = new DB(c.env.DB)

  let list = await db.getList<any>(listID)
  if (!list) {
    const data = await client.doGraphQL(gqlListById, { listId: listID }, '')
    list = parseGraphList(data)
    if (list) await db.setList(listID, list, CACHE_LIST)
  }
  if (!list) return c.json({ error: 'List not found' }, 404)

  if (tab === 'members') {
    const cachedRow = reload ? null : await db.getListContentWithMeta<any>(listID, 'members', cursor)
    if (cachedRow) {
      const age = now() - cachedRow.fetchedAt
      return c.json({
        list,
        ...cachedRow.data,
        meta: { fromCache: true, age, cachedAt: cachedRow.fetchedAt },
      })
    }
    const start = Date.now()
    const vars: Record<string, unknown> = { listId: listID, count: 200 }
    if (cursor) vars.cursor = cursor
    const data = await client.doGraphQL(gqlListMembers, vars, '')
    const result = parseListMembers(data)
    const membersData = { users: result.users, cursor: result.cursor }
    await db.setListContent(listID, 'members', cursor, membersData, CACHE_LIST)
    return c.json({ list, ...membersData, meta: { fromCache: false, duration: Date.now() - start } })
  } else {
    const cachedRow = reload ? null : await db.getListContentWithMeta<any>(listID, 'tweets', cursor)
    if (cachedRow) {
      const age = now() - cachedRow.fetchedAt
      return c.json({
        list,
        ...cachedRow.data,
        meta: { fromCache: true, age, cachedAt: cachedRow.fetchedAt },
      })
    }
    const start = Date.now()
    const vars: Record<string, unknown> = { rest_id: listID, count: 40 }
    if (cursor) vars.cursor = cursor
    const data = await client.doGraphQL(gqlListTweets, vars, '')
    const result = parseListTimeline(data)
    const tweetsData = { tweets: result.tweets, cursor: result.cursor }
    await db.setListContent(listID, 'tweets', cursor, tweetsData, CACHE_LIST)
    return c.json({ list, ...tweetsData, meta: { fromCache: false, duration: Date.now() - start } })
  }
})

export default app
