import * as SQLite from 'expo-sqlite'
import type { Tweet } from '../api/types'

let db: SQLite.SQLiteDatabase | null = null

async function getDB(): Promise<SQLite.SQLiteDatabase> {
  if (db) return db
  db = await SQLite.openDatabaseAsync('xcache.db')
  await db.execAsync(`
    CREATE TABLE IF NOT EXISTS bookmarks (
      tweet_id TEXT PRIMARY KEY,
      tweet_json TEXT NOT NULL,
      bookmarked_at INTEGER NOT NULL
    );
  `)
  return db
}

export async function addBookmark(tweet: Tweet): Promise<void> {
  const d = await getDB()
  const now = Math.floor(Date.now() / 1000)
  await d.runAsync(
    'INSERT OR REPLACE INTO bookmarks (tweet_id, tweet_json, bookmarked_at) VALUES (?, ?, ?)',
    [tweet.id, JSON.stringify(tweet), now]
  )
}

export async function removeBookmark(tweetId: string): Promise<void> {
  const d = await getDB()
  await d.runAsync('DELETE FROM bookmarks WHERE tweet_id = ?', [tweetId])
}

export async function isBookmarked(tweetId: string): Promise<boolean> {
  const d = await getDB()
  const row = await d.getFirstAsync<{ tweet_id: string }>(
    'SELECT tweet_id FROM bookmarks WHERE tweet_id = ?',
    [tweetId]
  )
  return !!row
}

export async function getBookmarks(): Promise<Tweet[]> {
  const d = await getDB()
  const rows = await d.getAllAsync<{ tweet_json: string }>(
    'SELECT tweet_json FROM bookmarks ORDER BY bookmarked_at DESC'
  )
  return rows.map(r => JSON.parse(r.tweet_json) as Tweet)
}

export async function getBookmarkCount(): Promise<number> {
  const d = await getDB()
  const row = await d.getFirstAsync<{ count: number }>(
    'SELECT COUNT(*) as count FROM bookmarks'
  )
  return row?.count ?? 0
}

export async function clearBookmarks(): Promise<void> {
  const d = await getDB()
  await d.execAsync('DELETE FROM bookmarks')
}
