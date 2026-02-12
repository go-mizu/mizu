import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { ensureSchema } from '../db'
import * as db from '../db'
import { renderLayout, renderChallengePage } from '../html'

const app = new Hono<HonoEnv>()

// Challenge page
app.get('/', async (c) => {
  await ensureSchema(c.env.DB)

  const currentYear = new Date().getFullYear()
  const challenge = await db.getChallenge(c.env.DB, currentYear)

  return c.html(renderLayout(`${currentYear} Reading Challenge - Books`, renderChallengePage(challenge)))
})

// Set challenge goal
app.post('/', async (c) => {
  await ensureSchema(c.env.DB)

  const body = await c.req.parseBody()
  const goal = parseInt(body.goal as string, 10)
  if (!goal || goal < 1) return c.redirect('/challenge', 302)

  const currentYear = new Date().getFullYear()
  await db.setChallenge(c.env.DB, currentYear, goal)

  return c.redirect('/challenge', 302)
})

export default app
