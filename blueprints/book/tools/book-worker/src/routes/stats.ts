import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { ensureSchema } from '../db'
import * as db from '../db'
import { renderLayout, renderStatsPage } from '../html'

const app = new Hono<HonoEnv>()

// Stats page with optional year query param
app.get('/', async (c) => {
  await ensureSchema(c.env.DB)

  const currentYear = new Date().getFullYear()
  const year = parseInt(c.req.query('year') || String(currentYear), 10) || currentYear

  const stats = await db.getStats(c.env.DB, year)

  return c.html(renderLayout(`${year} Stats - Books`, renderStatsPage(stats, year)))
})

// Stats for specific year (redirect to query param)
app.get('/:year', (c) => {
  const year = c.req.param('year')
  return c.redirect(`/stats?year=${year}`, 302)
})

export default app
