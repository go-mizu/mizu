import { Hono } from 'hono'
import { cors } from 'hono/cors'
import type { Env, HonoEnv } from './types'
import translateRoute from './routes/translate'
import languagesRoute from './routes/languages'
import detectRoute from './routes/detect'
import ttsRoute from './routes/tts'
import pageRoute from './routes/page'

const app = new Hono<HonoEnv>()

app.use('/api/*', cors({
  origin: '*',
  allowMethods: ['GET', 'POST', 'OPTIONS'],
  allowHeaders: ['Content-Type'],
}))

app.get('/api/health', (c) => c.json({ status: 'ok' }))
app.route('/api', translateRoute)
app.route('/api', languagesRoute)
app.route('/api', detectRoute)
app.route('/api', ttsRoute)
app.route('', pageRoute)

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext): Promise<Response> {
    const url = new URL(request.url)
    if (url.pathname.startsWith('/api/') || url.pathname.startsWith('/page/')) {
      return app.fetch(request, env, ctx)
    }
    if (env.ASSETS) {
      return env.ASSETS.fetch(request)
    }
    return new Response(JSON.stringify({
      error: 'No static assets available',
      detail: 'Run "npm run build" to generate the SPA, or use /api/ endpoints directly.',
      api: { health: '/api/health', translate: 'POST /api/translate', languages: 'GET /api/languages', detect: 'POST /api/detect', tts: 'GET /api/tts?tl=LANG&q=TEXT' },
    }), { status: 200, headers: { 'Content-Type': 'application/json' } })
  },
}
