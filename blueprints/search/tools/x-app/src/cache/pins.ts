import * as SQLite from 'expo-sqlite'
import type { Profile } from '../api/types'

let db: SQLite.SQLiteDatabase | null = null

async function getDB(): Promise<SQLite.SQLiteDatabase> {
  if (db) return db
  db = await SQLite.openDatabaseAsync('xcache.db')
  await db.execAsync(`
    CREATE TABLE IF NOT EXISTS pinned_profiles (
      username TEXT PRIMARY KEY,
      profile_json TEXT NOT NULL,
      pinned_at INTEGER NOT NULL
    );
  `)
  return db
}

export async function pinProfile(profile: Profile): Promise<void> {
  const d = await getDB()
  const now = Math.floor(Date.now() / 1000)
  await d.runAsync(
    'INSERT OR REPLACE INTO pinned_profiles (username, profile_json, pinned_at) VALUES (?, ?, ?)',
    [profile.username.toLowerCase(), JSON.stringify(profile), now]
  )
}

export async function unpinProfile(username: string): Promise<void> {
  const d = await getDB()
  await d.runAsync('DELETE FROM pinned_profiles WHERE username = ?', [username.toLowerCase()])
}

export async function isPinned(username: string): Promise<boolean> {
  const d = await getDB()
  const row = await d.getFirstAsync<{ username: string }>(
    'SELECT username FROM pinned_profiles WHERE username = ?',
    [username.toLowerCase()]
  )
  return !!row
}

export async function getPinnedProfiles(): Promise<Profile[]> {
  const d = await getDB()
  const rows = await d.getAllAsync<{ profile_json: string }>(
    'SELECT profile_json FROM pinned_profiles ORDER BY pinned_at DESC'
  )
  return rows.map(r => JSON.parse(r.profile_json) as Profile)
}

export async function getPinnedCount(): Promise<number> {
  const d = await getDB()
  const row = await d.getFirstAsync<{ count: number }>(
    'SELECT COUNT(*) as count FROM pinned_profiles'
  )
  return row?.count ?? 0
}

export async function clearPinnedProfiles(): Promise<void> {
  const d = await getDB()
  await d.execAsync('DELETE FROM pinned_profiles')
}
