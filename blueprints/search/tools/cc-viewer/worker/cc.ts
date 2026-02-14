import { gunzipSync } from 'node:zlib'
import type { CDXEntry, Crawl, WARCRecord, DomainStats, URLGroup, CrawlStats, WARCFileListing, DomainSummary, ClusterMeta, ClusterEntry, ClusterBrowseResult, ClusterPageResult, NewsEntry, StatsData, GraphData } from './types'
import { Cache } from './cache'

const CC_CDX = 'https://index.commoncrawl.org'
const CC_DATA = 'https://data.commoncrawl.org'
const CC_INFO = 'https://index.commoncrawl.org/collinfo.json'

const LATEST_CRAWL = 'CC-MAIN-2026-04'

// Max decompressed body we'll keep (512 KB) — prevents OOM on large pages
const MAX_BODY_SIZE = 512 * 1024

export class CCClient {
  private cache: Cache

  constructor(cache: Cache) {
    this.cache = cache
  }

  async listCrawls(): Promise<Crawl[]> {
    const cached = await this.cache.get<Crawl[]>('crawls')
    if (cached) return cached

    const res = await fetch(CC_INFO)
    if (!res.ok) throw new Error(`Failed to fetch crawl list: ${res.status}`)
    const crawls = await res.json() as Crawl[]
    await this.cache.set('crawls', crawls)
    return crawls
  }

  async getLatestCrawl(): Promise<string> {
    try {
      const crawls = await this.listCrawls()
      if (crawls.length > 0) return crawls[0].id
    } catch { /* fallback */ }
    return LATEST_CRAWL
  }

  async lookupURL(url: string, crawlID?: string): Promise<{ entries: CDXEntry[], crawl: string }> {
    const crawl = crawlID || await this.getLatestCrawl()
    const cacheKey = `url:${crawl}:${url}`
    const cached = await this.cache.get<CDXEntry[]>(cacheKey)
    if (cached) return { entries: cached, crawl }

    const endpoint = `${CC_CDX}/${crawl}-index?url=${encodeURIComponent(url)}&output=json`
    const res = await fetch(endpoint)
    if (!res.ok) {
      if (res.status === 404) return { entries: [], crawl }
      throw new Error(`CDX API error: ${res.status}`)
    }

    const text = await res.text()
    const entries = parseCDXResponse(text)
    await this.cache.set(cacheKey, entries)
    return { entries, crawl }
  }

  async lookupDomain(domain: string, crawlID?: string, page = 0): Promise<{ entries: CDXEntry[], crawl: string, totalPages: number }> {
    const crawl = crawlID || await this.getLatestCrawl()
    const cacheKey = `domain:${crawl}:${domain}:${page}`
    const cached = await this.cache.get<{ entries: CDXEntry[], totalPages: number }>(cacheKey)
    if (cached) return { ...cached, crawl }

    // Use matchType=domain to include all subdomains (same as CLI)
    const base = `${CC_CDX}/${crawl}-index?url=${encodeURIComponent(domain)}&matchType=domain&output=json`
    const countRes = await fetch(`${base}&showNumPages=true`)
    let totalPages = 1
    if (countRes.ok) {
      const countText = await countRes.text()
      try {
        const parsed = JSON.parse(countText)
        totalPages = typeof parsed === 'number' ? parsed : (parsed.pages || 1)
      } catch {
        totalPages = parseInt(countText) || 1
      }
    }

    const endpoint = `${base}&page=${page}`
    const res = await fetch(endpoint)
    if (!res.ok) {
      if (res.status === 404) return { entries: [], crawl, totalPages: 0 }
      throw new Error(`CDX API error: ${res.status}`)
    }

    const text = await res.text()
    const entries = parseCDXResponse(text)
    await this.cache.set(cacheKey, { entries, totalPages })
    return { entries, crawl, totalPages }
  }

  async fetchWARC(filename: string, offset: number, length: number): Promise<WARCRecord> {
    const cacheKey = `warc:${filename}:${offset}:${length}`
    const cached = await this.cache.get<WARCRecord>(cacheKey)
    if (cached) return cached

    const url = `${CC_DATA}/${filename}`
    const end = offset + length - 1
    const res = await fetch(url, {
      headers: { 'Range': `bytes=${offset}-${end}` }
    })

    if (!res.ok && res.status !== 206) {
      throw new Error(`WARC fetch failed: ${res.status}`)
    }

    const compressed = await res.arrayBuffer()
    const record = await parseWARCRecord(new Uint8Array(compressed))
    await this.cache.set(cacheKey, record)
    return record
  }

  async fetchCrawlStats(crawlID: string): Promise<CrawlStats> {
    const cacheKey = `crawl-stats:${crawlID}`
    const cached = await this.cache.get<CrawlStats>(cacheKey)
    if (cached) return cached

    // Fetch manifests in parallel + HEAD one WARC for actual size
    const [segRes, warcRes, idxRes] = await Promise.all([
      fetch(`${CC_DATA}/crawl-data/${crawlID}/segment.paths.gz`),
      fetch(`${CC_DATA}/crawl-data/${crawlID}/warc.paths.gz`),
      fetch(`${CC_DATA}/crawl-data/${crawlID}/cc-index.paths.gz`),
    ])

    const countLines = async (res: Response): Promise<string[]> => {
      if (!res.ok) return []
      const compressed = new Uint8Array(await res.arrayBuffer())
      const text = await decompressText(compressed)
      return text.trim().split('\n').filter(l => l.length > 0)
    }

    const [segLines, warcLines, idxLines] = await Promise.all([
      countLines(segRes),
      countLines(warcRes),
      countLines(idxRes),
    ])

    const segments = segLines.length
    const warcFiles = warcLines.length
    const indexFiles = idxLines.length

    // HEAD 3 sample WARC files for accurate average size
    let avgWARCSize = 893_000_000 // fallback: ~893 MB (known CC average)
    if (warcLines.length > 0) {
      const sampleIndices = [0, Math.floor(warcLines.length / 2), warcLines.length - 1]
      const sizes = await Promise.all(sampleIndices.map(async (i) => {
        try {
          const headRes = await fetch(`${CC_DATA}/${warcLines[i]}`, { method: 'HEAD' })
          const cl = headRes.headers.get('content-length')
          return cl ? parseInt(cl) : 0
        } catch { return 0 }
      }))
      const validSizes = sizes.filter(s => s > 0)
      if (validSizes.length > 0) {
        avgWARCSize = Math.round(validSizes.reduce((a, b) => a + b, 0) / validSizes.length)
      }
    }

    const estimatedSizeBytes = warcFiles * avgWARCSize

    const stats: CrawlStats = { segments, warcFiles, indexFiles, estimatedSizeBytes, avgWARCSize }
    await this.cache.set(cacheKey, stats)
    return stats
  }

  async fetchWARCFileList(crawlID: string): Promise<WARCFileListing> {
    const cacheKey = `warc-files:${crawlID}`
    const cached = await this.cache.get<WARCFileListing>(cacheKey)
    if (cached) return cached

    const res = await fetch(`${CC_DATA}/crawl-data/${crawlID}/warc.paths.gz`)
    if (!res.ok) throw new Error(`Failed to fetch WARC manifest: ${res.status}`)

    const compressed = new Uint8Array(await res.arrayBuffer())
    const text = await decompressText(compressed)
    const files = text.trim().split('\n').filter(l => l.length > 0)

    const listing: WARCFileListing = { files, totalFiles: files.length }
    await this.cache.set(cacheKey, listing)
    return listing
  }

  async browseCDX(crawlID: string, prefix: string, page = 0): Promise<{ entries: CDXEntry[], totalPages: number }> {
    const cacheKey = `cdx-browse:${crawlID}:${prefix}:${page}`
    const cached = await this.cache.get<{ entries: CDXEntry[], totalPages: number }>(cacheKey)
    if (cached) return cached

    const base = `${CC_CDX}/${crawlID}-index?url=${encodeURIComponent(prefix)}&matchType=prefix&output=json`

    // Get page count
    const countRes = await fetch(`${base}&showNumPages=true`)
    let totalPages = 1
    if (countRes.ok) {
      const countText = await countRes.text()
      try {
        const parsed = JSON.parse(countText)
        totalPages = typeof parsed === 'number' ? parsed : (parsed.pages || 1)
      } catch {
        totalPages = parseInt(countText) || 1
      }
    }

    const res = await fetch(`${base}&page=${page}`)
    if (!res.ok) {
      if (res.status === 404) {
        const result = { entries: [], totalPages: 0 }
        await this.cache.set(cacheKey, result)
        return result
      }
      throw new Error(`CDX API error: ${res.status}`)
    }

    const text = await res.text()
    const entries = parseCDXResponse(text)
    const result = { entries, totalPages }
    await this.cache.set(cacheKey, result)
    return result
  }

  async getCrawlDetail(crawlID: string): Promise<Crawl | null> {
    const crawls = await this.listCrawls()
    return crawls.find(c => c.id === crawlID) || null
  }

  computeDomainStats(entries: CDXEntry[]): DomainStats {
    const paths = new Set<string>()
    const statusCounts: Record<string, number> = {}
    const mimeCounts: Record<string, number> = {}
    let totalSize = 0

    for (const e of entries) {
      try {
        const u = new URL(e.url)
        paths.add(u.pathname)
      } catch { /* skip */ }
      const s = e.status || 'unknown'
      statusCounts[s] = (statusCounts[s] || 0) + 1
      const m = e.mime || 'unknown'
      mimeCounts[m] = (mimeCounts[m] || 0) + 1
      totalSize += parseInt(e.length) || 0
    }

    return {
      totalPages: entries.length,
      uniquePaths: paths.size,
      totalSize,
      statusCounts,
      mimeCounts
    }
  }

  async getClusterMeta(crawlID: string): Promise<ClusterMeta | null> {
    return this.cache.get<ClusterMeta>(`cluster:${crawlID}:meta`)
  }

  async getClusterDomains(crawlID: string, page = 0): Promise<ClusterBrowseResult> {
    const meta = await this.getClusterMeta(crawlID)
    if (!meta) return { domains: [], page, totalPages: 0, crawl: crawlID }
    const domains = await this.cache.get<DomainSummary[]>(`cluster:${crawlID}:domains:${page}`) || []
    return { domains, page, totalPages: meta.totalPages, crawl: crawlID }
  }

  async getClusterEntries(crawlID: string, page = 0): Promise<ClusterPageResult> {
    const meta = await this.getClusterMeta(crawlID)
    if (!meta) return { entries: [], page, totalPages: 0, crawl: crawlID }
    const entries = await this.cache.get<ClusterEntry[]>(`cluster:${crawlID}:entries:${page}`) || []
    return { entries, page, totalPages: meta.entriesPages, crawl: crawlID }
  }

  async fetchWATRecord(filename: string, offset: number, length: number): Promise<WARCRecord> {
    // WAT files mirror WARC layout but under /wat/ instead of /warc/
    const watFilename = filename.replace('/warc/', '/wat/').replace('.warc.gz', '.warc.wat.gz')
    const cacheKey = `wat:${watFilename}:${offset}:${length}`
    const cached = await this.cache.get<WARCRecord>(cacheKey)
    if (cached) return cached

    const url = `${CC_DATA}/${watFilename}`
    const end = offset + length - 1
    const res = await fetch(url, {
      headers: { 'Range': `bytes=${offset}-${end}` }
    })

    if (!res.ok && res.status !== 206) {
      throw new Error(`WAT fetch failed: ${res.status}`)
    }

    const compressed = await res.arrayBuffer()
    const record = await parseWARCRecord(new Uint8Array(compressed))
    await this.cache.set(cacheKey, record)
    return record
  }

  async fetchWETRecord(filename: string, offset: number, length: number): Promise<WARCRecord> {
    // WET files mirror WARC layout but under /wet/ instead of /warc/
    const wetFilename = filename.replace('/warc/', '/wet/').replace('.warc.gz', '.warc.wet.gz')
    const cacheKey = `wet:${wetFilename}:${offset}:${length}`
    const cached = await this.cache.get<WARCRecord>(cacheKey)
    if (cached) return cached

    const url = `${CC_DATA}/${wetFilename}`
    const end = offset + length - 1
    const res = await fetch(url, {
      headers: { 'Range': `bytes=${offset}-${end}` }
    })

    if (!res.ok && res.status !== 206) {
      throw new Error(`WET fetch failed: ${res.status}`)
    }

    const compressed = await res.arrayBuffer()
    const record = await parseWARCRecord(new Uint8Array(compressed))
    await this.cache.set(cacheKey, record)
    return record
  }

  async fetchRobotsTxt(domain: string, crawlID?: string): Promise<{ body: string; crawl: string; found: boolean }> {
    const crawl = crawlID || await this.getLatestCrawl()
    const cacheKey = `robots:${crawl}:${domain}`
    const cached = await this.cache.get<{ body: string; found: boolean }>(cacheKey)
    if (cached) return { ...cached, crawl }

    // Look up robots.txt in CDX
    const robotsUrl = `https://${domain}/robots.txt`
    const { entries } = await this.lookupURL(robotsUrl, crawl)

    if (entries.length === 0) {
      // Try http variant
      const httpUrl = `http://${domain}/robots.txt`
      const httpResult = await this.lookupURL(httpUrl, crawl)
      if (httpResult.entries.length === 0) {
        const result = { body: '', found: false }
        await this.cache.set(cacheKey, result)
        return { ...result, crawl }
      }
      // Use the latest successful entry
      const entry = httpResult.entries.find(e => e.status === '200') || httpResult.entries[0]
      const record = await this.fetchWARC(entry.filename, parseInt(entry.offset), parseInt(entry.length))
      const result = { body: record.body, found: true }
      await this.cache.set(cacheKey, result)
      return { ...result, crawl }
    }

    const entry = entries.find(e => e.status === '200') || entries[0]
    const record = await this.fetchWARC(entry.filename, parseInt(entry.offset), parseInt(entry.length))
    const result = { body: record.body, found: true }
    await this.cache.set(cacheKey, result)
    return { ...result, crawl }
  }

  async lookupNews(date: string, page = 0, limit = 50): Promise<{ entries: NewsEntry[]; totalPages: number }> {
    const cacheKey = `news:${date}:${page}:${limit}`
    const cached = await this.cache.get<{ entries: NewsEntry[]; totalPages: number }>(cacheKey)
    if (cached) return cached

    // CC-NEWS uses CDX API with matchType=prefix on the news crawl
    // CC-NEWS crawl entries are dated YYYYMMDD in timestamps
    const base = `${CC_CDX}/CC-NEWS-index?url=*&output=json&filter=timestamp:${date}*`
    const countRes = await fetch(`${base}&showNumPages=true`)
    let totalPages = 1
    if (countRes.ok) {
      const countText = await countRes.text()
      try {
        const parsed = JSON.parse(countText)
        totalPages = typeof parsed === 'number' ? parsed : (parsed.pages || 1)
      } catch {
        totalPages = parseInt(countText) || 1
      }
    }

    const res = await fetch(`${base}&page=${page}&limit=${limit}`)
    if (!res.ok) {
      if (res.status === 404) {
        const result = { entries: [] as NewsEntry[], totalPages: 0 }
        await this.cache.set(cacheKey, result)
        return result
      }
      throw new Error(`CC-NEWS CDX API error: ${res.status}`)
    }

    const text = await res.text()
    const cdxEntries = parseCDXResponse(text)
    const entries: NewsEntry[] = cdxEntries.map(e => ({
      url: e.url,
      timestamp: e.timestamp,
      status: e.status,
      mime: e.mime,
      filename: e.filename,
      offset: e.offset,
      length: e.length,
    }))

    const result = { entries, totalPages }
    await this.cache.set(cacheKey, result)
    return result
  }

  async listNewsDates(): Promise<string[]> {
    const cacheKey = 'news:dates'
    const cached = await this.cache.get<string[]>(cacheKey)
    if (cached) return cached

    // Fetch the CC-NEWS WARC paths to extract available dates
    const res = await fetch(`${CC_DATA}/crawl-data/CC-NEWS/warc.paths.gz`)
    if (!res.ok) {
      // Fallback: generate recent dates
      const dates: string[] = []
      const now = new Date()
      for (let i = 0; i < 30; i++) {
        const d = new Date(now.getTime() - i * 86400000)
        dates.push(d.toISOString().substring(0, 10).replace(/-/g, ''))
      }
      return dates
    }

    const compressed = new Uint8Array(await res.arrayBuffer())
    const text = await decompressText(compressed)
    const lines = text.trim().split('\n').filter(l => l.length > 0)

    // Extract dates from filenames like: crawl-data/CC-NEWS/2026/01/CC-NEWS-20260115...warc.gz
    const dateSet = new Set<string>()
    for (const line of lines) {
      const match = line.match(/CC-NEWS-(\d{8})/)
      if (match) dateSet.add(match[1])
    }

    const dates = Array.from(dateSet).sort().reverse()
    await this.cache.set(cacheKey, dates)
    return dates
  }

  async getStats(crawlID: string): Promise<StatsData | null> {
    return this.cache.get<StatsData>(`stats:${crawlID}`)
  }

  async getGraph(crawlID: string, type: 'host' | 'domain' = 'host'): Promise<GraphData | null> {
    return this.cache.get<GraphData>(`graph:${crawlID}:${type}`)
  }

  async lookupURLAcrossCrawls(url: string, maxCrawls = 5): Promise<Array<{ crawl: string; entries: CDXEntry[] }>> {
    const cacheKey = `url-across:${url}:${maxCrawls}`
    const cached = await this.cache.get<Array<{ crawl: string; entries: CDXEntry[] }>>(cacheKey)
    if (cached) return cached

    const crawls = await this.listCrawls()
    const targets = crawls.slice(0, maxCrawls)

    const results = await Promise.all(
      targets.map(async (c) => {
        try {
          const { entries } = await this.lookupURL(url, c.id)
          return { crawl: c.id, entries }
        } catch {
          return { crawl: c.id, entries: [] }
        }
      })
    )

    const nonEmpty = results.filter(r => r.entries.length > 0)
    await this.cache.set(cacheKey, nonEmpty)
    return nonEmpty
  }
}

export function surtToDomain(surtHost: string): string {
  return surtHost.split(',').reverse().join('.')
}

export function groupByURL(entries: CDXEntry[]): URLGroup[] {
  const map = new Map<string, URLGroup>()
  for (const e of entries) {
    let path = e.url
    try {
      const u = new URL(e.url)
      path = u.pathname + (u.search || '')
    } catch { /* keep full */ }

    const existing = map.get(e.url)
    if (existing) {
      existing.count++
      existing.entries.push(e)
      // Keep latest (highest timestamp)
      if (e.timestamp > existing.latestTimestamp) {
        existing.latestTimestamp = e.timestamp
        existing.latestStatus = e.status
        existing.latestMime = e.mime
        existing.latestLength = e.length
      }
    } else {
      map.set(e.url, {
        url: e.url,
        path,
        count: 1,
        latestTimestamp: e.timestamp,
        latestStatus: e.status,
        latestMime: e.mime,
        latestLength: e.length,
        entries: [e],
      })
    }
  }
  // Sort by capture count descending, then path alphabetically
  return Array.from(map.values()).sort((a, b) => b.count - a.count || a.path.localeCompare(b.path))
}

function parseCDXResponse(text: string): CDXEntry[] {
  const lines = text.trim().split('\n').filter(l => l.length > 0)
  const entries: CDXEntry[] = []
  for (const line of lines) {
    try {
      entries.push(JSON.parse(line))
    } catch { /* skip malformed */ }
  }
  return entries
}

async function decompressText(compressed: Uint8Array): Promise<string> {
  const ds = new DecompressionStream('gzip')
  const writer = ds.writable.getWriter()
  const reader = ds.readable.getReader()
  writer.write(compressed as unknown as BufferSource).catch(() => {})
  writer.close().catch(() => {})
  let text = ''
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    text += new TextDecoder().decode(value)
  }
  return text
}

async function decompressGzip(compressed: Uint8Array, limit: number): Promise<{ data: Uint8Array; truncated: boolean }> {
  // Use Node.js zlib — CF Worker DecompressionStream throws "Memory limit
  // would be exceeded before EOF" on certain valid gzip streams.
  try {
    const buf = gunzipSync(Buffer.from(compressed))
    if (buf.byteLength > limit) {
      return { data: new Uint8Array(buf.buffer, buf.byteOffset, limit), truncated: true }
    }
    return { data: new Uint8Array(buf.buffer, buf.byteOffset, buf.byteLength), truncated: false }
  } catch {
    return { data: new Uint8Array(0), truncated: true }
  }
}

async function parseWARCRecord(compressed: Uint8Array): Promise<WARCRecord> {
  const hardLimit = MAX_BODY_SIZE + 8192 // body cap + generous header room
  const { data: decompressed, truncated } = await decompressGzip(compressed, hardLimit)

  const text = new TextDecoder('utf-8').decode(decompressed)
  const record: WARCRecord = {
    warcType: '',
    targetURI: '',
    date: '',
    recordID: '',
    httpStatus: 0,
    httpHeaders: {},
    body: '',
    contentType: '',
    contentLength: 0,
  }

  const doubleCRLF = '\r\n\r\n'
  const firstSplit = text.indexOf(doubleCRLF)
  if (firstSplit === -1) {
    record.body = text
    return record
  }

  // Parse WARC headers
  const warcHeaders = text.substring(0, firstSplit)
  for (const line of warcHeaders.split('\r\n')) {
    const colon = line.indexOf(':')
    if (colon === -1) continue
    const key = line.substring(0, colon).trim().toLowerCase()
    const val = line.substring(colon + 1).trim()
    if (key === 'warc-type') record.warcType = val
    else if (key === 'warc-target-uri') record.targetURI = val
    else if (key === 'warc-date') record.date = val
    else if (key === 'warc-record-id') record.recordID = val
    else if (key === 'content-length') record.contentLength = parseInt(val) || 0
  }

  const rest = text.substring(firstSplit + doubleCRLF.length)

  if (record.warcType === 'response') {
    const httpSplit = rest.indexOf(doubleCRLF)
    if (httpSplit !== -1) {
      const httpHeaderBlock = rest.substring(0, httpSplit)
      const httpLines = httpHeaderBlock.split('\r\n')

      if (httpLines.length > 0) {
        const statusMatch = httpLines[0].match(/HTTP\/[\d.]+ (\d+)/)
        if (statusMatch) record.httpStatus = parseInt(statusMatch[1])
      }

      for (let i = 1; i < httpLines.length; i++) {
        const colon = httpLines[i].indexOf(':')
        if (colon === -1) continue
        const key = httpLines[i].substring(0, colon).trim()
        const val = httpLines[i].substring(colon + 1).trim()
        record.httpHeaders[key] = val
        if (key.toLowerCase() === 'content-type') record.contentType = val
      }

      record.body = rest.substring(httpSplit + doubleCRLF.length)
    } else {
      record.body = rest
    }
  } else {
    record.body = rest
  }

  const fullBodyLen = record.contentLength || record.body.length
  let bodyTruncated = truncated
  if (record.body.length > MAX_BODY_SIZE) {
    record.body = record.body.substring(0, MAX_BODY_SIZE)
    bodyTruncated = true
  }
  record.contentLength = fullBodyLen
  if (bodyTruncated) {
    record.httpHeaders['X-CC-Viewer-Truncated'] = `true (body capped at ${MAX_BODY_SIZE} bytes, original ~${fullBodyLen})`
  }

  return record
}

export function formatTimestamp(ts: string): string {
  if (!ts || ts.length < 14) return ts
  // CDX timestamp format: 20260115123456
  const y = ts.substring(0, 4)
  const m = ts.substring(4, 6)
  const d = ts.substring(6, 8)
  const h = ts.substring(8, 10)
  const min = ts.substring(10, 12)
  const s = ts.substring(12, 14)
  return `${y}-${m}-${d} ${h}:${min}:${s}`
}

export function crawlToDate(crawlID: string): string {
  // CC-MAIN-2026-04 → Jan 2026
  const match = crawlID.match(/CC-MAIN-(\d{4})-(\d{2})/)
  if (!match) return crawlID
  const year = match[1]
  const week = parseInt(match[2])
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
  const monthIdx = Math.min(Math.floor((week - 1) / 4.33), 11)
  return `${months[monthIdx]} ${year}`
}

export function statusClass(status: string): string {
  const code = parseInt(status)
  if (code >= 200 && code < 300) return 'st-ok'
  if (code >= 300 && code < 400) return 'st-redirect'
  return 'st-error'
}
