export interface Env {
  KV: KVNamespace
  DB: D1Database
  AUTH_TOKEN?: string
  PERPLEXITY_API_KEY?: string
  ACCOUNT_SECRET?: string
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

// --- Thinking Steps ---

export interface ThinkingStep {
  stepType: string       // e.g. 'INITIAL', 'THINKING', 'SEARCH', 'ANSWER', 'FINAL', etc.
  content: string        // text content of the step
  status?: string        // optional status like 'pending', 'complete'
  timestamp?: number     // ms since stream start
}

// --- App Types ---

export interface MediaItem {
  type: 'image' | 'video'
  url: string
  thumbnail?: string
  title?: string
  sourceUrl?: string
  duration?: string
  width?: number
  height?: number
}

export interface SearchResult {
  query: string
  answer: string
  citations: Citation[]
  webResults: WebResult[]
  relatedQueries: string[]
  images: MediaItem[]
  videos: MediaItem[]
  thinkingSteps: ThinkingStep[]
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
  thumbnail?: string
}

export interface WebResult {
  name: string
  url: string
  snippet: string
  date?: string
  thumbnail?: string
}

export interface ThreadMessage {
  role: 'user' | 'assistant'
  content: string
  citations?: Citation[]
  webResults?: WebResult[]
  relatedQueries?: string[]
  images?: MediaItem[]
  videos?: MediaItem[]
  thinkingSteps?: ThinkingStep[]
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

// --- Registration Log ---

export interface RegistrationLog {
  timestamp: string
  event: 'start' | 'email_created' | 'signin_sent' | 'email_received' | 'auth_complete' | 'account_saved' | 'relogin' | 'disabled' | 'error'
  message: string
  provider?: string
  email?: string
  accountId?: string
  durationMs?: number
  error?: string
}

// --- OG Cache ---

export interface OGData {
  title: string
  description: string
  image: string
  siteName: string
}

// --- Account Types ---

export interface Account {
  id: string
  email: string
  emailProvider: string            // 'mail.tm', 'mail.gw', etc.
  emailPasswordEnc: string         // AES-256-GCM encrypted (iv_hex:ct_base64)
  session: SessionState
  proQueries: number
  status: 'active' | 'exhausted' | 'failed' | 'disabled'
  createdAt: string
  lastUsedAt: string
  disabledAt?: string
  disableReason?: string
  totalQueriesUsed: number         // lifetime counter
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
