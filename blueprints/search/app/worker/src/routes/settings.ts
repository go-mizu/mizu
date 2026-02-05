import { Hono } from 'hono'
import { KVStore } from '../store/kv'
import type { HonoEnv } from '../types'

const app = new Hono<HonoEnv>()

app.get('/', async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV)
  const settings = await kvStore.getSettings()
  return c.json(settings)
})

app.put('/', async (c) => {
  const body = await c.req.json()
  const kvStore = new KVStore(c.env.SEARCH_KV)
  const settings = await kvStore.updateSettings(body)
  return c.json(settings)
})

export default app
