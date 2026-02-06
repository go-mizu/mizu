import { Hono } from 'hono'
import { JinaReaderEngine } from '../engines/jina'
import type { HonoEnv } from '../types'

const app = new Hono<HonoEnv>()

app.get('/', async (c) => {
  const url = c.req.query('url') ?? ''
  if (!url) {
    return c.json({ error: 'Missing required parameter: url' }, 400)
  }

  // Validate URL
  try {
    const parsed = new URL(url)
    if (!['http:', 'https:'].includes(parsed.protocol)) {
      return c.json({ error: 'Only http and https URLs are supported' }, 400)
    }
  } catch {
    return c.json({ error: 'Invalid URL' }, 400)
  }

  const apiKey = (c.env as unknown as Record<string, string>).JINA_API_KEY
  if (!apiKey) {
    return c.json({ error: 'Reader service not configured' }, 503)
  }

  try {
    const reader = new JinaReaderEngine()
    const result = await reader.readPage(url, apiKey)
    return c.json(result)
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Failed to read page'
    return c.json({ error: message }, 502)
  }
})

export default app
