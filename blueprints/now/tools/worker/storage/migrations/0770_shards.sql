-- Per-actor table sharding (spec/0763 §20)
-- Each actor gets isolated tables: f_{shard}, e_{shard}, b_{shard}
-- Shard = first 16 hex chars of SHA-256(actor), computable without DB lookup
-- Tables are created lazily on first access; data migrated from shared tables

CREATE TABLE IF NOT EXISTS shards (
  actor      TEXT PRIMARY KEY,
  shard      TEXT NOT NULL UNIQUE,
  next_tx    INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL
);
