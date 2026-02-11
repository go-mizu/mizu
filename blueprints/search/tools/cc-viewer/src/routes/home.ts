import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { renderLayout, renderHomePage } from '../html'

const app = new Hono<HonoEnv>()

app.get('/', (c) => {
  return c.html(renderLayout('Common Crawl Viewer', renderHomePage(), { isHome: true }))
})

export default app
