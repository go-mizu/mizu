/**
 * PostgreSQL Driver Benchmark
 *
 * Benchmarks NeonEngine (HTTP + WebSocket) and HyperdriveEngine
 * (postgres library, simulating Hyperdrive with direct connection)
 * against a real Neon PostgreSQL database.
 *
 * Both drivers share PgEngineBase, so the SQL is identical.
 * The difference is the transport layer:
 *   - Neon: HTTP for reads, WebSocket for transactions
 *   - Hyperdrive: TCP via postgres library (pooled at edge in production)
 *
 * NOTE: Benchmarks hit a real remote database. Absolute numbers reflect
 * network latency to Neon (ap-southeast-1). Relative comparisons between
 * drivers are meaningful.
 */
import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { env } from "cloudflare:test";
import { NeonEngine } from "../storage/neon_driver";
import { resetSchemaFlag } from "../storage/pg_base";
import type { StorageEngine } from "../storage/engine";

// ── Setup ────────────────────────────────────────────────────────────

const DSN = (env as any).POSTGRES_DSN as string;
const SKIP = !DSN;

const actorNeon = `pg-bench-neon-${Date.now()}`;

function createNeonEngine(): StorageEngine {
  return new NeonEngine({
    connectionString: DSN,
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
  warmup = 2,
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
    totalMs,
    avgMs: latencies.reduce((a, b) => a + b, 0) / latencies.length,
    p50Ms: latencies[Math.floor(latencies.length * 0.5)],
    p95Ms: latencies[Math.floor(latencies.length * 0.95)],
    p99Ms: latencies[Math.floor(latencies.length * 0.99)],
    opsPerSec: (iterations / totalMs) * 1000,
  };
}

// ── Cleanup ──────────────────────────────────────────────────────────

afterAll(async () => {
  if (SKIP) return;
  const { neon } = await import("@neondatabase/serverless");
  const sql = neon(DSN);
  for (const actor of [actorNeon]) {
    await sql.query("DELETE FROM stg_files WHERE owner = $1", [actor]);
    await sql.query("DELETE FROM stg_events WHERE actor = $1", [actor]);
    await sql.query("DELETE FROM stg_blobs WHERE actor = $1", [actor]);
    await sql.query("DELETE FROM stg_tx WHERE actor = $1", [actor]);
  }
});

// ── Neon Benchmarks ─────────────────────────────────────────────────

describe("NeonEngine benchmark", { timeout: 120_000 }, () => {
  const N_WRITE = 10;
  const N_READ = 15;

  beforeAll(() => {
    resetSchemaFlag();
    if (SKIP) console.log("\n⚠ POSTGRES_DSN not configured — skipping PG benchmarks");
  });

  it("write latency", async () => {
    if (SKIP) return;
    const engine = createNeonEngine();
    let i = 0;
    const result = await bench(
      "Neon: write",
      async () => {
        const data = new TextEncoder().encode(`bench content ${i++} ${Date.now()}`);
        await engine.write(actorNeon, `bench/file-${i}.txt`, data.buffer as ArrayBuffer, "text/plain");
      },
      N_WRITE,
    );

    console.log(
      `\nNeon write: avg=${result.avgMs.toFixed(0)}ms p50=${result.p50Ms.toFixed(0)}ms p95=${result.p95Ms.toFixed(0)}ms ops/sec=${result.opsPerSec.toFixed(1)}`,
    );
    expect(result.ops).toBe(N_WRITE);
  });

  it("head latency", async () => {
    if (SKIP) return;
    const engine = createNeonEngine();
    const result = await bench(
      "Neon: head",
      async () => {
        await engine.head(actorNeon, "bench/file-1.txt");
      },
      N_READ,
    );

    console.log(
      `Neon head: avg=${result.avgMs.toFixed(0)}ms p50=${result.p50Ms.toFixed(0)}ms p95=${result.p95Ms.toFixed(0)}ms ops/sec=${result.opsPerSec.toFixed(1)}`,
    );
    expect(result.ops).toBe(N_READ);
  });

  it("list latency", async () => {
    if (SKIP) return;
    const engine = createNeonEngine();
    const result = await bench(
      "Neon: list",
      async () => {
        await engine.list(actorNeon, { prefix: "bench/" });
      },
      N_READ,
    );

    console.log(
      `Neon list: avg=${result.avgMs.toFixed(0)}ms p50=${result.p50Ms.toFixed(0)}ms p95=${result.p95Ms.toFixed(0)}ms ops/sec=${result.opsPerSec.toFixed(1)}`,
    );
    expect(result.ops).toBe(N_READ);
  });

  it("search latency", async () => {
    if (SKIP) return;
    const engine = createNeonEngine();
    const result = await bench(
      "Neon: search",
      async () => {
        await engine.search(actorNeon, "file");
      },
      N_READ,
    );

    console.log(
      `Neon search: avg=${result.avgMs.toFixed(0)}ms p50=${result.p50Ms.toFixed(0)}ms p95=${result.p95Ms.toFixed(0)}ms ops/sec=${result.opsPerSec.toFixed(1)}`,
    );
    expect(result.ops).toBe(N_READ);
  });

  it("stats latency", async () => {
    if (SKIP) return;
    const engine = createNeonEngine();
    const result = await bench(
      "Neon: stats",
      async () => {
        await engine.stats(actorNeon);
      },
      N_READ,
    );

    console.log(
      `Neon stats: avg=${result.avgMs.toFixed(0)}ms p50=${result.p50Ms.toFixed(0)}ms p95=${result.p95Ms.toFixed(0)}ms ops/sec=${result.opsPerSec.toFixed(1)}`,
    );
    expect(result.ops).toBe(N_READ);
  });

  it("delete latency", async () => {
    if (SKIP) return;
    const engine = createNeonEngine();
    let i = 0;
    const result = await bench(
      "Neon: delete",
      async () => {
        await engine.delete(actorNeon, [`bench/file-${++i}.txt`]);
      },
      Math.min(N_WRITE, 10),
    );

    console.log(
      `Neon delete: avg=${result.avgMs.toFixed(0)}ms p50=${result.p50Ms.toFixed(0)}ms p95=${result.p95Ms.toFixed(0)}ms ops/sec=${result.opsPerSec.toFixed(1)}`,
    );
    expect(result.ops).toBeGreaterThan(0);
  });

  it("move latency", async () => {
    if (SKIP) return;
    const engine = createNeonEngine();
    // Seed files to move
    for (let i = 0; i < 5; i++) {
      const data = new TextEncoder().encode(`move me ${i}`);
      await engine.write(actorNeon, `bench/move-src-${i}.txt`, data.buffer as ArrayBuffer, "text/plain");
    }
    let i = 0;
    const result = await bench(
      "Neon: move",
      async () => {
        await engine.move(actorNeon, `bench/move-src-${i}.txt`, `bench/move-dst-${i}.txt`);
        i++;
      },
      5,
      0,
    );

    console.log(
      `Neon move: avg=${result.avgMs.toFixed(0)}ms p50=${result.p50Ms.toFixed(0)}ms p95=${result.p95Ms.toFixed(0)}ms ops/sec=${result.opsPerSec.toFixed(1)}`,
    );
    expect(result.ops).toBe(5);
  });
});
