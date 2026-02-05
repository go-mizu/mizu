import { Hono } from 'hono'
import { KVStore } from '../store/kv'
import { generateId } from '../lib/utils'
import type { SearchLens } from '../types'

type Env = {
  Bindings: {
    SEARCH_KV: KVNamespace
    ENVIRONMENT: string
  }
}

const app = new Hono<Env>()

app.get('/', async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV)
  const lenses = await kvStore.listLenses()
  return c.json(lenses)
})

app.post('/', async (c) => {
  const body = await c.req.json<Partial<SearchLens>>()
  const now = new Date().toISOString()
  const lens: SearchLens = {
    id: generateId(),
    name: body.name ?? 'Untitled',
    description: body.description,
    domains: body.domains,
    exclude: body.exclude,
    include_keywords: body.include_keywords,
    exclude_keywords: body.exclude_keywords,
    keywords: body.keywords,
    region: body.region,
    file_type: body.file_type,
    date_before: body.date_before,
    date_after: body.date_after,
    is_public: body.is_public ?? false,
    is_built_in: false,
    is_shared: body.is_shared ?? false,
    share_link: body.share_link,
    user_id: body.user_id,
    created_at: now,
    updated_at: now,
  }

  const kvStore = new KVStore(c.env.SEARCH_KV)
  await kvStore.createLens(lens)
  return c.json(lens, 201)
})

app.get('/:id', async (c) => {
  const id = c.req.param('id')
  const kvStore = new KVStore(c.env.SEARCH_KV)
  const lens = await kvStore.getLens(id)

  if (!lens) {
    return c.json({ error: 'Lens not found' }, 404)
  }

  return c.json(lens)
})

app.put('/:id', async (c) => {
  const id = c.req.param('id')
  const body = await c.req.json()

  const kvStore = new KVStore(c.env.SEARCH_KV)
  const lens = await kvStore.updateLens(id, body)
  return c.json(lens)
})

app.delete('/:id', async (c) => {
  const id = c.req.param('id')
  const kvStore = new KVStore(c.env.SEARCH_KV)
  await kvStore.deleteLens(id)
  return c.json({ success: true })
})

export default app
