// Bump version to invalidate stale cache entries after schema/query changes
const V = 'v2:'

export class Cache {
  private kv: KVNamespace | undefined

  constructor(kv?: KVNamespace) {
    this.kv = kv
  }

  async get<T>(key: string): Promise<T | null> {
    if (!this.kv) return null
    const val = await this.kv.get(V + key, 'text')
    if (!val) return null
    return JSON.parse(val) as T
  }

  async set<T>(key: string, value: T): Promise<void> {
    if (!this.kv) return
    try {
      await this.kv.put(V + key, JSON.stringify(value))
    } catch { /* best-effort — KV write quota may be exhausted */ }
  }
}
