/**
 * Storage Driver Benchmark
 *
 * Compares D1Engine (shared D1 + per-actor sharding) vs DOEngine
 * (per-actor Durable Object with local SQLite).
 *
 * Metrics: latency (p50, p95, p99), throughput (ops/sec), cost per op.
 *
 * NOTE: This runs in Miniflare (local emulation), so absolute numbers
 * don't reflect production. Relative comparisons between drivers are
 * meaningful since both run in the same environment.
 */
import { describe, it, expect, beforeAll } from "vitest";
import { env } from "cloudflare:test";
import { D1Engine } from "../storage/d1_driver";
import { DOEngine } from "../storage/do_driver";
import type { StorageEngine } from "../storage/engine";

// ── Setup ────────────────────────────────────────────────────────────

function createD1Engine(): D1Engine {
  return new D1Engine({
    db: env.DB,
    bucket: env.BUCKET,
    r2Endpoint: "https://test.r2.cloudflarestorage.com",
    r2AccessKeyId: "test-key-id",
    r2SecretAccessKey: "test-secret-key",
  });
}

// DOEngine requires a DurableObjectNamespace which is only available
// when wrangler is configured with the DO binding. We test it if available.
function createDOEngine(): StorageEngine | null {
  const ns = (env as any).STORAGE_DO;
  if (!ns) return null;
  return new DOEngine({
    ns,
    bucket: env.BUCKET,
    r2Endpoint: "https://test.r2.cloudflarestorage.com",
    r2AccessKeyId: "test-key-id",
    r2SecretAccessKey: "test-secret-key",
  });
}

// ── Benchmark helpers ────────────────────────────────────────────────

interface BenchResult {
  name: string;
  ops: number;
  totalMs: number;
  avgMs: number;
  p50Ms: number;
  p95Ms: number;
  p99Ms: number;
  opsPerSec: number;
}

async function bench(
  name: string,
  fn: () => Promise<void>,
  iterations: number,
  warmup = 3,
): Promise<BenchResult> {
  // Warmup
  for (let i = 0; i < warmup; i++) await fn();

  const latencies: number[] = [];
  const t0 = performance.now();

  for (let i = 0; i < iterations; i++) {
    const start = performance.now();
    await fn();
    latencies.push(performance.now() - start);
  }

  const totalMs = performance.now() - t0;
  latencies.sort((a, b) => a - b);

  return {
    name,
    ops: iterations,
    totalMs,
    avgMs: latencies.reduce((a, b) => a + b, 0) / latencies.length,
    p50Ms: latencies[Math.floor(latencies.length * 0.5)],
    p95Ms: latencies[Math.floor(latencies.length * 0.95)],
    p99Ms: latencies[Math.floor(latencies.length * 0.99)],
    opsPerSec: (iterations / totalMs) * 1000,
  };
}

function formatResults(results: BenchResult[]): string {
  const header = "| Operation | Driver | Ops | Avg (ms) | p50 (ms) | p95 (ms) | p99 (ms) | ops/sec |";
  const sep =    "|-----------|--------|-----|----------|----------|----------|----------|---------|";
  const rows = results.map((r) =>
    `| ${r.name.padEnd(30)} | ${r.ops.toString().padStart(3)} | ${r.avgMs.toFixed(2).padStart(8)} | ${r.p50Ms.toFixed(2).padStart(8)} | ${r.p95Ms.toFixed(2).padStart(8)} | ${r.p99Ms.toFixed(2).padStart(8)} | ${r.opsPerSec.toFixed(0).padStart(7)} |`,
  );
  return [header, sep, ...rows].join("\n");
}

// ── Schema setup ─────────────────────────────────────────────────────

beforeAll(async () => {
  const db = env.DB;
  const stmts = [
    `CREATE TABLE IF NOT EXISTS actors (actor TEXT PRIMARY KEY, type TEXT NOT NULL DEFAULT 'human' CHECK(type IN ('human','agent')), public_key TEXT, email TEXT UNIQUE, created_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS files (owner TEXT NOT NULL, path TEXT NOT NULL, name TEXT NOT NULL, size INTEGER NOT NULL DEFAULT 0, type TEXT NOT NULL DEFAULT 'application/octet-stream', addr TEXT, tx INTEGER, tx_time INTEGER, updated_at INTEGER NOT NULL, PRIMARY KEY (owner, path))`,
    `CREATE INDEX IF NOT EXISTS idx_files_name ON files(owner, name COLLATE NOCASE)`,
    `CREATE TABLE IF NOT EXISTS tx_counter (actor TEXT PRIMARY KEY, next_tx INTEGER NOT NULL DEFAULT 1)`,
    `CREATE TABLE IF NOT EXISTS events (id INTEGER PRIMARY KEY AUTOINCREMENT, tx INTEGER NOT NULL, actor TEXT NOT NULL, action TEXT NOT NULL CHECK(action IN ('write','move','delete')), path TEXT NOT NULL, addr TEXT, size INTEGER NOT NULL DEFAULT 0, type TEXT, meta TEXT, msg TEXT, ts INTEGER NOT NULL)`,
    `CREATE INDEX IF NOT EXISTS idx_events_actor_tx ON events(actor, tx)`,
    `CREATE TABLE IF NOT EXISTS blobs (addr TEXT NOT NULL, actor TEXT NOT NULL, size INTEGER NOT NULL, ref_count INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL, PRIMARY KEY (addr, actor))`,
    `CREATE TABLE IF NOT EXISTS shards (actor TEXT PRIMARY KEY, shard TEXT NOT NULL UNIQUE, next_tx INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL)`,
  ];
  for (const sql of stmts) await db.exec(sql);

  await db.prepare(
    "INSERT OR IGNORE INTO actors (actor, type, public_key, created_at) VALUES (?, 'agent', 'testkey', ?)",
  ).bind("bench-d1", Date.now()).run();
  await db.prepare(
    "INSERT OR IGNORE INTO actors (actor, type, public_key, created_at) VALUES (?, 'agent', 'testkey', ?)",
  ).bind("bench-do", Date.now()).run();
});

// ── D1 Engine benchmarks ─────────────────────────────────────────────

describe("D1Engine benchmark", () => {
  const N_WRITE = 50;
  const N_READ = 100;
  const N_LIST = 50;
  const N_SEARCH = 50;
  const actor = "bench-d1";

  it("write latency", async () => {
    const engine = createD1Engine();
    let i = 0;
    const result = await bench("D1: write", async () => {
      const data = new TextEncoder().encode(`file content ${i++} ${Date.now()}`);
      await engine.write(actor, `bench/file-${i}.txt`, data.buffer as ArrayBuffer, "text/plain");
    }, N_WRITE);

    console.log(`\nD1 write: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBe(N_WRITE);
  });

  it("head latency", async () => {
    const engine = createD1Engine();
    const result = await bench("D1: head", async () => {
      await engine.head(actor, "bench/file-1.txt");
    }, N_READ);

    console.log(`D1 head: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBe(N_READ);
  });

  it("list latency", async () => {
    const engine = createD1Engine();
    const result = await bench("D1: list", async () => {
      await engine.list(actor, { prefix: "bench/" });
    }, N_LIST);

    console.log(`D1 list: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBe(N_LIST);
  });

  it("search latency", async () => {
    const engine = createD1Engine();
    const result = await bench("D1: search", async () => {
      await engine.search(actor, "file");
    }, N_SEARCH);

    console.log(`D1 search: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBe(N_SEARCH);
  });

  it("stats latency", async () => {
    const engine = createD1Engine();
    const result = await bench("D1: stats", async () => {
      await engine.stats(actor);
    }, N_READ);

    console.log(`D1 stats: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBe(N_READ);
  });

  it("delete latency", async () => {
    const engine = createD1Engine();
    let i = 0;
    const result = await bench("D1: delete", async () => {
      await engine.delete(actor, [`bench/file-${++i}.txt`]);
    }, Math.min(N_WRITE, 30));

    console.log(`D1 delete: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBeGreaterThan(0);
  });

  it("move latency", async () => {
    const engine = createD1Engine();
    // Seed some files to move
    for (let i = 0; i < 10; i++) {
      const data = new TextEncoder().encode(`move me ${i}`);
      await engine.write(actor, `bench/move-src-${i}.txt`, data.buffer as ArrayBuffer, "text/plain");
    }
    let i = 0;
    const result = await bench("D1: move", async () => {
      await engine.move(actor, `bench/move-src-${i}.txt`, `bench/move-dst-${i}.txt`);
      i++;
    }, 10, 0);

    console.log(`D1 move: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBe(10);
  });
});

// ── DO Engine benchmarks (only if DO binding is available) ───────────

describe("DOEngine benchmark", () => {
  const N_WRITE = 50;
  const N_READ = 100;
  const N_LIST = 50;
  const N_SEARCH = 50;
  const actor = "bench-do";

  let doEngine: StorageEngine | null;

  beforeAll(() => {
    doEngine = createDOEngine();
    if (!doEngine) {
      console.log("\n⚠ STORAGE_DO binding not available — skipping DO benchmarks");
    }
  });

  it("write latency", async () => {
    if (!doEngine) return; // skip
    let i = 0;
    const result = await bench("DO: write", async () => {
      const data = new TextEncoder().encode(`file content ${i++} ${Date.now()}`);
      await doEngine!.write(actor, `bench/file-${i}.txt`, data.buffer as ArrayBuffer, "text/plain");
    }, N_WRITE);

    console.log(`\nDO write: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBe(N_WRITE);
  });

  it("head latency", async () => {
    if (!doEngine) return;
    const result = await bench("DO: head", async () => {
      await doEngine!.head(actor, "bench/file-1.txt");
    }, N_READ);

    console.log(`DO head: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBe(N_READ);
  });

  it("list latency", async () => {
    if (!doEngine) return;
    const result = await bench("DO: list", async () => {
      await doEngine!.list(actor, { prefix: "bench/" });
    }, N_LIST);

    console.log(`DO list: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBe(N_LIST);
  });

  it("search latency", async () => {
    if (!doEngine) return;
    const result = await bench("DO: search", async () => {
      await doEngine!.search(actor, "file");
    }, N_SEARCH);

    console.log(`DO search: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBe(N_SEARCH);
  });

  it("stats latency", async () => {
    if (!doEngine) return;
    const result = await bench("DO: stats", async () => {
      await doEngine!.stats(actor);
    }, N_READ);

    console.log(`DO stats: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBe(N_READ);
  });

  it("delete latency", async () => {
    if (!doEngine) return;
    let i = 0;
    const result = await bench("DO: delete", async () => {
      await doEngine!.delete(actor, [`bench/file-${++i}.txt`]);
    }, Math.min(N_WRITE, 30));

    console.log(`DO delete: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBeGreaterThan(0);
  });

  it("move latency", async () => {
    if (!doEngine) return;
    for (let i = 0; i < 10; i++) {
      const data = new TextEncoder().encode(`move me ${i}`);
      await doEngine!.write(actor, `bench/move-src-${i}.txt`, data.buffer as ArrayBuffer, "text/plain");
    }
    let i = 0;
    const result = await bench("DO: move", async () => {
      await doEngine!.move(actor, `bench/move-src-${i}.txt`, `bench/move-dst-${i}.txt`);
      i++;
    }, 10, 0);

    console.log(`DO move: avg=${result.avgMs.toFixed(2)}ms p50=${result.p50Ms.toFixed(2)}ms p95=${result.p95Ms.toFixed(2)}ms ops/sec=${result.opsPerSec.toFixed(0)}`);
    expect(result.ops).toBe(10);
  });
});
