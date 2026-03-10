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

const app = new Hono<HonoEnv>()

app.use('*', cors())

function gql(c: any) {
  return new GraphQLClient(c.env.X_AUTH_TOKEN, c.env.X_CT0, c.env.X_BEARER_TOKEN)
}

// GET /api/profile/:username
app.get('/profile/:username', async (c) => {
  const username = c.req.param('username')
  const db = new DB(c.env.DB)

  let profile = await db.getProfile<any>(username)
  if (!profile) {
    profile = await fetchProfileWithFallback(c.env, username)
    if (profile) await db.setProfile(username, profile, CACHE_PROFILE)
  }

  if (!profile) return c.json({ error: 'User not found' }, 404)
  return c.json({ profile })
})

// GET /api/tweets/:username?tab=tweets|replies|media&cursor=
app.get('/tweets/:username', async (c) => {
  const username = c.req.param('username')
  const tab = c.req.query('tab') || 'tweets'
  const cursor = c.req.query('cursor') || ''
  const db = new DB(c.env.DB)

  let profile = await db.getProfile<any>(username)
  if (!profile) {
    profile = await fetchProfileWithFallback(c.env, username)
    if (profile) await db.setProfile(username, profile, CACHE_PROFILE)
  }
  if (!profile) return c.json({ error: 'User not found' }, 404)

  let timelineData = await db.getTimeline<any>(username, tab, cursor)
  if (!timelineData) {
    const result = await fetchUserTimelineWithFallback(c.env, username, tab, cursor, profile.id || '')
    timelineData = { tweets: result.tweets, cursor: result.cursor }
    await db.setTimeline(username, tab, cursor, timelineData, CACHE_TIMELINE)
  }

  return c.json(timelineData)
})

// GET /api/tweet/:id
app.get('/tweet/:id', async (c) => {
  const tweetID = c.req.param('id')
  const cursor = c.req.query('cursor') || ''
  const db = new DB(c.env.DB)

  let cached = await db.getTweet<any>(tweetID, cursor)
  if (!cached) {
    const result = await fetchTweetConversation(c.env, tweetID, cursor, false)

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

  const db = new DB(c.env.DB)

  if (mode === SearchPeople) {
    let usersData = await db.getSearch<any>(query, mode, cursor)
    if (!usersData) {
      const result = await fetchSearchUsersWithFallback(c.env, query, cursor)
      usersData = { users: result.users, cursor: result.cursor }
      await db.setSearch(query, mode, cursor, usersData, CACHE_SEARCH)
    }
    return c.json(usersData)
  } else {
    let searchData = await db.getSearch<any>(query, mode, cursor)
    if (!searchData) {
      const result = await fetchSearchTweetsWithFallback(c.env, query, mode, cursor)
      searchData = { tweets: result.tweets, cursor: result.cursor }
      await db.setSearch(query, mode, cursor, searchData, CACHE_SEARCH)
    }
    return c.json(searchData)
  }
})

// GET /api/followers/:username?cursor=
app.get('/followers/:username', async (c) => handleFollow(c, 'followers'))
app.get('/following/:username', async (c) => handleFollow(c, 'following'))

async function handleFollow(c: any, type: 'followers' | 'following') {
  const username = c.req.param('username')
  const cursor = c.req.query('cursor') || ''
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
  let followData = await db.getFollow<any>(username, type, cursor)
  if (!followData) {
    const vars: Record<string, unknown> = {
      userId: profile.id, count: 50, includePromotedContent: false,
    }
    if (cursor) vars.cursor = cursor
    const data = await client.doGraphQL(endpoint, vars, '')
    const result = parseFollowList(data)
    followData = { users: result.users, cursor: result.cursor }
    await db.setFollow(username, type, cursor, followData, CACHE_FOLLOW)
  }

  return c.json(followData)
}

// GET /api/list/:id?tab=tweets|members&cursor=
app.get('/list/:id', async (c) => {
  const listID = c.req.param('id')
  const tab = c.req.query('tab') || 'tweets'
  const cursor = c.req.query('cursor') || ''
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
    let membersData = await db.getListContent<any>(listID, 'members', cursor)
    if (!membersData) {
      const vars: Record<string, unknown> = { listId: listID, count: 200 }
      if (cursor) vars.cursor = cursor
      const data = await client.doGraphQL(gqlListMembers, vars, '')
      const result = parseListMembers(data)
      membersData = { users: result.users, cursor: result.cursor }
      await db.setListContent(listID, 'members', cursor, membersData, CACHE_LIST)
    }
    return c.json({ list, ...membersData })
  } else {
    let tweetsData = await db.getListContent<any>(listID, 'tweets', cursor)
    if (!tweetsData) {
      const vars: Record<string, unknown> = { rest_id: listID, count: 40 }
      if (cursor) vars.cursor = cursor
      const data = await client.doGraphQL(gqlListTweets, vars, '')
      const result = parseListTimeline(data)
      tweetsData = { tweets: result.tweets, cursor: result.cursor }
      await db.setListContent(listID, 'tweets', cursor, tweetsData, CACHE_LIST)
    }
    return c.json({ list, ...tweetsData })
  }
})

export default app
