import { Hono } from 'hono'
import { SuggestService } from '../services/suggest'
import { CacheStore } from '../store/cache'
import type { HonoEnv } from '../types'

const app = new Hono<HonoEnv>()

app.get('/', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const cache = new CacheStore(c.env.SEARCH_KV)
  const suggestService = new SuggestService(cache)
  const suggestions = await suggestService.suggest(q)
  return c.json(suggestions)
})

app.get('/trending', async (c) => {
  const cache = new CacheStore(c.env.SEARCH_KV)
  const suggestService = new SuggestService(cache)
  const trending = await suggestService.trending()
  return c.json(trending)
})

export default app
