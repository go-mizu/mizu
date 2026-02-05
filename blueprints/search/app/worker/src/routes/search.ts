import { Hono } from 'hono'
import { createDefaultMetaSearch } from '../engines/metasearch'
import { CacheStore } from '../store/cache'
import { KVStore } from '../store/kv'
import { SearchService } from '../services/search'
import { BangService } from '../services/bang'
import { InstantService } from '../services/instant'
import { KnowledgeService } from '../services/knowledge'
import type { SearchOptions } from '../types'

type Env = {
  Bindings: {
    SEARCH_KV: KVNamespace
    ENVIRONMENT: string
  }
}

function extractSearchOptions(c: { req: { query: (key: string) => string | undefined } }): SearchOptions {
  return {
    page: parseInt(c.req.query('page') ?? '1', 10),
    per_page: parseInt(c.req.query('per_page') ?? '10', 10),
    time_range: c.req.query('time') ?? '',
    region: c.req.query('region') ?? '',
    language: c.req.query('lang') ?? 'en',
    safe_search: c.req.query('safe') ?? 'moderate',
  }
}

function createServices(kv: KVNamespace) {
  const cache = new CacheStore(kv)
  const kvStore = new KVStore(kv)
  const metaSearch = createDefaultMetaSearch()
  const bangService = new BangService(kvStore)
  const instantService = new InstantService(cache)
  const knowledgeService = new KnowledgeService(cache)
  const searchService = new SearchService(metaSearch, cache, kvStore, bangService, instantService, knowledgeService)
  return { searchService }
}

const app = new Hono<Env>()

app.get('/', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const options = extractSearchOptions(c)
  const { searchService } = createServices(c.env.SEARCH_KV)
  const results = await searchService.search(q, options)
  return c.json(results)
})

app.get('/images', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const options = extractSearchOptions(c)
  const { searchService } = createServices(c.env.SEARCH_KV)
  const results = await searchService.searchImages(q, options)
  return c.json(results)
})

app.get('/videos', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const options = extractSearchOptions(c)
  const { searchService } = createServices(c.env.SEARCH_KV)
  const results = await searchService.searchVideos(q, options)
  return c.json(results)
})

app.get('/news', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const options = extractSearchOptions(c)
  const { searchService } = createServices(c.env.SEARCH_KV)
  const results = await searchService.searchNews(q, options)
  return c.json(results)
})

export default app
