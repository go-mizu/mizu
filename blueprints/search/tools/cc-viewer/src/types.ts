export interface Env {
  KV?: KVNamespace
  ENVIRONMENT: string
}

export type HonoEnv = {
  Bindings: Env
}

export interface CDXEntry {
  urlkey: string
  timestamp: string
  url: string
  mime: string
  'mime-detected'?: string
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
  'cdx-api': string
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
  avgWARCSize: number  // actual average from HEAD sampling
}

export interface WARCFileListing {
  files: string[]
  totalFiles: number
}

export interface CDXBrowseResult {
  entries: CDXEntry[]
  crawl: string
  page: number
  totalPages: number
  prefix: string
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

export interface ClusterPageResult {
  entries: ClusterEntry[]
  page: number
  totalPages: number
  crawl: string
}
