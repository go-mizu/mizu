import { Hono } from 'hono'
import type { HonoEnv, GraphData } from '../types'
import { CCClient } from '../cc'
import { Cache } from '../cache'
import { Analytics } from '../analytics'

const CC_DATA = 'https://data.commoncrawl.org'

const app = new Hono<HonoEnv>()

// GET /graph — web graph summary
app.get('/', async (c) => {
  const start = Date.now()
  const crawl = c.req.query('crawl')
  const type = (c.req.query('type') || 'host') as 'host' | 'domain'
  const cache = new Cache(c.env.KV)
  const cc = new CCClient(cache)
  const analytics = new Analytics(c.env.ANALYTICS)

  try {
    const crawlID = crawl || await cc.getLatestCrawl()

    // Try pre-processed graph data from KV
    const preProcessed = await cc.getGraph(crawlID, type)
    if (preProcessed) {
      analytics.apiCall('/api/graph', Date.now() - start, true)
      return c.json(preProcessed)
    }

    // Fallback: check if graph data is available for this crawl
    const crawlSlug = crawlID.toLowerCase()
    const graphUrl = `${CC_DATA}/projects/hyperlinkgraph/${crawlSlug}/${type}/vertices.paths.gz`
    const headRes = await fetch(graphUrl, { method: 'HEAD' })

    if (!headRes.ok) {
      // Graph data may not be available for the latest crawl
      // Try to find the most recent crawl that has graph data
      const crawls = await cc.listCrawls()
      let foundGraph: GraphData | null = null

      for (const c of crawls.slice(1, 6)) {
        const slug = c.id.toLowerCase()
        const checkUrl = `${CC_DATA}/projects/hyperlinkgraph/${slug}/${type}/vertices.paths.gz`
        const check = await fetch(checkUrl, { method: 'HEAD' })
        if (check.ok) {
          const contentLength = parseInt(check.headers.get('content-length') || '0')
          foundGraph = {
            crawl: c.id,
            type,
            totalNodes: estimateNodes(contentLength, type),
            totalEdges: estimateEdges(contentLength, type),
            topRanked: [],
          }
          break
        }
      }

      if (foundGraph) {
        await cache.set(`graph:${foundGraph.crawl}:${type}`, foundGraph)
        analytics.apiCall('/api/graph', Date.now() - start, false)
        return c.json({
          ...foundGraph,
          note: `Graph data not available for ${crawlID}. Showing data from ${foundGraph.crawl}.`,
        })
      }

      return c.json({
        crawl: crawlID,
        type,
        totalNodes: 0,
        totalEdges: 0,
        topRanked: [],
        available: false,
        message: 'Web graph data is not yet available for this crawl. Graph datasets are published periodically by Common Crawl.',
      })
    }

    const contentLength = parseInt(headRes.headers.get('content-length') || '0')
    const graphData: GraphData = {
      crawl: crawlID,
      type,
      totalNodes: estimateNodes(contentLength, type),
      totalEdges: estimateEdges(contentLength, type),
      topRanked: [],
    }

    await cache.set(`graph:${crawlID}:${type}`, graphData)
    analytics.apiCall('/api/graph', Date.now() - start, false)
    return c.json(graphData)
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    analytics.error('/api/graph', 500, message)
    return c.json({ error: message }, 500)
  }
})

// GET /graph/rank — domain ranking lookup
app.get('/rank', async (c) => {
  const start = Date.now()
  const domain = c.req.query('domain')
  if (!domain) return c.json({ error: 'domain parameter required' }, 400)

  const crawl = c.req.query('crawl')
  const type = (c.req.query('type') || 'host') as 'host' | 'domain'
  const cache = new Cache(c.env.KV)
  const cc = new CCClient(cache)
  const analytics = new Analytics(c.env.ANALYTICS)

  try {
    const crawlID = crawl || await cc.getLatestCrawl()
    const cacheKey = `rank:${crawlID}:${type}:${domain}`
    const cached = await cache.get<object>(cacheKey)
    if (cached) {
      analytics.apiCall('/api/graph/rank', Date.now() - start, true)
      return c.json(cached)
    }

    // Check KV for pre-processed rank data for this domain
    const rankData = await cache.get<{ harmonicCentrality: number; pageRank: number; inDegree: number }>(`domain-rank:${crawlID}:${type}:${domain}`)

    if (rankData) {
      const result = {
        domain,
        crawl: crawlID,
        type,
        ...rankData,
      }
      await cache.set(cacheKey, result)
      analytics.apiCall('/api/graph/rank', Date.now() - start, true)
      return c.json(result)
    }

    // Ranking data is only available from pre-processed graph datasets
    // If not in KV, we cannot compute it on the fly
    const result = {
      domain,
      crawl: crawlID,
      type,
      harmonicCentrality: null,
      pageRank: null,
      inDegree: null,
      available: false,
      message: 'Domain ranking data requires pre-processed graph datasets. Run the index-process queue job to populate.',
    }
    analytics.apiCall('/api/graph/rank', Date.now() - start, false)
    return c.json(result)
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    analytics.error('/api/graph/rank', 500, message)
    return c.json({ error: message }, 500)
  }
})

// GET /graph/annotations — GneissWeb quality scores
app.get('/annotations', async (c) => {
  const start = Date.now()
  const crawl = c.req.query('crawl')
  const url = c.req.query('url')
  const domain = c.req.query('domain')
  const cache = new Cache(c.env.KV)
  const cc = new CCClient(cache)
  const analytics = new Analytics(c.env.ANALYTICS)

  try {
    const crawlID = crawl || await cc.getLatestCrawl()

    if (url) {
      // Look up quality annotation for a specific URL
      const cacheKey = `annotation:${crawlID}:url:${url}`
      const cached = await cache.get<object>(cacheKey)
      if (cached) {
        analytics.apiCall('/api/graph/annotations', Date.now() - start, true)
        return c.json(cached)
      }

      const result = {
        url,
        crawl: crawlID,
        qualityScore: null,
        categories: [],
        available: false,
        message: 'GneissWeb quality annotations require pre-processed data. Available for select crawls.',
      }
      analytics.apiCall('/api/graph/annotations', Date.now() - start, false)
      return c.json(result)
    }

    if (domain) {
      // Look up aggregate quality for a domain
      const cacheKey = `annotation:${crawlID}:domain:${domain}`
      const cached = await cache.get<object>(cacheKey)
      if (cached) {
        analytics.apiCall('/api/graph/annotations', Date.now() - start, true)
        return c.json(cached)
      }

      const result = {
        domain,
        crawl: crawlID,
        avgQualityScore: null,
        totalAnnotated: 0,
        available: false,
        message: 'GneissWeb quality annotations require pre-processed data. Available for select crawls.',
      }
      analytics.apiCall('/api/graph/annotations', Date.now() - start, false)
      return c.json(result)
    }

    // General annotations info
    const crawlSlug = crawlID.toLowerCase()
    const annotationsUrl = `${CC_DATA}/contrib/GneissWeb/${crawlSlug}/`
    const headRes = await fetch(annotationsUrl, { method: 'HEAD' })

    const result = {
      crawl: crawlID,
      available: headRes.ok,
      baseUrl: annotationsUrl,
      description: 'GneissWeb quality annotations provide content quality scores for web pages in Common Crawl data.',
      message: headRes.ok
        ? 'GneissWeb annotations are available for this crawl.'
        : 'GneissWeb annotations are not available for this crawl.',
    }

    analytics.apiCall('/api/graph/annotations', Date.now() - start, false)
    return c.json(result)
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : 'Unknown error'
    analytics.error('/api/graph/annotations', 500, message)
    return c.json({ error: message }, 500)
  }
})

// ---- Helpers ----

function estimateNodes(manifestSize: number, type: 'host' | 'domain'): number {
  // Rough estimate from manifest file size
  // Host graph is typically larger than domain graph
  if (manifestSize === 0) return 0
  const base = Math.round((manifestSize * 5) / 30)
  return type === 'host' ? base : Math.round(base * 0.7)
}

function estimateEdges(manifestSize: number, type: 'host' | 'domain'): number {
  const nodes = estimateNodes(manifestSize, type)
  // Average ~15 edges per node in host graph, ~12 in domain graph
  return type === 'host' ? nodes * 15 : nodes * 12
}

export default app
