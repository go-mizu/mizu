import type { ThreadStore } from './storage'
import { THREAD_ID_LEN } from './config'
import type { Thread, ThreadMessage, ThreadSummary, SearchResult } from './types'

function nanoid(len: number): string {
  const chars = '0123456789abcdefghijklmnopqrstuvwxyz'
  const bytes = crypto.getRandomValues(new Uint8Array(len))
  return Array.from(bytes, b => chars[b % chars.length]).join('')
}

function truncate(s: string, max: number): string {
  return s.length <= max ? s : s.slice(0, max) + '...'
}

export class ThreadManager {
  private store: ThreadStore

  constructor(store: ThreadStore) {
    this.store = store
  }

  async createThread(query: string, mode: string, model: string, result: SearchResult): Promise<Thread> {
    const id = nanoid(THREAD_ID_LEN)
    const now = new Date().toISOString()

    const thread: Thread = {
      id,
      title: truncate(query, 100),
      mode,
      model,
      messages: [
        { role: 'user', content: query, createdAt: now },
        {
          role: 'assistant',
          content: result.answer,
          citations: result.citations,
          webResults: result.webResults,
          relatedQueries: result.relatedQueries,
          images: result.images,
          videos: result.videos,
          thinkingSteps: result.thinkingSteps?.length ? result.thinkingSteps : undefined,
          backendUUID: result.backendUUID,
          model: result.model,
          durationMs: result.durationMs,
          createdAt: result.createdAt,
        },
      ],
      createdAt: now,
      updatedAt: now,
    }

    await this.store.createThread(thread)
    return thread
  }

  async getThread(id: string): Promise<Thread | null> {
    return this.store.getThread(id)
  }

  async addFollowUp(id: string, query: string, result: SearchResult): Promise<Thread | null> {
    const now = new Date().toISOString()
    const userMsg: ThreadMessage = { role: 'user', content: query, createdAt: now }
    const assistantMsg: ThreadMessage = {
      role: 'assistant',
      content: result.answer,
      citations: result.citations,
      webResults: result.webResults,
      relatedQueries: result.relatedQueries,
      images: result.images,
      videos: result.videos,
      thinkingSteps: result.thinkingSteps?.length ? result.thinkingSteps : undefined,
      backendUUID: result.backendUUID,
      model: result.model,
      durationMs: result.durationMs,
      createdAt: result.createdAt,
    }
    return this.store.addFollowUp(id, userMsg, assistantMsg)
  }

  async deleteThread(id: string): Promise<boolean> {
    return this.store.deleteThread(id)
  }

  async listThreads(): Promise<ThreadSummary[]> {
    return this.store.listThreads()
  }

  getLastBackendUUID(thread: Thread): string | null {
    for (let i = thread.messages.length - 1; i >= 0; i--) {
      if (thread.messages[i].role === 'assistant' && thread.messages[i].backendUUID) {
        return thread.messages[i].backendUUID!
      }
    }
    return null
  }
}
