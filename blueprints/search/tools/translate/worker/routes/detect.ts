import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { detectWithFallback } from '../providers/chain'
import { track } from '../analytics'

const route = new Hono<HonoEnv>()

route.post('/detect', async (c) => {
  const body = await c.req.json<{ text?: string }>()

  if (!body.text || typeof body.text !== 'string' || body.text.trim().length === 0) {
    return c.json({ error: 'Missing or empty "text" field' }, 400)
  }

  const t0 = Date.now()

  try {
    const result = await detectWithFallback(body.text)
    track(c.env, {
      event: 'detect',
      sl: result.language,
      latencyMs: Date.now() - t0,
      chars: body.text.length,
      success: true,
    })
    return c.json(result)
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Detection failed'
    track(c.env, {
      event: 'detect',
      extra: message,
      latencyMs: Date.now() - t0,
      chars: body.text.length,
      success: false,
    })
    return c.json({ error: message }, 502)
  }
})

export default route
