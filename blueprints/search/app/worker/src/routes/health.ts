import { Hono } from 'hono'
import type { HonoEnv } from '../types'

const app = new Hono<HonoEnv>()

app.get('/', (c) => {
  return c.json({ status: 'ok' })
})

export default app
