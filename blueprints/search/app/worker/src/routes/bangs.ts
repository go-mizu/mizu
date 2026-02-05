import { Hono } from 'hono'
import { BangService } from '../services/bang'
import { KVStore } from '../store/kv'
import type { HonoEnv } from '../types'

const app = new Hono<HonoEnv>()

app.get('/', async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV)
  const bangService = new BangService(kvStore)
  const bangs = await bangService.listBangs()
  return c.json(bangs)
})

app.get('/parse', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const kvStore = new KVStore(c.env.SEARCH_KV)
  const bangService = new BangService(kvStore)
  const result = await bangService.parse(q)
  return c.json(result)
})

app.post('/', async (c) => {
  const body = await c.req.json()

  const kvStore = new KVStore(c.env.SEARCH_KV)
  const bangService = new BangService(kvStore)
  const bang = await bangService.createBang(body)
  return c.json(bang, 201)
})

app.delete('/:id', async (c) => {
  const id = c.req.param('id')
  const kvStore = new KVStore(c.env.SEARCH_KV)
  const bangService = new BangService(kvStore)
  await bangService.deleteBang(id)
  return c.json({ success: true })
})

export default app
