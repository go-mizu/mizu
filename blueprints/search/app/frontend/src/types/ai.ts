// AI Mode Types

export type AIMode = 'quick' | 'deep' | 'research'

export interface Citation {
  index: number
  url: string
  title: string
  snippet?: string
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
  response?: AIResponse
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
