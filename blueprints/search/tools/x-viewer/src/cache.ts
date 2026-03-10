const now = () => Math.floor(Date.now() / 1000)

export interface CachedRow<T> {
  data: T
  fetchedAt: number
}

export class DB {
  private d1: D1Database

  constructor(d1: D1Database) {
    this.d1 = d1
  }

  // --- Profile ---

  async getProfile<T>(username: string): Promise<T | null> {
    const row = await this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM profiles WHERE username = ? AND expires_at > ?',
      [username.toLowerCase(), now()]
    )
    return row ? row.data : null
  }

  async getProfileWithMeta<T>(username: string): Promise<CachedRow<T> | null> {
    return this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM profiles WHERE username = ? AND expires_at > ?',
      [username.toLowerCase(), now()]
    )
  }

  async setProfile<T>(username: string, data: T, ttl: number): Promise<void> {
    await this.run(
      'INSERT OR REPLACE INTO profiles (username, data, fetched_at, expires_at) VALUES (?, ?, ?, ?)',
      [username.toLowerCase(), JSON.stringify(data), now(), now() + ttl]
    )
  }

  // --- Tweet ---

  async getTweet<T>(tweetID: string, cursor = ''): Promise<T | null> {
    const row = await this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM tweets WHERE tweet_id = ? AND cursor = ? AND expires_at > ?',
      [tweetID, cursor, now()]
    )
    return row ? row.data : null
  }

  async getTweetWithMeta<T>(tweetID: string, cursor = ''): Promise<CachedRow<T> | null> {
    return this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM tweets WHERE tweet_id = ? AND cursor = ? AND expires_at > ?',
      [tweetID, cursor, now()]
    )
  }

  async setTweet<T>(tweetID: string, cursor: string, data: T, ttl: number): Promise<void> {
    await this.run(
      'INSERT OR REPLACE INTO tweets (tweet_id, cursor, data, fetched_at, expires_at) VALUES (?, ?, ?, ?, ?)',
      [tweetID, cursor, JSON.stringify(data), now(), now() + ttl]
    )
  }

  // --- Article ---

  async getArticle(tweetID: string): Promise<{ tweetData: string; body: string } | null> {
    try {
      const row = await this.d1
        .prepare('SELECT tweet_data, body FROM articles WHERE tweet_id = ? AND expires_at > ? AND body != \'\'')
        .bind(tweetID, now())
        .first<{ tweet_data: string; body: string }>()
      if (!row) return null
      return { tweetData: row.tweet_data, body: row.body }
    } catch {
      return null
    }
  }

  async setArticle(tweetID: string, tweetData: string, body: string, ttl: number): Promise<void> {
    await this.run(
      'INSERT OR REPLACE INTO articles (tweet_id, tweet_data, body, fetched_at, expires_at) VALUES (?, ?, ?, ?, ?)',
      [tweetID, tweetData, body, now(), now() + ttl]
    )
  }

  // --- Timeline ---

  async getTimeline<T>(username: string, tab: string, cursor = ''): Promise<T | null> {
    const row = await this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM timelines WHERE username = ? AND tab = ? AND cursor = ? AND expires_at > ?',
      [username.toLowerCase(), tab, cursor, now()]
    )
    return row ? row.data : null
  }

  async getTimelineWithMeta<T>(username: string, tab: string, cursor = ''): Promise<CachedRow<T> | null> {
    return this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM timelines WHERE username = ? AND tab = ? AND cursor = ? AND expires_at > ?',
      [username.toLowerCase(), tab, cursor, now()]
    )
  }

  async setTimeline<T>(username: string, tab: string, cursor: string, data: T, ttl: number): Promise<void> {
    await this.run(
      'INSERT OR REPLACE INTO timelines (username, tab, cursor, data, fetched_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)',
      [username.toLowerCase(), tab, cursor, JSON.stringify(data), now(), now() + ttl]
    )
  }

  // --- Search ---

  async getSearch<T>(query: string, mode: string, cursor = ''): Promise<T | null> {
    const row = await this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM searches WHERE query = ? AND mode = ? AND cursor = ? AND expires_at > ?',
      [query, mode, cursor, now()]
    )
    return row ? row.data : null
  }

  async getSearchWithMeta<T>(query: string, mode: string, cursor = ''): Promise<CachedRow<T> | null> {
    return this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM searches WHERE query = ? AND mode = ? AND cursor = ? AND expires_at > ?',
      [query, mode, cursor, now()]
    )
  }

  async setSearch<T>(query: string, mode: string, cursor: string, data: T, ttl: number): Promise<void> {
    await this.run(
      'INSERT OR REPLACE INTO searches (query, mode, cursor, data, fetched_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)',
      [query, mode, cursor, JSON.stringify(data), now(), now() + ttl]
    )
  }

  // --- List ---

  async getList<T>(listID: string): Promise<T | null> {
    const row = await this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM lists WHERE list_id = ? AND expires_at > ?',
      [listID, now()]
    )
    return row ? row.data : null
  }

  async getListWithMeta<T>(listID: string): Promise<CachedRow<T> | null> {
    return this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM lists WHERE list_id = ? AND expires_at > ?',
      [listID, now()]
    )
  }

  async setList<T>(listID: string, data: T, ttl: number): Promise<void> {
    await this.run(
      'INSERT OR REPLACE INTO lists (list_id, data, fetched_at, expires_at) VALUES (?, ?, ?, ?)',
      [listID, JSON.stringify(data), now(), now() + ttl]
    )
  }

  async getListContent<T>(listID: string, contentType: string, cursor = ''): Promise<T | null> {
    const row = await this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM list_content WHERE list_id = ? AND content_type = ? AND cursor = ? AND expires_at > ?',
      [listID, contentType, cursor, now()]
    )
    return row ? row.data : null
  }

  async getListContentWithMeta<T>(listID: string, contentType: string, cursor = ''): Promise<CachedRow<T> | null> {
    return this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM list_content WHERE list_id = ? AND content_type = ? AND cursor = ? AND expires_at > ?',
      [listID, contentType, cursor, now()]
    )
  }

  async setListContent<T>(listID: string, contentType: string, cursor: string, data: T, ttl: number): Promise<void> {
    await this.run(
      'INSERT OR REPLACE INTO list_content (list_id, content_type, cursor, data, fetched_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)',
      [listID, contentType, cursor, JSON.stringify(data), now(), now() + ttl]
    )
  }

  // --- Follow ---

  async getFollow<T>(username: string, followType: string, cursor = ''): Promise<T | null> {
    const row = await this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM follows WHERE username = ? AND follow_type = ? AND cursor = ? AND expires_at > ?',
      [username.toLowerCase(), followType, cursor, now()]
    )
    return row ? row.data : null
  }

  async getFollowWithMeta<T>(username: string, followType: string, cursor = ''): Promise<CachedRow<T> | null> {
    return this.getRowWithMeta<T>(
      'SELECT data, fetched_at FROM follows WHERE username = ? AND follow_type = ? AND cursor = ? AND expires_at > ?',
      [username.toLowerCase(), followType, cursor, now()]
    )
  }

  async setFollow<T>(username: string, followType: string, cursor: string, data: T, ttl: number): Promise<void> {
    await this.run(
      'INSERT OR REPLACE INTO follows (username, follow_type, cursor, data, fetched_at, expires_at) VALUES (?, ?, ?, ?, ?, ?)',
      [username.toLowerCase(), followType, cursor, JSON.stringify(data), now(), now() + ttl]
    )
  }

  // --- Internal helpers ---

  private async getRowWithMeta<T>(sql: string, params: unknown[]): Promise<CachedRow<T> | null> {
    try {
      const row = await this.d1
        .prepare(sql)
        .bind(...params)
        .first<{ data: string; fetched_at: number }>()
      if (!row) return null
      return { data: JSON.parse(row.data) as T, fetchedAt: row.fetched_at }
    } catch {
      return null
    }
  }

  private async run(sql: string, params: unknown[]): Promise<void> {
    try {
      await this.d1.prepare(sql).bind(...params).run()
    } catch { /* write failed — continue */ }
  }
}
