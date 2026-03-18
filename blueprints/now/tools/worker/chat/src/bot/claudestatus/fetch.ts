import type { SummaryResponse, IncidentsResponse } from "./format";
import { SITE_NAME } from "../../constants";

const MEM_TTL_MS = 5 * 60 * 1000;   // 5 minutes
const D1_TTL_MS  = 15 * 60 * 1000;  // 15 minutes

interface CacheEntry { data: unknown; expiresAt: number; }
const memCache = new Map<string, CacheEntry>();

const BASE = "https://status.claude.com/api/v2";

async function fetchWithCache<T>(
  key: string,
  url: string,
  db: D1Database
): Promise<{ data: T; stale: boolean } | null> {
  const now = Date.now();

  // 1. Memory cache
  const mem = memCache.get(key);
  if (mem && mem.expiresAt > now) {
    return { data: mem.data as T, stale: false };
  }

  // 2. D1 cache
  const row = await db
    .prepare("SELECT value, expires_at FROM bot_cache WHERE key = ?")
    .bind(key)
    .first<{ value: string; expires_at: number }>();

  if (row && row.expires_at > now) {
    const parsed = JSON.parse(row.value) as T;
    memCache.set(key, { data: parsed, expiresAt: row.expires_at });
    return { data: parsed, stale: false };
  }

  // 3. Live fetch
  try {
    const res = await fetch(url, { headers: { "User-Agent": `${SITE_NAME}/1.0` } });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const data = (await res.json()) as T;

    const expiresAt = now + D1_TTL_MS;
    // Write D1 (upsert)
    try {
      await db
        .prepare("INSERT OR REPLACE INTO bot_cache (key, value, expires_at) VALUES (?, ?, ?)")
        .bind(key, JSON.stringify(data), expiresAt)
        .run();
    } catch (e) {
      console.error("[claudestatus] D1 write failed:", e);
    }
    // Write memory
    memCache.set(key, { data, expiresAt: now + MEM_TTL_MS });

    return { data, stale: false };
  } catch {
    // 4. Stale fallback — return any existing D1 row (even expired)
    if (row) {
      const parsed = JSON.parse(row.value) as T;
      return { data: parsed, stale: true };
    }
    return null;
  }
}

export async function fetchSummary(
  db: D1Database
): Promise<{ data: SummaryResponse; stale: boolean } | null> {
  return fetchWithCache<SummaryResponse>(
    "claudestatus:summary",
    `${BASE}/summary.json`,
    db
  );
}

export async function fetchIncidents(
  db: D1Database
): Promise<{ data: IncidentsResponse; stale: boolean } | null> {
  return fetchWithCache<IncidentsResponse>(
    "claudestatus:incidents",
    `${BASE}/incidents.json`,
    db
  );
}
