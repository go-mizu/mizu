export interface Env {
  KV: KVNamespace
  AUTH_TOKEN?: string
  ENVIRONMENT: string
}

export type HonoEnv = { Bindings: Env }

// --- SSE Types ---

export interface SSEPayload {
  query_str: string
  params: SSEParams
}

export interface SSEParams {
  attachments: string[]
  frontend_context_uuid: string
  frontend_uuid: string
  is_incognito: boolean
  language: string
  last_backend_uuid: string | null
  mode: string
  model_preference: string
  source: string
  sources: string[]
  version: string
}

export interface SessionState {
  csrfToken: string
  cookies: string // serialized cookie header
  createdAt: string
}

// --- App Types ---

export interface SearchResult {
  query: string
  answer: string
  citations: Citation[]
  webResults: WebResult[]
  relatedQueries: string[]
  backendUUID: string
  mode: string
  model: string
  durationMs: number
  createdAt: string
}

export interface Citation {
  url: string
  title: string
  snippet: string
  date?: string
  domain: string
  favicon: string
}

export interface WebResult {
  name: string
  url: string
  snippet: string
  date?: string
}

export interface ThreadMessage {
  role: 'user' | 'assistant'
  content: string
  citations?: Citation[]
  webResults?: WebResult[]
  relatedQueries?: string[]
  backendUUID?: string
  model?: string
  durationMs?: number
  createdAt: string
}

export interface Thread {
  id: string
  title: string
  mode: string
  model: string
  messages: ThreadMessage[]
  createdAt: string
  updatedAt: string
}

export interface ThreadIndex {
  threads: ThreadSummary[]
}

export interface ThreadSummary {
  id: string
  title: string
  mode: string
  model: string
  messageCount: number
  createdAt: string
  updatedAt: string
}

// --- Account Types ---

export interface Account {
  id: string
  email: string
  session: SessionState
  proQueries: number
  status: 'active' | 'exhausted' | 'failed'
  createdAt: string
  lastUsedAt: string
}

export interface AccountIndex {
  accounts: AccountSummary[]
}

export interface AccountSummary {
  id: string
  email: string
  proQueries: number
  status: string
  createdAt: string
}
