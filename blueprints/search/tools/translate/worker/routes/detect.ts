import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { detectWithFallback } from '../providers/chain'

const route = new Hono<HonoEnv>()

route.post('/detect', async (c) => {
  const body = await c.req.json<{ text?: string }>()

  if (!body.text || typeof body.text !== 'string' || body.text.trim().length === 0) {
    return c.json({ error: 'Missing or empty "text" field' }, 400)
  }

  try {
    const result = await detectWithFallback(body.text)
    return c.json(result)
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Detection failed'
    return c.json({ error: message }, 502)
  }
})

export default route
