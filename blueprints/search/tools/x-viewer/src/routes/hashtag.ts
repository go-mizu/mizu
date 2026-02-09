import { Hono } from 'hono'
import type { HonoEnv } from '../types'

const app = new Hono<HonoEnv>()

app.get('/:tag', (c) => {
  const tag = c.req.param('tag')
  return c.redirect(`/search?q=${encodeURIComponent('#' + tag)}`)
})

export default app
