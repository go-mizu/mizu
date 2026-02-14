import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { allLanguages } from '../providers/chain'

const route = new Hono<HonoEnv>()

route.get('/languages', (c) => {
  return c.json({ languages: allLanguages() })
})

export default route
