import { Hono } from 'hono'
import type { HonoEnv } from '../types'

const TTS_BASE = 'https://translate.google.com/translate_tts'

const route = new Hono<HonoEnv>()

route.get('/tts', async (c) => {
  const tl = c.req.query('tl')
  const q = c.req.query('q')

  if (!tl || typeof tl !== 'string') {
    return c.json({ error: 'Missing "tl" query parameter (target language)' }, 400)
  }

  if (!q || typeof q !== 'string' || q.trim().length === 0) {
    return c.json({ error: 'Missing or empty "q" query parameter (text)' }, 400)
  }

  if (q.length > 200) {
    return c.json({ error: 'Text exceeds maximum length of 200 characters for TTS' }, 400)
  }

  const params = new URLSearchParams({
    ie: 'UTF-8',
    client: 'tw-ob',
    tl,
    q,
    total: '1',
    idx: '0',
    textlen: String(q.length),
  })

  try {
    const resp = await fetch(`${TTS_BASE}?${params.toString()}`, {
      headers: {
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
      },
    })

    if (!resp.ok) {
      return c.json({ error: `TTS upstream returned ${resp.status}` }, 502)
    }

    const audio = await resp.arrayBuffer()
    return new Response(audio, {
      headers: {
        'Content-Type': 'audio/mpeg',
        'Cache-Control': 'public, max-age=86400',
      },
    })
  } catch (err) {
    const message = err instanceof Error ? err.message : 'TTS request failed'
    return c.json({ error: message }, 502)
  }
})

export default route
