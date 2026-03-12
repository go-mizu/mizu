import type { D1Database } from "@cloudflare/workers-types";
import type { PageCacheRow } from "./types";

// ── In-memory L1 cache ────────────────────────────────────────────────────────
// Module-level Map; lives for the isolate lifetime (no TTL).
// Key: makeCacheKey(url, endpoint, paramsHash)

interface MemEntry {
  html: string | null;
  markdown: string | null;
  result: string | null;
  title: string | null;
}

const MEM: Map<string, MemEntry> = new Map();

export function makeCacheKey(url: string, endpoint: string, paramsHash: string): string {
  return `${url}\0${endpoint}\0${paramsHash}`;
}

// ── params hashing ────────────────────────────────────────────────────────────

/**
 * Deterministic 16-char hex hash of an arbitrary object.
 * Keys are sorted before serialisation so {a,b} and {b,a} hash identically.
 * Returns "" for null/undefined (signals "no params").
 */
export async function hashParams(obj: unknown): Promise<string> {
  if (obj === null || obj === undefined) return "";
  const sorted = sortedStringify(obj);
  const buf = await crypto.subtle.digest(
    "SHA-256",
    new TextEncoder().encode(sorted)
  );
  return Array.from(new Uint8Array(buf))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("")
    .slice(0, 16);
}

function sortedStringify(val: unknown): string {
  if (typeof val !== "object" || val === null) return JSON.stringify(val);
  if (Array.isArray(val)) return "[" + val.map(sortedStringify).join(",") + "]";
  const keys = Object.keys(val as object).sort();
  const pairs = keys.map((k) => JSON.stringify(k) + ":" + sortedStringify((val as Record<string, unknown>)[k]));
  return "{" + pairs.join(",") + "}";
}

// ── L1: in-memory ─────────────────────────────────────────────────────────────

export function memGet(key: string): MemEntry | undefined {
  return MEM.get(key);
}

export function memSet(key: string, entry: MemEntry): void {
  MEM.set(key, entry);
}

// ── L2: D1 ───────────────────────────────────────────────────────────────────

export async function d1Get(
  db: D1Database,
  url: string,
  endpoint: string,
  paramsHash: string
): Promise<PageCacheRow | null> {
  return db
    .prepare(
      "SELECT * FROM page_cache WHERE url = ? AND endpoint = ? AND params_hash = ? LIMIT 1"
    )
    .bind(url, endpoint, paramsHash)
    .first<PageCacheRow>();
}

export async function d1Set(
  db: D1Database,
  row: Omit<PageCacheRow, "created_at">
): Promise<void> {
  await db
    .prepare(
      `INSERT OR REPLACE INTO page_cache
         (url, endpoint, params_hash, html, markdown, result, title, created_at)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
    )
    .bind(
      row.url,
      row.endpoint,
      row.params_hash,
      row.html ?? null,
      row.markdown ?? null,
      row.result ?? null,
      row.title ?? null,
      Date.now()
    )
    .run();
}

// ── Combined read (L1 → L2) ───────────────────────────────────────────────────

export async function cacheGet(
  db: D1Database,
  url: string,
  endpoint: string,
  paramsHash: string
): Promise<MemEntry | null> {
  const key = makeCacheKey(url, endpoint, paramsHash);

  // L1 hit
  const mem = memGet(key);
  if (mem) return mem;

  // L2 hit
  const row = await d1Get(db, url, endpoint, paramsHash);
  if (row) {
    const entry: MemEntry = {
      html: row.html,
      markdown: row.markdown,
      result: row.result,
      title: row.title,
    };
    memSet(key, entry);
    return entry;
  }

  return null;
}

// ── Combined write (L1 + L2) ──────────────────────────────────────────────────

export async function cacheSet(
  db: D1Database,
  url: string,
  endpoint: string,
  paramsHash: string,
  entry: MemEntry & { title?: string | null }
): Promise<void> {
  const key = makeCacheKey(url, endpoint, paramsHash);
  memSet(key, entry);
  await d1Set(db, {
    url,
    endpoint,
    params_hash: paramsHash,
    html: entry.html ?? null,
    markdown: entry.markdown ?? null,
    result: entry.result ?? null,
    title: entry.title ?? null,
  });
}
