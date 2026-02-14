import type { Env, QueueMessage, StatsData, GraphData } from './types'
import { CCClient } from './cc'
import { Cache } from './cache'

const CC_DATA = 'https://data.commoncrawl.org'
const CC_INFO = 'https://index.commoncrawl.org/collinfo.json'
const GRAPH_BASE = 'https://data.commoncrawl.org/projects/hyperlinkgraph'

// ---- Producer: enqueue jobs ----

export async function enqueueCacheWarm(queue: Queue, crawl: string): Promise<void> {
  const msg: QueueMessage = { type: 'cache-warm', crawl, timestamp: new Date().toISOString() }
  await queue.send(msg)
}

export async function enqueueIndexProcess(queue: Queue, crawl: string): Promise<void> {
  const msg: QueueMessage = { type: 'index-process', crawl, timestamp: new Date().toISOString() }
  await queue.send(msg)
}

export async function enqueueNotify(queue: Queue): Promise<void> {
  const msg: QueueMessage = { type: 'notify', timestamp: new Date().toISOString() }
  await queue.send(msg)
}

// ---- Consumer: process jobs ----

export async function processQueue(batch: MessageBatch<QueueMessage>, env: Env): Promise<void> {
  const cache = new Cache(env.KV)
  const cc = new CCClient(cache)

  for (const message of batch.messages) {
    const msg = message.body
    try {
      switch (msg.type) {
        case 'cache-warm':
          await handleCacheWarm(cc, cache, msg.crawl!)
          break
        case 'index-process':
          await handleIndexProcess(cc, cache, msg.crawl!)
          break
        case 'notify':
          await handleNotify(cc, cache, env)
          break
      }
      message.ack()
    } catch (err) {
      console.error(`[Queue] Failed to process ${msg.type}:`, err)
      const retry = msg.retry || 0
      if (retry < 3) {
        message.retry({ delaySeconds: Math.pow(2, retry) * 60 })
      } else {
        // Max retries exceeded, ack to avoid infinite loop
        console.error(`[Queue] Giving up on ${msg.type} after ${retry} retries`)
        message.ack()
      }
    }
  }
}

// ---- Handlers ----

async function handleCacheWarm(cc: CCClient, cache: Cache, crawlID: string): Promise<void> {
  // Pre-fetch crawl stats and WARC file listing
  await cc.fetchCrawlStats(crawlID)
  await cc.fetchWARCFileList(crawlID)

  // Pre-compute stats summary from manifest data
  const stats = await cc.fetchCrawlStats(crawlID)
  const listing = await cc.fetchWARCFileList(crawlID)

  // Build a basic StatsData from what we can derive
  const statsData: StatsData = {
    crawl: crawlID,
    totalPages: estimateTotalPages(stats.warcFiles),
    totalDomains: estimateTotalDomains(stats.warcFiles),
    totalSize: stats.estimatedSizeBytes,
    tldDistribution: {},
    mimeDistribution: {},
    statusDistribution: {},
    languageDistribution: {},
  }

  await cache.set(`stats:${crawlID}`, statsData)

  // Sample a few WARC files to build distribution estimates
  const sampleFiles = sampleItems(listing.files, 5)
  const tldCounts: Record<string, number> = {}
  const mimeCounts: Record<string, number> = {}
  const statusCounts: Record<string, number> = {}
  const langCounts: Record<string, number> = {}

  for (const file of sampleFiles) {
    // Extract segment info from filename pattern:
    // crawl-data/CC-MAIN-YYYY-WW/segments/SEG/warc/CC-MAIN-...warc.gz
    const parts = file.split('/')
    if (parts.length >= 4) {
      const segment = parts[3] || 'unknown'
      // Use segment as a proxy for distribution diversity
      tldCounts[segment.substring(0, 8)] = (tldCounts[segment.substring(0, 8)] || 0) + 1
    }
  }

  // Update statsData with sampled distributions
  statsData.tldDistribution = tldCounts
  await cache.set(`stats:${crawlID}`, statsData)
}

async function handleIndexProcess(cc: CCClient, cache: Cache, crawlID: string): Promise<void> {
  // Attempt to fetch and cache web graph summary data
  const graphTypes = ['host', 'domain'] as const
  for (const type of graphTypes) {
    const graphData = await fetchGraphSummary(crawlID, type)
    if (graphData) {
      await cache.set(`graph:${crawlID}:${type}`, graphData)
    }
  }
}

async function handleNotify(cc: CCClient, cache: Cache, env: Env): Promise<void> {
  // Check for new crawls
  const crawls = await cc.listCrawls()
  if (crawls.length === 0) return

  const latestCrawl = crawls[0].id
  const lastKnown = await cache.get<string>('last-known-crawl')

  if (lastKnown !== latestCrawl) {
    await cache.set('last-known-crawl', latestCrawl)

    // Send webhook notification if configured
    if (env.WEBHOOK_URL) {
      try {
        await fetch(env.WEBHOOK_URL, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            event: 'new_crawl',
            crawl: latestCrawl,
            timestamp: new Date().toISOString(),
            total_crawls: crawls.length,
          }),
        })
      } catch (err) {
        console.error('[Notify] Webhook failed:', err)
      }
    }

    // Queue cache warming for the new crawl
    if (env.QUEUE) {
      await enqueueCacheWarm(env.QUEUE, latestCrawl)
      await enqueueIndexProcess(env.QUEUE, latestCrawl)
    }
  }
}

// ---- Cron handler ----

export async function handleScheduled(event: ScheduledEvent, env: Env, ctx: ExecutionContext): Promise<void> {
  const cache = new Cache(env.KV)
  const cc = new CCClient(cache)

  // Determine which cron fired based on the schedule
  // "0 0 * * *" = daily midnight: check for new crawls
  // "0 */6 * * *" = every 6 hours: warm caches
  const hour = new Date(event.scheduledTime).getUTCHours()

  if (hour === 0) {
    // Daily: check for new crawls and notify
    if (env.QUEUE) {
      await enqueueNotify(env.QUEUE)
    } else {
      // No queue — run inline
      await handleNotify(cc, cache, env)
    }
  }

  // Every 6 hours: warm cache for latest crawl
  const latestCrawl = await cc.getLatestCrawl()
  if (env.QUEUE) {
    await enqueueCacheWarm(env.QUEUE, latestCrawl)
  } else {
    await handleCacheWarm(cc, cache, latestCrawl)
  }
}

// ---- Helpers ----

function estimateTotalPages(warcFiles: number): number {
  // Average CC crawl: ~3 billion pages across ~72,000 WARC files
  // ~41,667 pages per WARC file
  return warcFiles * 41_667
}

function estimateTotalDomains(warcFiles: number): number {
  // Average CC crawl: ~35 million unique domains across ~72,000 WARC files
  // ~486 domains per WARC file (with heavy overlap)
  return Math.round(warcFiles * 486 * 0.3) // Rough dedup factor
}

function sampleItems<T>(items: T[], count: number): T[] {
  if (items.length <= count) return items
  const result: T[] = []
  const step = Math.floor(items.length / count)
  for (let i = 0; i < count; i++) {
    result.push(items[i * step])
  }
  return result
}

async function fetchGraphSummary(crawlID: string, type: 'host' | 'domain'): Promise<GraphData | null> {
  // CC web graph data is published per-crawl at a known location
  // Format: https://data.commoncrawl.org/projects/hyperlinkgraph/cc-main-YYYY-WW/host/vertices.txt.gz
  const crawlSlug = crawlID.toLowerCase()
  const verticesUrl = `${GRAPH_BASE}/${crawlSlug}/${type}/vertices.stats.gz`

  try {
    const res = await fetch(verticesUrl, { method: 'HEAD' })
    if (!res.ok) return null

    // Graph stats are large — we store a summary with top-ranked nodes
    // For now, return a placeholder with metadata from the HEAD response
    const contentLength = parseInt(res.headers.get('content-length') || '0')

    return {
      crawl: crawlID,
      type,
      totalNodes: type === 'host' ? estimateGraphNodes(contentLength) : Math.round(estimateGraphNodes(contentLength) * 0.7),
      totalEdges: type === 'host' ? estimateGraphEdges(contentLength) : Math.round(estimateGraphEdges(contentLength) * 0.6),
      topRanked: [],
    }
  } catch {
    return null
  }
}

function estimateGraphNodes(verticesFileSize: number): number {
  // Rough estimate: ~50 bytes per vertex line, compressed ~5x
  return Math.round((verticesFileSize * 5) / 50)
}

function estimateGraphEdges(verticesFileSize: number): number {
  // Typical web graph: ~15 edges per node
  return estimateGraphNodes(verticesFileSize) * 15
}
