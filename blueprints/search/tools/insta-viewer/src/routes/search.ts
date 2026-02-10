import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { InstagramClient } from '../instagram'
import { Cache } from '../cache'
import { parseSearchResults } from '../parse'
import { renderLayout, renderSearchResults, renderError } from '../html'
import { CACHE_SEARCH } from '../config'

const app = new Hono<HonoEnv>()

app.get('/', async (c) => {
  const query = c.req.query('q') || ''
  if (!query) return c.html(renderError('Search', 'Please enter a search query.'))

  // Redirect @username to profile
  if (query.startsWith('@') && !query.includes(' ')) {
    const username = query.slice(1)
    return c.redirect(`/${username}`)
  }

  const client = new InstagramClient(c.env.INSTA_SESSION_ID, c.env.INSTA_CSRF_TOKEN, c.env.INSTA_DS_USER_ID, c.env.INSTA_MID, c.env.INSTA_IG_DID)
  const cache = new Cache(c.env.KV)

  try {
    const cacheKey = `search:${query.toLowerCase()}`
    let result = await cache.get<any>(cacheKey)
    if (!result) {
      const data = await client.search(query)
      result = parseSearchResults(data)
      await cache.set(cacheKey, result, CACHE_SEARCH)
    }

    const content = renderSearchResults(result)
    return c.html(renderLayout(`Search: ${query}`, content, { query }))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (msg.includes('rate limited')) return c.html(renderError('Rate Limited', 'Too many requests.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
})

export default app
