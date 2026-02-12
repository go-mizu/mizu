import { Hono } from 'hono'
import { cors } from 'hono/cors'
import type { HonoEnv } from '../types'
import { GraphQLClient } from '../graphql'
import { Cache } from '../cache'
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

const app = new Hono<HonoEnv>()

app.use('*', cors())

function gql(c: any) {
  return new GraphQLClient(c.env.X_AUTH_TOKEN, c.env.X_CT0, c.env.X_BEARER_TOKEN)
}
function kv(c: any) {
  return new Cache(c.env.KV)
}

// GET /api/profile/:username
app.get('/profile/:username', async (c) => {
  const username = c.req.param('username')
  const cache = kv(c)

  const profileKey = `profile:${username.toLowerCase()}`
  let profile = await cache.get<any>(profileKey)
  if (!profile) {
    profile = await fetchProfileWithFallback(c.env, username)
    if (profile) await cache.set(profileKey, profile, CACHE_PROFILE)
  }

  if (!profile) return c.json({ error: 'User not found' }, 404)
  return c.json({ profile })
})

// GET /api/tweets/:username?tab=tweets|replies|media&cursor=
app.get('/tweets/:username', async (c) => {
  const username = c.req.param('username')
  const tab = c.req.query('tab') || 'tweets'
  const cursor = c.req.query('cursor') || ''
  const cache = kv(c)

  // Get profile for user ID
  const profileKey = `profile:${username.toLowerCase()}`
  let profile = await cache.get<any>(profileKey)
  if (!profile) {
    profile = await fetchProfileWithFallback(c.env, username)
    if (profile) await cache.set(profileKey, profile, CACHE_PROFILE)
  }
  if (!profile) return c.json({ error: 'User not found' }, 404)

  const cacheKey = `tweets:${username.toLowerCase()}:${tab}:${cursor}`
  let timelineData = await cache.get<any>(cacheKey)
  if (!timelineData) {
    const result = await fetchUserTimelineWithFallback(c.env, username, tab, cursor, profile.id || '')
    timelineData = { tweets: result.tweets, cursor: result.cursor }
    await cache.set(cacheKey, timelineData, CACHE_TIMELINE)
  }

  return c.json(timelineData)
})

// GET /api/tweet/:id
app.get('/tweet/:id', async (c) => {
  const tweetID = c.req.param('id')
  const cursor = c.req.query('cursor') || ''
  const cache = kv(c)

  const cacheKey = cursor ? `tweet:${tweetID}:${cursor}` : `tweet:${tweetID}`
  let cached = await cache.get<any>(cacheKey)
  if (!cached) {
    const result = await fetchTweetConversation(c.env, tweetID, cursor, false)

    if (cursor && !result.mainTweet) {
      const firstPage = await cache.get<any>(`tweet:${tweetID}`)
      if (firstPage?.mainTweet) {
        cached = { mainTweet: firstPage.mainTweet, replies: result.replies, cursor: result.cursor }
      }
    }
    if (!cached && result.mainTweet) {
      cached = { mainTweet: result.mainTweet, replies: result.replies, cursor: result.cursor }
    }
    if (cached) await cache.set(cacheKey, cached, CACHE_TWEET)
  }

  if (!cached || !cached.mainTweet) return c.json({ error: 'Tweet not found' }, 404)
  return c.json({ tweet: cached.mainTweet, replies: cached.replies, cursor: cached.cursor })
})

// GET /api/search?q=&mode=Top|Latest|People|Photos&cursor=
app.get('/search', async (c) => {
  const query = c.req.query('q') || ''
  const mode = c.req.query('mode') || 'Top'
  const cursor = c.req.query('cursor') || ''
  if (!query) return c.json({ error: 'Query required' }, 400)

  const cache = kv(c)
  const cacheKey = `search:${query}:${mode}:${cursor}`

  if (mode === SearchPeople) {
    let usersData = await cache.get<any>(cacheKey)
    if (!usersData) {
      const result = await fetchSearchUsersWithFallback(c.env, query, cursor)
      usersData = { users: result.users, cursor: result.cursor }
      await cache.set(cacheKey, usersData, CACHE_SEARCH)
    }
    return c.json(usersData)
  } else {
    let searchData = await cache.get<any>(cacheKey)
    if (!searchData) {
      const result = await fetchSearchTweetsWithFallback(c.env, query, mode, cursor)
      searchData = { tweets: result.tweets, cursor: result.cursor }
      await cache.set(cacheKey, searchData, CACHE_SEARCH)
    }
    return c.json(searchData)
  }
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
  const client = gql(c)
  const cache = kv(c)

  const profileKey = `profile:${username.toLowerCase()}`
  let profile = await cache.get<any>(profileKey)
  if (!profile) {
    const data = await client.doGraphQL(gqlUserByScreenName, {
      screen_name: username, withSafetyModeUserFields: true,
    }, userFieldToggles)
    profile = parseUserResult(data)
    if (profile) await cache.set(profileKey, profile, CACHE_PROFILE)
  }
  if (!profile) return c.json({ error: 'User not found' }, 404)

  const endpoint = type === 'followers' ? gqlFollowers : gqlFollowing
  const cacheKey = `${type}:${username.toLowerCase()}:${cursor}`
  let followData = await cache.get<any>(cacheKey)
  if (!followData) {
    const vars: Record<string, unknown> = {
      userId: profile.id, count: 50, includePromotedContent: false,
    }
    if (cursor) vars.cursor = cursor
    const data = await client.doGraphQL(endpoint, vars, '')
    const result = parseFollowList(data)
    followData = { users: result.users, cursor: result.cursor }
    await cache.set(cacheKey, followData, CACHE_FOLLOW)
  }

  return c.json(followData)
}

// GET /api/list/:id?tab=tweets|members&cursor=
app.get('/list/:id', async (c) => {
  const listID = c.req.param('id')
  const tab = c.req.query('tab') || 'tweets'
  const cursor = c.req.query('cursor') || ''
  const client = gql(c)
  const cache = kv(c)

  const listKey = `list:${listID}`
  let list = await cache.get<any>(listKey)
  if (!list) {
    const data = await client.doGraphQL(gqlListById, { listId: listID }, '')
    list = parseGraphList(data)
    if (list) await cache.set(listKey, list, CACHE_LIST)
  }
  if (!list) return c.json({ error: 'List not found' }, 404)

  if (tab === 'members') {
    const membersKey = `list-members:${listID}:${cursor}`
    let membersData = await cache.get<any>(membersKey)
    if (!membersData) {
      const vars: Record<string, unknown> = { listId: listID, count: 200 }
      if (cursor) vars.cursor = cursor
      const data = await client.doGraphQL(gqlListMembers, vars, '')
      const result = parseListMembers(data)
      membersData = { users: result.users, cursor: result.cursor }
      await cache.set(membersKey, membersData, CACHE_LIST)
    }
    return c.json({ list, ...membersData })
  } else {
    const tweetsKey = `list-tweets:${listID}:${cursor}`
    let tweetsData = await cache.get<any>(tweetsKey)
    if (!tweetsData) {
      const vars: Record<string, unknown> = { rest_id: listID, count: 40 }
      if (cursor) vars.cursor = cursor
      const data = await client.doGraphQL(gqlListTweets, vars, '')
      const result = parseListTimeline(data)
      tweetsData = { tweets: result.tweets, cursor: result.cursor }
      await cache.set(tweetsKey, tweetsData, CACHE_LIST)
    }
    return c.json({ list, ...tweetsData })
  }
})

export default app
