export interface Env {
  DB: D1Database
  KV: KVNamespace
  ASSETS: Fetcher
  ENRICH_QUEUE: Queue
  ENVIRONMENT: string
}

export type HonoEnv = { Bindings: Env }

export interface OLSearchResult {
  key: string
  title: string
  author_name?: string[]
  first_publish_year?: number
  cover_i?: number
  isbn?: string[]
  subject?: string[]
  publisher?: string[]
  language?: string[]
  ratings_average?: number
  ratings_count?: number
  edition_count?: number
}
