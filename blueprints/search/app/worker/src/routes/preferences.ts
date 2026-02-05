import { Hono } from 'hono'
import { KVStore } from '../store/kv'
import { generateId } from '../lib/utils'
import type { UserPreference } from '../types'

type Env = {
  Bindings: {
    SEARCH_KV: KVNamespace
    ENVIRONMENT: string
  }
}

const app = new Hono<Env>()

app.get('/', async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV)
  const preferences = await kvStore.listPreferences()
  return c.json(preferences)
})

app.post('/', async (c) => {
  const body = await c.req.json<{
    domain: string
    action: string
    level?: number
  }>()

  if (!body.domain || !body.action) {
    return c.json({ error: 'Missing required fields: domain, action' }, 400)
  }

  const pref: UserPreference = {
    id: generateId(),
    domain: body.domain,
    action: body.action,
    level: body.level ?? 0,
    created_at: new Date().toISOString(),
  }

  const kvStore = new KVStore(c.env.SEARCH_KV)
  await kvStore.setPreference(pref)
  return c.json({ success: true })
})

app.delete('/:domain', async (c) => {
  const domain = c.req.param('domain')
  const kvStore = new KVStore(c.env.SEARCH_KV)
  await kvStore.deletePreference(domain)
  return c.json({ success: true })
})

export default app
