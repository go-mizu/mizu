/**
 * Storage Driver Benchmark — v1 vs v2 (inode)
 *
 * Compares path-based (v1) vs inode-based (v2) engines across D1 and DO drivers.
 * Key metric: directory move performance (O(n) for v1 vs O(1) for v2).
 *
 * Runs in Miniflare. Absolute numbers reflect local emulation.
 * Relative comparisons between v1 and v2 are meaningful.
 */
import { describe, it, expect, beforeAll } from "vitest";
import { env } from "cloudflare:test";
import { D1Engine } from "../storage/d1_driver";
import { D1V2Engine } from "../storage/d1_v2_driver";
import { DOEngine } from "../storage/do_driver";
import { DOV2Engine } from "../storage/do_v2_driver";
import type { StorageEngine } from "../storage/engine";

// ── Setup ────────────────────────────────────────────────────────────

const r2Config = {
  bucket: env.BUCKET,
  r2Endpoint: "https://test.r2.cloudflarestorage.com",
  r2AccessKeyId: "test-key-id",
  r2SecretAccessKey: "test-secret-key",
};

function createD1v1(): D1Engine { return new D1Engine({ db: env.DB, ...r2Config }); }
function createD1v2(): D1V2Engine { return new D1V2Engine({ db: env.DB, ...r2Config }); }

function createDOv1(): StorageEngine | null {
  const ns = (env as any).STORAGE_DO;
  return ns ? new DOEngine({ ns, ...r2Config }) : null;
}

function createDOv2(): StorageEngine | null {
  const ns = (env as any).STORAGE_DO_V2;
  return ns ? new DOV2Engine({ ns, ...r2Config }) : null;
}

// ── Benchmark helpers ────────────────────────────────────────────────

interface BenchResult {
  name: string;
  ops: number;
  avgMs: number;
  p50Ms: number;
  p95Ms: number;
  opsPerSec: number;
}

async function bench(
  name: string,
  fn: () => Promise<void>,
  iterations: number,
  warmup = 3,
): Promise<BenchResult> {
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
    avgMs: latencies.reduce((a, b) => a + b, 0) / latencies.length,
    p50Ms: latencies[Math.floor(latencies.length * 0.5)],
    p95Ms: latencies[Math.floor(latencies.length * 0.95)],
    opsPerSec: (iterations / totalMs) * 1000,
  };
}

function fmtResult(r: BenchResult): string {
  return `${r.name.padEnd(35)} avg=${r.avgMs.toFixed(2).padStart(8)}ms  p50=${r.p50Ms.toFixed(2).padStart(8)}ms  p95=${r.p95Ms.toFixed(2).padStart(8)}ms  ops/s=${r.opsPerSec.toFixed(0).padStart(6)}`;
}

function textContent(s: string): ArrayBuffer {
  return new TextEncoder().encode(s).buffer as ArrayBuffer;
}

// ── Schema setup ─────────────────────────────────────────────────────

beforeAll(async () => {
  const db = env.DB;
  // v1 shared tables (for D1 v1 sharding migration)
  const stmts = [
    `CREATE TABLE IF NOT EXISTS actors (actor TEXT PRIMARY KEY, type TEXT NOT NULL DEFAULT 'human' CHECK(type IN ('human','agent')), public_key TEXT, email TEXT UNIQUE, created_at INTEGER NOT NULL)`,
    `CREATE TABLE IF NOT EXISTS files (owner TEXT NOT NULL, path TEXT NOT NULL, name TEXT NOT NULL, size INTEGER NOT NULL DEFAULT 0, type TEXT NOT NULL DEFAULT 'application/octet-stream', addr TEXT, tx INTEGER, tx_time INTEGER, updated_at INTEGER NOT NULL, PRIMARY KEY (owner, path))`,
    `CREATE INDEX IF NOT EXISTS idx_files_name ON files(owner, name COLLATE NOCASE)`,
    `CREATE TABLE IF NOT EXISTS tx_counter (actor TEXT PRIMARY KEY, next_tx INTEGER NOT NULL DEFAULT 1)`,
    `CREATE TABLE IF NOT EXISTS events (id INTEGER PRIMARY KEY AUTOINCREMENT, tx INTEGER NOT NULL, actor TEXT NOT NULL, action TEXT NOT NULL CHECK(action IN ('write','move','delete')), path TEXT NOT NULL, addr TEXT, size INTEGER NOT NULL DEFAULT 0, type TEXT, meta TEXT, msg TEXT, ts INTEGER NOT NULL)`,
    `CREATE INDEX IF NOT EXISTS idx_events_actor_tx ON events(actor, tx)`,
    `CREATE TABLE IF NOT EXISTS blobs (addr TEXT NOT NULL, actor TEXT NOT NULL, size INTEGER NOT NULL, ref_count INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL, PRIMARY KEY (addr, actor))`,
    `CREATE TABLE IF NOT EXISTS shards (actor TEXT PRIMARY KEY, shard TEXT NOT NULL UNIQUE, next_tx INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL)`,
    // v2 shards table (separate namespace)
    `CREATE TABLE IF NOT EXISTS shards_v2 (actor TEXT PRIMARY KEY, shard TEXT NOT NULL UNIQUE, next_tx INTEGER NOT NULL DEFAULT 1, created_at INTEGER NOT NULL)`,
  ];
  for (const sql of stmts) await db.exec(sql);
});

// ── D1 v1 vs v2 benchmarks ──────────────────────────────────────────

describe("D1 v1 vs v2 benchmark", () => {
  const N = 30;
  const results: BenchResult[] = [];

  it("D1 v1: write", async () => {
    const engine = createD1v1();
    const actor = "bench-d1-v1";
    let i = 0;
    const r = await bench("D1v1: write", async () => {
      await engine.write(actor, `bench/file-${i++}.txt`, textContent(`v1 content ${i}`), "text/plain");
    }, N);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("D1 v2: write", async () => {
    const engine = createD1v2();
    const actor = "bench-d1-v2";
    let i = 0;
    const r = await bench("D1v2: write", async () => {
      await engine.write(actor, `bench/file-${i++}.txt`, textContent(`v2 content ${i}`), "text/plain");
    }, N);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("D1 v1: head", async () => {
    const engine = createD1v1();
    const r = await bench("D1v1: head", () => engine.head("bench-d1-v1", "bench/file-1.txt").then(() => {}), N * 2);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("D1 v2: head", async () => {
    const engine = createD1v2();
    const r = await bench("D1v2: head", () => engine.head("bench-d1-v2", "bench/file-1.txt").then(() => {}), N * 2);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("D1 v1: list", async () => {
    const engine = createD1v1();
    const r = await bench("D1v1: list", () => engine.list("bench-d1-v1", { prefix: "bench/" }).then(() => {}), N);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("D1 v2: list", async () => {
    const engine = createD1v2();
    const r = await bench("D1v2: list", () => engine.list("bench-d1-v2", { prefix: "bench/" }).then(() => {}), N);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("D1 v1: search", async () => {
    const engine = createD1v1();
    const r = await bench("D1v1: search", () => engine.search("bench-d1-v1", "file").then(() => {}), N);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("D1 v2: search", async () => {
    const engine = createD1v2();
    const r = await bench("D1v2: search", () => engine.search("bench-d1-v2", "file").then(() => {}), N);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("D1 v1: stats", async () => {
    const engine = createD1v1();
    const r = await bench("D1v1: stats", () => engine.stats("bench-d1-v1").then(() => {}), N);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("D1 v2: stats", async () => {
    const engine = createD1v2();
    const r = await bench("D1v2: stats", () => engine.stats("bench-d1-v2").then(() => {}), N);
    results.push(r);
    console.log(fmtResult(r));
  });

  // ── Move (single file) ──────────────────────────────────────────

  it("D1 v1: move (single)", async () => {
    const engine = createD1v1();
    const actor = "bench-d1-v1-move";
    for (let i = 0; i < 10; i++) {
      await engine.write(actor, `m/src-${i}.txt`, textContent(`m ${i}`), "text/plain");
    }
    let i = 0;
    const r = await bench("D1v1: move (single)", async () => {
      await engine.move(actor, `m/src-${i}.txt`, `m/dst-${i}.txt`);
      i++;
    }, 10, 0);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("D1 v2: move (single)", async () => {
    const engine = createD1v2();
    const actor = "bench-d1-v2-move";
    for (let i = 0; i < 10; i++) {
      await engine.write(actor, `m/src-${i}.txt`, textContent(`m ${i}`), "text/plain");
    }
    let i = 0;
    const r = await bench("D1v2: move (single)", async () => {
      await engine.move(actor, `m/src-${i}.txt`, `m/dst-${i}.txt`);
      i++;
    }, 10, 0);
    results.push(r);
    console.log(fmtResult(r));
  });

  // ── DIRECTORY MOVE — the killer benchmark ────────────────────────

  it("D1 v1: move directory (50 files)", async () => {
    const engine = createD1v1();
    const actor = "bench-d1-v1-dirmove";
    // Seed 50 files under projects/src/
    for (let i = 0; i < 50; i++) {
      await engine.write(actor, `projects/src/file-${i}.txt`, textContent(`dir move ${i}`), "text/plain");
    }
    // v1 directory move: must move each file individually
    const start = performance.now();
    for (let i = 0; i < 50; i++) {
      await engine.move(actor, `projects/src/file-${i}.txt`, `projects/dst/file-${i}.txt`);
    }
    const elapsed = performance.now() - start;
    const r: BenchResult = {
      name: "D1v1: move dir (50 files)",
      ops: 1,
      avgMs: elapsed,
      p50Ms: elapsed,
      p95Ms: elapsed,
      opsPerSec: 1000 / elapsed,
    };
    results.push(r);
    console.log(fmtResult(r));
  });

  it("D1 v2: move directory (50 files)", async () => {
    const engine = createD1v2();
    const actor = "bench-d1-v2-dirmove";
    // Seed 50 files under projects/src/
    for (let i = 0; i < 50; i++) {
      await engine.write(actor, `projects/src/file-${i}.txt`, textContent(`dir move ${i}`), "text/plain");
    }
    // v2 directory move: O(1) — move the directory node itself
    const start = performance.now();
    await engine.move(actor, "projects/src/", "projects/dst/");
    const elapsed = performance.now() - start;
    const r: BenchResult = {
      name: "D1v2: move dir (50 files)",
      ops: 1,
      avgMs: elapsed,
      p50Ms: elapsed,
      p95Ms: elapsed,
      opsPerSec: 1000 / elapsed,
    };
    results.push(r);
    console.log(fmtResult(r));
  });

  it("D1 v1: delete", async () => {
    const engine = createD1v1();
    let i = 0;
    const r = await bench("D1v1: delete", async () => {
      await engine.delete("bench-d1-v1", [`bench/file-${i++}.txt`]);
    }, Math.min(N, 20));
    results.push(r);
    console.log(fmtResult(r));
  });

  it("D1 v2: delete", async () => {
    const engine = createD1v2();
    let i = 0;
    const r = await bench("D1v2: delete", async () => {
      await engine.delete("bench-d1-v2", [`bench/file-${i++}.txt`]);
    }, Math.min(N, 20));
    results.push(r);
    console.log(fmtResult(r));
  });

  it("print D1 summary", () => {
    console.log("\n" + "=".repeat(90));
    console.log("D1 v1 vs v2 Summary");
    console.log("=".repeat(90));
    for (const r of results) console.log(fmtResult(r));
    console.log("=".repeat(90) + "\n");
    expect(results.length).toBeGreaterThan(0);
  });
});

// ── DO v1 vs v2 benchmarks (only if DO bindings available) ──────────

describe("DO v1 vs v2 benchmark", () => {
  const N = 30;
  const results: BenchResult[] = [];
  let doV1: StorageEngine | null;
  let doV2: StorageEngine | null;

  beforeAll(() => {
    doV1 = createDOv1();
    doV2 = createDOv2();
    if (!doV1) console.log("\n⚠ STORAGE_DO binding not available — skipping DO v1 benchmarks");
    if (!doV2) console.log("\n⚠ STORAGE_DO_V2 binding not available — skipping DO v2 benchmarks");
  });

  it("DO v1: write", async () => {
    if (!doV1) return;
    const actor = "bench-do-v1";
    let i = 0;
    const r = await bench("DOv1: write", async () => {
      await doV1!.write(actor, `bench/file-${i++}.txt`, textContent(`v1 content ${i}`), "text/plain");
    }, N);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("DO v2: write", async () => {
    if (!doV2) return;
    const actor = "bench-do-v2";
    let i = 0;
    const r = await bench("DOv2: write", async () => {
      await doV2!.write(actor, `bench/file-${i++}.txt`, textContent(`v2 content ${i}`), "text/plain");
    }, N);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("DO v1: head", async () => {
    if (!doV1) return;
    const r = await bench("DOv1: head", () => doV1!.head("bench-do-v1", "bench/file-1.txt").then(() => {}), N * 2);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("DO v2: head", async () => {
    if (!doV2) return;
    const r = await bench("DOv2: head", () => doV2!.head("bench-do-v2", "bench/file-1.txt").then(() => {}), N * 2);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("DO v1: list", async () => {
    if (!doV1) return;
    const r = await bench("DOv1: list", () => doV1!.list("bench-do-v1", { prefix: "bench/" }).then(() => {}), N);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("DO v2: list", async () => {
    if (!doV2) return;
    const r = await bench("DOv2: list", () => doV2!.list("bench-do-v2", { prefix: "bench/" }).then(() => {}), N);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("DO v1: search", async () => {
    if (!doV1) return;
    const r = await bench("DOv1: search", () => doV1!.search("bench-do-v1", "file").then(() => {}), N);
    results.push(r);
    console.log(fmtResult(r));
  });

  it("DO v2: search", async () => {
    if (!doV2) return;
    const r = await bench("DOv2: search", () => doV2!.search("bench-do-v2", "file").then(() => {}), N);
    results.push(r);
    console.log(fmtResult(r));
  });

  // ── DIRECTORY MOVE — the killer benchmark ────────────────────────

  it("DO v1: move directory (50 files)", async () => {
    if (!doV1) return;
    const actor = "bench-do-v1-dirmove";
    for (let i = 0; i < 50; i++) {
      await doV1!.write(actor, `projects/src/file-${i}.txt`, textContent(`dir move ${i}`), "text/plain");
    }
    const start = performance.now();
    for (let i = 0; i < 50; i++) {
      await doV1!.move(actor, `projects/src/file-${i}.txt`, `projects/dst/file-${i}.txt`);
    }
    const elapsed = performance.now() - start;
    const r: BenchResult = {
      name: "DOv1: move dir (50 files)",
      ops: 1, avgMs: elapsed, p50Ms: elapsed, p95Ms: elapsed, opsPerSec: 1000 / elapsed,
    };
    results.push(r);
    console.log(fmtResult(r));
  });

  it("DO v2: move directory (50 files)", async () => {
    if (!doV2) return;
    const actor = "bench-do-v2-dirmove";
    for (let i = 0; i < 50; i++) {
      await doV2!.write(actor, `projects/src/file-${i}.txt`, textContent(`dir move ${i}`), "text/plain");
    }
    const start = performance.now();
    await doV2!.move(actor, "projects/src/", "projects/dst/");
    const elapsed = performance.now() - start;
    const r: BenchResult = {
      name: "DOv2: move dir (50 files)",
      ops: 1, avgMs: elapsed, p50Ms: elapsed, p95Ms: elapsed, opsPerSec: 1000 / elapsed,
    };
    results.push(r);
    console.log(fmtResult(r));
  });

  it("print DO summary", () => {
    console.log("\n" + "=".repeat(90));
    console.log("DO v1 vs v2 Summary");
    console.log("=".repeat(90));
    for (const r of results) console.log(fmtResult(r));
    console.log("=".repeat(90) + "\n");
    expect(true).toBe(true);
  });
});
