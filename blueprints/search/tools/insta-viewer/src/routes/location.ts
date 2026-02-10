import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { SessionManager } from '../session'
import { Cache } from '../cache'
import { parseLocationPosts } from '../parse'
import { renderLayout, renderPostGrid, renderPageHeader, renderPagination, renderError } from '../html'
import { CACHE_LOCATION, qhLocation } from '../config'

const app = new Hono<HonoEnv>()

// /explore/locations/:id or /explore/locations/:id/:name
app.get('/:id/:name?', async (c) => {
  const locationId = c.req.param('id')
  const cursor = c.req.query('cursor') || ''
  const client = await new SessionManager(c.env).getClient()
  const cache = new Cache(c.env.KV)

  try {
    const cacheKey = `location:${locationId}:${cursor}`
    let data = await cache.get<any>(cacheKey)
    if (!data) {
      const vars: Record<string, unknown> = { id: locationId, first: 12 }
      if (cursor) vars.after = cursor
      const resp = await client.graphqlQuery(qhLocation, vars)
      data = parseLocationPosts(resp)
      await cache.set(cacheKey, data, CACHE_LOCATION)
    }

    const locationName = data.locationName || c.req.param('name')?.replace(/-/g, ' ') || 'Location'
    const header = cursor ? '' : renderPageHeader(
      '<svg viewBox="0 0 24 24" fill="currentColor" width="44" height="44"><path d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7zm0 9.5c-1.38 0-2.5-1.12-2.5-2.5s1.12-2.5 2.5-2.5 2.5 1.12 2.5 2.5-1.12 2.5-2.5 2.5z"/></svg>',
      locationName,
      ''
    )
    const grid = renderPostGrid(data.posts || [])
    const pagination = renderPagination(data.cursor || '', `/explore/locations/${locationId}`)

    return c.html(renderLayout(locationName, header + grid + pagination))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (msg.includes('rate limited')) return c.html(renderError('Rate Limited', 'Too many requests.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
})

export default app
