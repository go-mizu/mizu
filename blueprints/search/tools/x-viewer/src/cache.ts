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

  async set<T>(key: string, value: T, ttlSeconds: number): Promise<void> {
    await this.kv.put(key, JSON.stringify(value), { expirationTtl: ttlSeconds })
  }
}
