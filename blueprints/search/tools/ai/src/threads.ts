import { Cache } from './cache'
import { CACHE_TTL, MAX_THREADS, THREAD_ID_LEN } from './config'
import type { Thread, ThreadIndex, ThreadSummary, SearchResult } from './types'

function nanoid(len: number): string {
  const chars = '0123456789abcdefghijklmnopqrstuvwxyz'
  const bytes = crypto.getRandomValues(new Uint8Array(len))
  return Array.from(bytes, b => chars[b % chars.length]).join('')
}

function truncate(s: string, max: number): string {
  return s.length <= max ? s : s.slice(0, max) + '...'
}

export class ThreadManager {
  private cache: Cache

  constructor(kv: KVNamespace) {
    this.cache = new Cache(kv)
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
          backendUUID: result.backendUUID,
          model: result.model,
          durationMs: result.durationMs,
          createdAt: result.createdAt,
        },
      ],
      createdAt: now,
      updatedAt: now,
    }

    await this.cache.set(`thread:${id}`, thread, CACHE_TTL.thread)
    await this.addToIndex(thread)
    return thread
  }

  async getThread(id: string): Promise<Thread | null> {
    return this.cache.get<Thread>(`thread:${id}`)
  }

  async addFollowUp(id: string, query: string, result: SearchResult): Promise<Thread | null> {
    const thread = await this.getThread(id)
    if (!thread) return null

    const now = new Date().toISOString()
    thread.messages.push(
      { role: 'user', content: query, createdAt: now },
      {
        role: 'assistant',
        content: result.answer,
        citations: result.citations,
        webResults: result.webResults,
        relatedQueries: result.relatedQueries,
        backendUUID: result.backendUUID,
        model: result.model,
        durationMs: result.durationMs,
        createdAt: result.createdAt,
      },
    )
    thread.updatedAt = now

    await this.cache.set(`thread:${id}`, thread, CACHE_TTL.thread)
    await this.updateIndex(thread)
    return thread
  }

  async deleteThread(id: string): Promise<boolean> {
    const thread = await this.getThread(id)
    if (!thread) return false
    await this.cache.delete(`thread:${id}`)
    await this.removeFromIndex(id)
    return true
  }

  async listThreads(): Promise<ThreadSummary[]> {
    const index = await this.cache.get<ThreadIndex>('threads:recent')
    return index?.threads || []
  }

  /** Get the last backend_uuid from a thread for follow-up queries. */
  getLastBackendUUID(thread: Thread): string | null {
    for (let i = thread.messages.length - 1; i >= 0; i--) {
      if (thread.messages[i].role === 'assistant' && thread.messages[i].backendUUID) {
        return thread.messages[i].backendUUID!
      }
    }
    return null
  }

  private async addToIndex(thread: Thread): Promise<void> {
    const index = (await this.cache.get<ThreadIndex>('threads:recent')) || { threads: [] }
    index.threads.unshift({
      id: thread.id, title: thread.title, mode: thread.mode, model: thread.model,
      messageCount: thread.messages.length, createdAt: thread.createdAt, updatedAt: thread.updatedAt,
    })
    if (index.threads.length > MAX_THREADS) index.threads = index.threads.slice(0, MAX_THREADS)
    await this.cache.set('threads:recent', index, CACHE_TTL.threadIndex)
  }

  private async updateIndex(thread: Thread): Promise<void> {
    const index = (await this.cache.get<ThreadIndex>('threads:recent')) || { threads: [] }
    index.threads = index.threads.filter(t => t.id !== thread.id)
    index.threads.unshift({
      id: thread.id, title: thread.title, mode: thread.mode, model: thread.model,
      messageCount: thread.messages.length, createdAt: thread.createdAt, updatedAt: thread.updatedAt,
    })
    await this.cache.set('threads:recent', index, CACHE_TTL.threadIndex)
  }

  private async removeFromIndex(id: string): Promise<void> {
    const index = (await this.cache.get<ThreadIndex>('threads:recent')) || { threads: [] }
    index.threads = index.threads.filter(t => t.id !== id)
    await this.cache.set('threads:recent', index, CACHE_TTL.threadIndex)
  }
}
