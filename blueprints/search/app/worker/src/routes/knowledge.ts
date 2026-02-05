import { Hono } from 'hono'
import { KnowledgeService } from '../services/knowledge'
import { CacheStore } from '../store/cache'
import type { HonoEnv } from '../types'

const app = new Hono<HonoEnv>()

app.get('/:query', async (c) => {
  const query = c.req.param('query')
  if (!query) {
    return c.json({ error: 'Missing required parameter: query' }, 400)
  }

  const cache = new CacheStore(c.env.SEARCH_KV)
  const knowledgeService = new KnowledgeService(cache)
  const panel = await knowledgeService.getPanel(query)

  if (!panel) {
    return c.json({ error: 'No knowledge panel found' }, 404)
  }

  return c.json(panel)
})

export default app
