import { api } from './client'
import type {
  AIMode,
  AISession,
  AIResponse,
  AIStreamEvent,
  AIModeInfo,
  Canvas,
  CanvasBlock,
  BlockType,
  ExportFormat,
} from '../types/ai'

export interface AIQueryRequest {
  text: string
  mode?: AIMode
  session_id?: string
}

export const aiApi = {
  // Get available AI modes
  getModes: (): Promise<{ modes: AIModeInfo[] }> => {
    return api.get('/api/ai/modes')
  },

  // Non-streaming query
  query: (request: AIQueryRequest): Promise<AIResponse> => {
    return api.post('/api/ai/query', request)
  },

  // Streaming query - returns an EventSource for SSE
  queryStream: (request: AIQueryRequest): EventSource => {
    // For SSE, we need to use a different approach
    // Create a POST request and handle SSE response
    const eventSource = new EventSource('/api/ai/query/stream?' + new URLSearchParams({
      text: request.text,
      mode: request.mode || 'quick',
      ...(request.session_id && { session_id: request.session_id }),
    }))
    return eventSource
  },

  // Alternative streaming using fetch
  queryStreamFetch: async function* (request: AIQueryRequest): AsyncGenerator<AIStreamEvent> {
    const response = await fetch('/api/ai/query/stream', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'text/event-stream',
      },
      body: JSON.stringify(request),
    })

    if (!response.ok) {
      throw new Error('Stream request failed')
    }

    const reader = response.body?.getReader()
    if (!reader) {
      throw new Error('No response body')
    }

    const decoder = new TextDecoder()
    let buffer = ''

    while (true) {
      const { done, value } = await reader.read()
      if (done) break

      buffer += decoder.decode(value, { stream: true })
      const lines = buffer.split('\n')
      buffer = lines.pop() || ''

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6).trim()
          if (data && data !== '[DONE]') {
            try {
              yield JSON.parse(data) as AIStreamEvent
            } catch {
              // Skip invalid JSON
            }
          }
        }
      }
    }
  },

  // Sessions
  listSessions: (limit = 20, offset = 0): Promise<{ sessions: AISession[]; total: number }> => {
    return api.get(`/api/ai/sessions?limit=${limit}&offset=${offset}`)
  },

  createSession: (title?: string): Promise<{ session: AISession }> => {
    return api.post('/api/ai/sessions', { title })
  },

  getSession: (id: string): Promise<{ session: AISession }> => {
    return api.get(`/api/ai/sessions/${id}`)
  },

  deleteSession: (id: string): Promise<void> => {
    return api.delete(`/api/ai/sessions/${id}`)
  },

  // Canvas
  getCanvas: (sessionId: string): Promise<{ canvas: Canvas }> => {
    return api.get(`/api/ai/canvas/${sessionId}`)
  },

  updateCanvas: (sessionId: string, title: string): Promise<{ canvas: Canvas }> => {
    return api.put(`/api/ai/canvas/${sessionId}`, { title })
  },

  addBlock: (sessionId: string, type: BlockType, content: string, order?: number): Promise<{ block: CanvasBlock }> => {
    return api.post(`/api/ai/canvas/${sessionId}/blocks`, { type, content, order })
  },

  updateBlock: (sessionId: string, blockId: string, updates: Partial<CanvasBlock>): Promise<{ block: CanvasBlock }> => {
    return api.put(`/api/ai/canvas/${sessionId}/blocks/${blockId}`, updates)
  },

  deleteBlock: (sessionId: string, blockId: string): Promise<void> => {
    return api.delete(`/api/ai/canvas/${sessionId}/blocks/${blockId}`)
  },

  reorderBlocks: (sessionId: string, blockIds: string[]): Promise<void> => {
    return api.post(`/api/ai/canvas/${sessionId}/reorder`, { block_ids: blockIds })
  },

  exportCanvas: (sessionId: string, format: ExportFormat = 'markdown'): Promise<Blob> => {
    return fetch(`/api/ai/canvas/${sessionId}/export?format=${format}`)
      .then(res => res.blob())
  },
}
