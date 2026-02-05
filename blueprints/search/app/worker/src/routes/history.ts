import { Hono } from 'hono'
import { KVStore } from '../store/kv'
import type { HonoEnv } from '../types'

const app = new Hono<HonoEnv>()

app.get('/', async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV)
  const history = await kvStore.listHistory()
  return c.json(history)
})

app.delete('/', async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV)
  await kvStore.clearHistory()
  return c.json({ success: true })
})

app.delete('/:id', async (c) => {
  const id = c.req.param('id')
  const kvStore = new KVStore(c.env.SEARCH_KV)
  await kvStore.deleteHistory(id)
  return c.json({ success: true })
})

export default app
