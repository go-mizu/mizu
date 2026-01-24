// AI Mode Types

export type AIMode = 'quick' | 'deep' | 'research' | 'deepsearch'

export type ModelCapability = 'text' | 'vision' | 'embeddings' | 'voice'

export interface ModelInfo {
  id: string
  provider: string
  name: string
  description?: string
  capabilities: ModelCapability[]
  context_size: number
  speed: 'fast' | 'balanced' | 'thorough'
  is_default?: boolean
  available: boolean
}

export interface Citation {
  index: number
  url: string
  title: string
  snippet?: string
  domain?: string        // Source domain for badge display
  favicon?: string       // Source favicon URL
  other_sources?: number // Count of other sources (for "+N" display)
}

export interface ImageResult {
  url: string
  thumbnail_url: string
  title: string
  source_url: string
  source_domain: string
  width: number
  height: number
}

export interface RelatedQuestion {
  text: string
  category?: 'deeper' | 'related' | 'practical' | 'comparison' | 'current'
}

export interface AIMessage {
  id: string
  session_id: string
  role: 'user' | 'assistant'
  content: string
  mode?: string
  citations?: Citation[]
  created_at: string
}

export interface AISession {
  id: string
  title: string
  messages?: AIMessage[]
  created_at: string
  updated_at: string
}

export interface AIResponse {
  text: string
  mode: AIMode
  citations: Citation[]
  follow_ups: string[]
  related_questions?: RelatedQuestion[]
  images?: ImageResult[]
  session_id: string
  sources_used: number
  thinking_steps?: string[]
}

export interface AIStreamEvent {
  type: 'start' | 'token' | 'citation' | 'thinking' | 'search' | 'done' | 'error'
  content?: string
  citation?: Citation
  thinking?: string
  query?: string
  error?: string
  response?: {
    text: string
    mode: AIMode
    citations: Citation[]
    follow_ups: string[]
    related_questions?: RelatedQuestion[]
    images?: ImageResult[]
    session_id: string
    sources_used: number
  }
}

export interface AIModeInfo {
  id: AIMode
  name: string
  description: string
  icon: string
  available: boolean
}

// Canvas types
export type BlockType = 'text' | 'ai_response' | 'note' | 'citation' | 'heading' | 'divider' | 'code'

export interface CanvasBlock {
  id: string
  canvas_id: string
  type: BlockType
  content: string
  meta?: Record<string, unknown>
  order: number
  created_at: string
}

export interface Canvas {
  id: string
  session_id: string
  title: string
  blocks?: CanvasBlock[]
  created_at: string
  updated_at: string
}

export type ExportFormat = 'markdown' | 'html' | 'json'
