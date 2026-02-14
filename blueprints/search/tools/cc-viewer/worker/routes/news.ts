import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { CCClient } from '../cc'
import { Cache } from '../cache'
import { Analytics } from '../analytics'

const app = new Hono<HonoEnv>()

// GET /news — CC-NEWS entries by date
app.get('/', async (c) => {
  const start = Date.now()
  const date = c.req.query('date')
  if (!date) {
    return c.json({
      error: 'date parameter required',
      detail: 'Provide a date in YYYYMMDD format (e.g., 20260115)',
      example: '/api/news?date=20260115',
    }, 400)
  }

  // Validate date format
  if (!/^\d{8}$/.test(date)) {
    return c.json({
      error: 'Invalid date format',
      detail: 'Date must be in YYYYMMDD format (e.g., 20260115)',
    }, 400)
  }

  const page = parseInt(c.req.query('page') || '0')
  const limit = parseInt(c.req.query('limit') || '50')
  const cache = new Cache(c.env.KV)
  const cc = new CCClient(cache)
  const analytics = new Analytics(c.env.ANALYTICS)

  try {
    const result = await cc.lookupNews(date, page, limit)

    // Compute basic stats for this date
    const domainCounts: Record<string, number> = {}
    const statusCounts: Record<string, number> = {}
    for (const entry of result.entries) {
      try {
        const u = new URL(entry.url)
        domainCounts[u.hostname] = (domainCounts[u.hostname] || 0) + 1
      } catch { /* skip */ }
      const s = entry.status || 'unknown'
      statusCounts[s] = (statusCounts[s] || 0) + 1
    }

    // Sort domains by count descending
    const topDomains = Object.entries(domainCounts)
      .sort(([, a], [, b]) => b - a)
      .slice(0, 20)
      .map(([domain, count]) => ({ domain, count }))

    analytics.apiCall('/api/news', Date.now() - start, false)
    analytics.search(date, 'news', result.entries.length)
    return c.json({
      date,
      entries: result.entries,
      page,
      totalPages: result.totalPages,
      total: result.entries.length,
      topDomains,
      statusCounts,
    })
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    analytics.error('/api/news', 500, message)
    return c.json({ error: message }, 500)
  }
})

// GET /news/dates — available dates list
app.get('/dates', async (c) => {
  const start = Date.now()
  const limit = parseInt(c.req.query('limit') || '90')
  const cache = new Cache(c.env.KV)
  const cc = new CCClient(cache)
  const analytics = new Analytics(c.env.ANALYTICS)

  try {
    const allDates = await cc.listNewsDates()
    const dates = allDates.slice(0, limit)

    // Format dates for the frontend
    const formatted = dates.map(d => ({
      raw: d,
      formatted: `${d.substring(0, 4)}-${d.substring(4, 6)}-${d.substring(6, 8)}`,
      year: d.substring(0, 4),
      month: d.substring(4, 6),
      day: d.substring(6, 8),
    }))

    // Group by month
    const byMonth: Record<string, string[]> = {}
    for (const d of dates) {
      const month = d.substring(0, 6)
      if (!byMonth[month]) byMonth[month] = []
      byMonth[month].push(d)
    }

    analytics.apiCall('/api/news/dates', Date.now() - start, false)
    return c.json({
      dates: formatted,
      totalDates: allDates.length,
      showing: dates.length,
      byMonth,
    })
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    analytics.error('/api/news/dates', 500, message)
    return c.json({ error: message }, 500)
  }
})

export default app
