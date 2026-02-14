import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { Cache } from '../cache'
import { Analytics } from '../analytics'

const app = new Hono<HonoEnv>()

// GET /analytics/popular — most viewed domains/URLs
app.get('/popular', async (c) => {
  const start = Date.now()
  const type = c.req.query('type') || 'domains' // 'domains' | 'urls' | 'searches'
  const limit = parseInt(c.req.query('limit') || '20')
  const period = c.req.query('period') || '7d' // '1d' | '7d' | '30d'
  const cache = new Cache(c.env.KV)
  const analytics = new Analytics(c.env.ANALYTICS)

  try {
    // Try to read from KV counters (populated by background jobs or inline tracking)
    const cacheKey = `popular:${type}:${period}:${limit}`
    const cached = await cache.get<object>(cacheKey)
    if (cached) {
      analytics.apiCall('/api/analytics/popular', Date.now() - start, true)
      return c.json(cached)
    }

    // Fallback: read from per-item counters in KV
    const counterPrefix = `counter:${type}:`
    const items: Array<{ name: string; count: number }> = []

    // We store top items in a known KV key updated periodically
    const topItems = await cache.get<Array<{ name: string; count: number }>>(`top:${type}`)
    if (topItems) {
      const result = {
        type,
        period,
        items: topItems.slice(0, limit),
        total: topItems.length,
        source: 'kv-counters',
      }
      await cache.set(cacheKey, result)
      analytics.apiCall('/api/analytics/popular', Date.now() - start, true)
      return c.json(result)
    }

    // If no counters exist yet, return well-known popular domains as seed data
    const seedDomains = [
      { name: 'wikipedia.org', count: 0 },
      { name: 'github.com', count: 0 },
      { name: 'stackoverflow.com', count: 0 },
      { name: 'reddit.com', count: 0 },
      { name: 'youtube.com', count: 0 },
      { name: 'twitter.com', count: 0 },
      { name: 'amazon.com', count: 0 },
      { name: 'facebook.com', count: 0 },
      { name: 'bbc.com', count: 0 },
      { name: 'nytimes.com', count: 0 },
    ]

    const seedURLs = [
      { name: 'https://en.wikipedia.org/wiki/Main_Page', count: 0 },
      { name: 'https://www.google.com/', count: 0 },
      { name: 'https://github.com/', count: 0 },
      { name: 'https://www.reddit.com/', count: 0 },
      { name: 'https://stackoverflow.com/', count: 0 },
    ]

    const seedSearches = [
      { name: 'wikipedia.org', count: 0 },
      { name: 'example.com', count: 0 },
      { name: 'github.com', count: 0 },
      { name: 'bbc.com/news', count: 0 },
      { name: 'stackoverflow.com', count: 0 },
    ]

    let seedData: Array<{ name: string; count: number }>
    switch (type) {
      case 'urls':
        seedData = seedURLs
        break
      case 'searches':
        seedData = seedSearches
        break
      default:
        seedData = seedDomains
    }

    const result = {
      type,
      period,
      items: seedData.slice(0, limit),
      total: seedData.length,
      source: 'seed-data',
      note: 'No analytics data collected yet. Counters will populate as the viewer is used.',
    }

    analytics.apiCall('/api/analytics/popular', Date.now() - start, false)
    return c.json(result)
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    analytics.error('/api/analytics/popular', 500, message)
    return c.json({ error: message }, 500)
  }
})

// GET /analytics/usage — API usage stats
app.get('/usage', async (c) => {
  const start = Date.now()
  const period = c.req.query('period') || '24h' // '1h' | '24h' | '7d' | '30d'
  const cache = new Cache(c.env.KV)
  const analytics = new Analytics(c.env.ANALYTICS)

  try {
    // Try to read pre-aggregated usage stats
    const cacheKey = `usage:${period}`
    const cached = await cache.get<object>(cacheKey)
    if (cached) {
      analytics.apiCall('/api/analytics/usage', Date.now() - start, true)
      return c.json(cached)
    }

    // Build usage stats from KV counters
    const endpoints = [
      'crawls', 'url', 'domain', 'domains', 'view', 'search',
      'stats', 'graph', 'news', 'analytics',
    ]

    const endpointStats = await Promise.all(
      endpoints.map(async (ep) => {
        const count = await cache.get<number>(`usage:${ep}:${period}`) || 0
        const avgLatency = await cache.get<number>(`latency:${ep}:${period}`) || 0
        return {
          endpoint: `/api/${ep}`,
          requests: count,
          avgLatencyMs: avgLatency,
        }
      })
    )

    const totalRequests = endpointStats.reduce((sum, e) => sum + e.requests, 0)
    const cacheHitRate = await cache.get<number>(`cache-hit-rate:${period}`) || 0

    const result = {
      period,
      totalRequests,
      cacheHitRate: Math.round(cacheHitRate * 100) / 100,
      endpoints: endpointStats.filter(e => e.requests > 0).sort((a, b) => b.requests - a.requests),
      kvReadsEstimate: totalRequests * 2, // Roughly 2 KV reads per request (cache check + data)
      timestamp: new Date().toISOString(),
    }

    await cache.set(cacheKey, result)
    analytics.apiCall('/api/analytics/usage', Date.now() - start, false)
    return c.json(result)
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    analytics.error('/api/analytics/usage', 500, message)
    return c.json({ error: message }, 500)
  }
})

export default app
