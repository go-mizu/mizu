export class Cache {
  private kv: KVNamespace

  constructor(kv: KVNamespace) {
    this.kv = kv
  }

  async get<T>(key: string): Promise<T | null> {
    const val = await this.kv.get(key, 'text')
    if (!val) return null
    return JSON.parse(val) as T
  }

  async set<T>(key: string, value: T, ttlSeconds?: number): Promise<void> {
    try {
      const opts: KVNamespacePutOptions = {}
      if (ttlSeconds && ttlSeconds > 0) opts.expirationTtl = ttlSeconds
      await this.kv.put(key, JSON.stringify(value), opts)
    } catch (e) {
      console.error(`[KV] set "${key}" failed:`, e instanceof Error ? e.message : String(e))
    }
  }

  async delete(key: string): Promise<void> {
    await this.kv.delete(key)
  }
}
