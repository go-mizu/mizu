import { Hono } from 'hono'
import { JinaReaderEngine, JinaKeyError } from '../engines/jina'
import type { HonoEnv } from '../types'

const app = new Hono<HonoEnv>()

/**
 * Resolve the best available Jina API key.
 * Priority: KV-stored key > env var.
 * Returns null if the key is currently rate-limited.
 */
async function resolveApiKey(
  kv: KVNamespace,
  envKey: string | undefined
): Promise<{ key: string | null; source: 'kv' | 'env' }> {
  // Check if key is currently marked as limited
  const statusRaw = await kv.get('secrets:jina_api_key:status');
  if (statusRaw) {
    try {
      const status = JSON.parse(statusRaw) as { limited: boolean };
      if (status.limited) {
        return { key: null, source: 'env' };
      }
    } catch {
      // ignore
    }
  }

  // Try KV-stored key first (allows runtime rotation without redeploy)
  const kvRaw = await kv.get('secrets:jina_api_key');
  if (kvRaw) {
    try {
      const data = JSON.parse(kvRaw) as { key: string };
      if (data.key) return { key: data.key, source: 'kv' };
    } catch {
      // Plain string fallback
      if (kvRaw.startsWith('jina_')) return { key: kvRaw, source: 'kv' };
    }
  }

  // Fall back to env var
  if (envKey) return { key: envKey, source: 'env' };

  return { key: null, source: 'env' };
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
  const { key: apiKey } = await resolveApiKey(c.env.SEARCH_KV, envKey)

  if (!apiKey) {
    return c.json({ error: 'Reader service not configured or temporarily rate-limited' }, 503)
  }

  try {
    const reader = new JinaReaderEngine()
    const result = await reader.readPage(url, apiKey)
    return c.json(result)
  } catch (err) {
    if (err instanceof JinaKeyError) {
      // Mark key as rate-limited in KV (TTL: 5 minutes for 429, 1 hour for 401)
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

/**
 * PUT /api/read/key - Update the Jina API key stored in KV.
 * This allows key rotation without redeploying the worker.
 */
app.put('/key', async (c) => {
  let body: { key?: string }
  try {
    body = await c.req.json()
  } catch {
    return c.json({ error: 'Invalid JSON body' }, 400)
  }

  if (!body.key || typeof body.key !== 'string') {
    return c.json({ error: 'Missing required field: key' }, 400)
  }

  await c.env.SEARCH_KV.put('secrets:jina_api_key', JSON.stringify({
    key: body.key,
    updatedAt: new Date().toISOString(),
  }))

  // Clear any rate-limit status for the new key
  await c.env.SEARCH_KV.delete('secrets:jina_api_key:status')

  return c.json({ success: true, message: 'API key updated' })
})

/**
 * GET /api/read/status - Check the Jina API key status.
 */
app.get('/status', async (c) => {
  const envKey = (c.env as unknown as Record<string, string>).JINA_API_KEY
  const { key: apiKey, source } = await resolveApiKey(c.env.SEARCH_KV, envKey)

  const statusRaw = await c.env.SEARCH_KV.get('secrets:jina_api_key:status')
  let limited = false
  let limitedAt: string | undefined
  if (statusRaw) {
    try {
      const status = JSON.parse(statusRaw) as { limited: boolean; limitedAt: string }
      limited = status.limited
      limitedAt = status.limitedAt
    } catch {
      // ignore
    }
  }

  return c.json({
    configured: !!apiKey,
    source,
    limited,
    limitedAt,
    keyPrefix: apiKey ? `${apiKey.slice(0, 8)}...` : null,
  })
})

export default app
