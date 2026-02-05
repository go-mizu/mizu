import { Hono } from 'hono'
import { InstantService } from '../services/instant'
import { CacheStore } from '../store/cache'

type Env = {
  Bindings: {
    SEARCH_KV: KVNamespace
    ENVIRONMENT: string
  }
}

const app = new Hono<Env>()

app.get('/calculate', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const cache = new CacheStore(c.env.SEARCH_KV)
  const instantService = new InstantService(cache)
  const result = instantService.calculate(q)
  return c.json({
    type: 'calculation',
    query: q,
    answer: result,
  })
})

app.get('/convert', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const cache = new CacheStore(c.env.SEARCH_KV)
  const instantService = new InstantService(cache)
  const result = instantService.convert(q)
  return c.json({
    type: 'conversion',
    query: q,
    answer: result,
  })
})

app.get('/currency', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const cache = new CacheStore(c.env.SEARCH_KV)
  const instantService = new InstantService(cache)
  const result = await instantService.currency(q)
  return c.json({
    type: 'currency',
    query: q,
    answer: result,
  })
})

app.get('/weather', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const cache = new CacheStore(c.env.SEARCH_KV)
  const instantService = new InstantService(cache)
  const result = await instantService.weather(q)
  return c.json({
    type: 'weather',
    query: q,
    answer: result,
  })
})

app.get('/define', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const cache = new CacheStore(c.env.SEARCH_KV)
  const instantService = new InstantService(cache)
  const result = await instantService.define(q)
  return c.json({
    type: 'definition',
    query: q,
    answer: result,
  })
})

app.get('/time', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const cache = new CacheStore(c.env.SEARCH_KV)
  const instantService = new InstantService(cache)
  const result = instantService.time(q)
  return c.json({
    type: 'time',
    query: q,
    answer: result,
  })
})

export default app
