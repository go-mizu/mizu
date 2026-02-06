import { Hono } from 'hono'
import { JinaReaderEngine, JinaKeyError } from '../engines/jina'
import type { HonoEnv } from '../types'

const app = new Hono<HonoEnv>()

/**
 * Resolve the Jina API key from env var.
 * Returns null if the key is currently rate-limited.
 */
async function resolveApiKey(
  kv: KVNamespace,
  envKey: string | undefined
): Promise<string | null> {
  if (!envKey) return null;

  // Check if key is currently marked as limited
  const statusRaw = await kv.get('secrets:jina_api_key:status');
  if (statusRaw) {
    try {
      const status = JSON.parse(statusRaw) as { limited: boolean };
      if (status.limited) return null;
    } catch {
      // ignore
    }
  }

  return envKey;
}

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

  const envKey = (c.env as unknown as Record<string, string>).JINA_API_KEY
  const apiKey = await resolveApiKey(c.env.SEARCH_KV, envKey)

  // Jina works without a key at reduced rate limits (20 RPM)
  try {
    const reader = new JinaReaderEngine()
    const result = await reader.readPage(url, apiKey ?? '')
    return c.json(result)
  } catch (err) {
    if (err instanceof JinaKeyError && apiKey) {
      // Only mark key as rate-limited if one was actually in use
      const ttl = err.statusCode === 429 ? 300 : 3600
      await c.env.SEARCH_KV.put(
        'secrets:jina_api_key:status',
        JSON.stringify({ limited: true, limitedAt: new Date().toISOString() }),
        { expirationTtl: ttl }
      )
      return c.json({
        error: err.statusCode === 429
          ? 'Reader service is temporarily rate-limited. Try again in a few minutes.'
          : 'Reader service API key is expired or invalid.',
      }, 503)
    }

    const message = err instanceof Error ? err.message : 'Failed to read page'
    return c.json({ error: message }, 502)
  }
})

export default app
