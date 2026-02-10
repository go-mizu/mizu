import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { SessionManager } from '../session'
import { Cache } from '../cache'
import { parseHashtagPosts } from '../parse'
import { renderLayout, renderPostGrid, renderPageHeader, renderPagination, renderError } from '../html'
import { CACHE_HASHTAG, qhHashtag } from '../config'

const app = new Hono<HonoEnv>()

app.get('/:tag', async (c) => {
  const tag = c.req.param('tag')
  const cursor = c.req.query('cursor') || ''
  const client = await new SessionManager(c.env).getClient()
  const cache = new Cache(c.env.KV)

  try {
    const cacheKey = `hashtag:${tag.toLowerCase()}:${cursor}`
    let data = await cache.get<any>(cacheKey)
    if (!data) {
      const vars: Record<string, unknown> = { tag_name: tag, first: 12 }
      if (cursor) vars.after = cursor
      const resp = await client.graphqlQuery(qhHashtag, vars)
      data = parseHashtagPosts(resp)
      await cache.set(cacheKey, data, CACHE_HASHTAG)
    }

    const header = cursor ? '' : renderPageHeader('#', `#${tag}`, `<strong>${data.posts?.length || 0}+</strong> posts`)
    const grid = renderPostGrid(data.posts || [])
    const pagination = renderPagination(data.cursor || '', `/explore/tags/${tag}`)

    return c.html(renderLayout(`#${tag}`, header + grid + pagination))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (msg.includes('rate limited')) return c.html(renderError('Rate Limited', 'Too many requests.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
})

export default app
