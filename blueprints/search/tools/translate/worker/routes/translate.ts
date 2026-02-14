import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { translateWithFallback } from '../providers/chain'

const route = new Hono<HonoEnv>()

route.post('/translate', async (c) => {
  const body = await c.req.json<{ text?: string; from?: string; to?: string }>()

  if (!body.text || typeof body.text !== 'string' || body.text.trim().length === 0) {
    return c.json({ error: 'Missing or empty "text" field' }, 400)
  }

  if (!body.to || typeof body.to !== 'string') {
    return c.json({ error: 'Missing "to" field (target language code)' }, 400)
  }

  if (body.text.length > 5000) {
    return c.json({ error: 'Text exceeds maximum length of 5000 characters' }, 400)
  }

  const from = body.from || 'auto'

  try {
    const result = await translateWithFallback(body.text, from, body.to)
    return c.json(result)
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Translation failed'
    return c.json({ error: message }, 502)
  }
})

export default route
