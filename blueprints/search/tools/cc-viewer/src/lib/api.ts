// Types matching worker/types.ts
export interface CDXEntry {
  urlkey: string
  timestamp: string
  url: string
  mime: string
  "mime-detected"?: string
  status: string
  digest: string
  length: string
  offset: string
  filename: string
  languages?: string
  charset?: string
  redirect?: string
}

export interface Crawl {
  id: string
  name: string
  "cdx-api": string
  timegate: string
  from?: string
  to?: string
}

export interface WARCRecord {
  warcType: string
  targetURI: string
  date: string
  recordID: string
  httpStatus: number
  httpHeaders: Record<string, string>
  body: string
  contentType: string
  contentLength: number
}

export interface DomainStats {
  totalPages: number
  uniquePaths: number
  totalSize: number
  statusCounts: Record<string, number>
  mimeCounts: Record<string, number>
}

export interface DomainSummary {
  domain: string
  pages: number
  size: number
}

export interface URLGroup {
  url: string
  path: string
  count: number
  latestTimestamp: string
  latestStatus: string
  latestMime: string
  latestLength: string
  entries: CDXEntry[]
}

export interface CrawlStats {
  segments: number
  warcFiles: number
  indexFiles: number
  estimatedSizeBytes: number
  avgWARCSize: number
}

export interface WARCFileListing {
  files: string[]
  totalFiles: number
}

export interface ClusterMeta {
  totalDomains: number
  totalPages: number
  totalEntries: number
  entriesPages: number
}

export interface ClusterEntry {
  surtKey: string
  domain: string
  cdxFile: string
  pageNum: number
}

export interface ClusterBrowseResult {
  domains: DomainSummary[]
  page: number
  totalPages: number
  crawl: string
}

export interface StatsData {
  crawl: string
  totalPages: number
  totalDomains: number
  totalSize: number
  tldDistribution: Record<string, number>
  mimeDistribution: Record<string, number>
  statusDistribution: Record<string, number>
  languageDistribution: Record<string, number>
}

export interface GraphData {
  crawl: string
  type: "host" | "domain"
  totalNodes: number
  totalEdges: number
  topRanked: Array<{
    domain: string
    harmonicCentrality: number
    pageRank: number
  }>
}

export interface GraphRankResult {
  domain: string
  harmonicCentrality: number
  pageRank: number
  inDegree: number
  outDegree: number
}

export interface AnnotationData {
  domain: string
  categories: string[]
  language: string
  quality: number
  lastChecked: string
}

export interface NewsEntry {
  url: string
  timestamp: string
  status: string
  mime: string
  filename: string
  offset: string
  length: string
}

export interface SearchResult {
  type: "url" | "domain" | "prefix"
  query: string
  entries: CDXEntry[]
  url?: string
  domain?: string
  stats?: DomainStats
  crawl: string
  totalPages?: number
  groups?: URLGroup[]
}

export interface PopularData {
  type: string
  period: string
  items: Array<{ name: string; count: number }>
  total: number
  source: string
}

export interface UsageData {
  period: string
  totalRequests: number
  cacheHitRate: number
  endpoints: Array<{ endpoint: string; requests: number; avgLatencyMs: number }>
  kvReadsEstimate: number
  timestamp: string
}

export interface DomainResult {
  entries: CDXEntry[]
  crawl: string
  page: number
  totalPages: number
  stats: DomainStats
  groups?: URLGroup[]
}

export interface URLResult {
  entries: CDXEntry[]
  crawl: string
  groups?: URLGroup[]
}

export interface CrawlDetailResult {
  crawl: Crawl
  stats: CrawlStats
}

export interface CrawlFilesResult {
  crawl: string
  files: string[]
  page: number
  perPage: number
  totalPages: number
  totalFiles: number
}

export interface CrawlCDXResult {
  crawl: string
  entries: CDXEntry[]
  page: number
  totalPages: number
  prefix: string
  clusterEntries?: ClusterEntry[]
}

export interface DomainsResult {
  domains: DomainSummary[]
  page: number
  totalPages: number
  crawl: string
}

export interface StatsTrend {
  crawl: string
  date: string
  warcFiles: number
  segments: number
  indexFiles: number
  estimatedSizeBytes: number
  estimatedSizeTB: number
  estimatedPages: number
}

export interface NewsDatesResult {
  dates: string[]
}

export interface NewsResult {
  date: string
  entries: NewsEntry[]
  page: number
  totalPages: number
  total: number
  topDomains?: Array<{ domain: string; count: number }>
  statusCounts?: Record<string, number>
}

// ---

const BASE = "/api"

async function get<T>(
  path: string,
  params?: Record<string, string | undefined>
): Promise<T> {
  const url = new URL(BASE + path, window.location.origin)
  if (params) {
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined && v !== "") {
        url.searchParams.set(k, v)
      }
    })
  }
  const res = await fetch(url.toString())
  if (!res.ok) {
    const text = await res.text().catch(() => "")
    let message = `API error: ${res.status}`
    try {
      const json = JSON.parse(text)
      if (json.error) message = json.error
    } catch {
      if (text) message = text
    }
    throw new Error(message)
  }
  return res.json()
}

export const api = {
  getCrawls: () =>
    get<{ crawls: Crawl[]; total: number }>("/crawls").then((r) => r.crawls),

  getCrawl: (id: string) =>
    get<CrawlDetailResult>(`/crawls/${encodeURIComponent(id)}`),

  getCrawlFiles: (id: string, page?: number) =>
    get<CrawlFilesResult>(`/crawls/${encodeURIComponent(id)}/files`, {
      page: String(page ?? 0),
    }),

  getCrawlCDX: (id: string, prefix: string, page?: number) =>
    get<CrawlCDXResult>(`/crawls/${encodeURIComponent(id)}/cdx`, {
      prefix,
      page: String(page ?? 0),
    }),

  getDomains: (crawl?: string, page?: number) =>
    get<DomainsResult>("/domains", {
      crawl,
      page: String(page ?? 0),
    }),

  getDomain: (domain: string, crawl?: string, page?: number) =>
    get<DomainResult>("/domains/lookup", {
      domain,
      crawl,
      page: String(page ?? 0),
    }),

  lookupURL: (url: string, crawl?: string) =>
    get<URLResult>("/url", { url, crawl }),

  getView: (file: string, offset: number, length: number) =>
    get<WARCRecord>("/view", {
      file,
      offset: String(offset),
      length: String(length),
    }),

  getWAT: (file: string, offset: number, length: number) =>
    get<WARCRecord>("/view/wat", {
      file,
      offset: String(offset),
      length: String(length),
    }),

  getWET: (file: string, offset: number, length: number) =>
    get<WARCRecord>("/view/wet", {
      file,
      offset: String(offset),
      length: String(length),
    }),

  getRobots: (domain: string, crawl?: string) =>
    get<{ found: boolean; domain: string; crawl: string; record?: WARCRecord; captures?: number }>(
      "/view/robots",
      { domain, crawl }
    ),

  getStats: (crawl?: string) => get<StatsData>("/stats", { crawl }),

  getStatsTrends: () =>
    get<{ trends: StatsTrend[]; totalCrawls: number; latestCrawl: string }>(
      "/stats/trends"
    ).then((r) => r.trends),

  getGraph: (crawl?: string, type?: string) =>
    get<GraphData>("/graph", { crawl, type }),

  getGraphRank: (domain: string) =>
    get<GraphRankResult>("/graph/rank", { domain }),

  getAnnotations: (domain: string) =>
    get<AnnotationData>("/graph/annotations", { domain }),

  getNews: (date?: string, page?: number) =>
    get<NewsResult>("/news", {
      date,
      page: String(page ?? 0),
    }),

  getNewsDates: () =>
    get<{
      dates: Array<{ raw: string; formatted: string; year: string; month: string; day: string }>
      totalDates: number
      showing: number
      byMonth: Record<string, string[]>
    }>("/news/dates").then((r) => ({
      dates: r.dates.map((d) => d.raw),
    })),

  search: (q: string, type?: string) =>
    get<SearchResult>("/search", { q, type }),

  getPopular: () => get<PopularData>("/analytics/popular"),

  getUsage: () => get<UsageData>("/analytics/usage"),
}
