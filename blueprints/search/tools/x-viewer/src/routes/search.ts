import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { GraphQLClient } from '../graphql'
import { Cache } from '../cache'
import { parseSearchTweets, parseSearchUsers } from '../parse'
import { renderLayout, renderTweetCard, renderMediaGrid, renderUserCard, renderPagination, renderError } from '../html'
import { gqlSearchTimeline, SearchTop, SearchPeople, SearchMedia, SearchLists, CACHE_SEARCH } from '../config'

const app = new Hono<HonoEnv>()

const tabs = ['Top', 'Latest', 'People', 'Media', 'Lists'] as const

// Permanent search path: /search/golang -> search for "golang"
app.get('/:keyword', async (c) => {
  const keyword = c.req.param('keyword')
  const mode = c.req.query('mode') || SearchTop
  const cursor = c.req.query('cursor') || ''
  return handleSearch(c, keyword, mode, cursor)
})

app.get('/', async (c) => {
  const query = c.req.query('q') || ''
  const mode = c.req.query('mode') || SearchTop
  const cursor = c.req.query('cursor') || ''

  if (query.startsWith('@')) {
    const username = query.slice(1).trim()
    if (username) return c.redirect(`/${username}`)
  }

  if (!query) {
    return c.html(renderLayout('Search', `<div class="sh"><h2>Search</h2></div><div class="err"><p>Enter a query in the search bar above.</p></div>`))
  }

  return handleSearch(c, query, mode, cursor)
})

async function handleSearch(c: any, query: string, mode: string, cursor: string) {
  if (query.startsWith('@')) {
    const username = query.slice(1).trim()
    if (username) return c.redirect(`/${username}`)
  }

  const gql = new GraphQLClient(c.env.X_AUTH_TOKEN, c.env.X_CT0, c.env.X_BEARER_TOKEN)
  const cache = new Cache(c.env.KV)
  const baseQ = encodeURIComponent(query)

  // Render tabs
  let content = '<div class="tabs">'
  for (const t of tabs) {
    content += `<a href="/search/${baseQ}?mode=${t}" class="${mode === t ? 'active' : ''}">${t}</a>`
  }
  content += '</div>'

  // Lists tab — no API product available
  if (mode === SearchLists) {
    content += `<div class="err"><p>List search is not yet available.</p></div>`
    return c.html(renderLayout(`${query} - Search`, content, { query }))
  }

  try {
    const cacheKey = `search:${query}:${mode}:${cursor}`

    // Map "Media" tab to "Photos" API product
    const apiProduct = mode === SearchMedia ? 'Photos' : mode

    if (mode === SearchPeople) {
      let usersData = await cache.get<{ users: unknown[]; cursor: string }>(cacheKey)
      if (!usersData) {
        const vars: Record<string, unknown> = {
          rawQuery: query,
          count: 40,
          querySource: 'typed_query',
          product: 'People',
        }
        if (cursor) vars.cursor = cursor
        const data = await gql.doGraphQL(gqlSearchTimeline, vars, '')
        const result = parseSearchUsers(data)
        usersData = { users: result.users, cursor: result.cursor }
        await cache.set(cacheKey, usersData, CACHE_SEARCH)
      }

      const users = (usersData.users || []) as Parameters<typeof renderUserCard>[0][]
      const nextCursor = usersData.cursor as string

      if (users.length === 0) {
        content += `<div class="err"><h2>No results</h2><p>Try searching for something else.</p></div>`
      } else {
        for (const u of users) content += renderUserCard(u)
      }
      content += renderPagination(nextCursor, `/search/${baseQ}?mode=${mode}`)
    } else {
      let searchData = await cache.get<{ tweets: unknown[]; cursor: string }>(cacheKey)
      if (!searchData) {
        const vars: Record<string, unknown> = {
          rawQuery: query,
          count: 40,
          querySource: 'typed_query',
          product: apiProduct,
        }
        if (cursor) vars.cursor = cursor
        const data = await gql.doGraphQL(gqlSearchTimeline, vars, '')
        const result = parseSearchTweets(data)
        searchData = { tweets: result.tweets, cursor: result.cursor }
        await cache.set(cacheKey, searchData, CACHE_SEARCH)
      }

      const tweets = (searchData.tweets || []) as Parameters<typeof renderTweetCard>[0][]
      const nextCursor = searchData.cursor as string

      if (tweets.length === 0) {
        content += `<div class="err"><h2>No results</h2><p>Try searching for something else.</p></div>`
      } else if (mode === SearchMedia) {
        content += renderMediaGrid(tweets)
      } else {
        for (const tweet of tweets) content += renderTweetCard(tweet)
      }
      content += renderPagination(nextCursor, `/search/${baseQ}?mode=${mode}`)
    }

    return c.html(renderLayout(`${query} - Search`, content, { query }))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (msg.includes('rate limited')) return c.html(renderError('Rate Limited', 'Too many requests. Please try again later.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
}

export default app
