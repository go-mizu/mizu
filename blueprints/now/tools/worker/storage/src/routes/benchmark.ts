// ── Benchmark endpoint — measures driver performance from the edge ────
//
// GET /benchmark?driver=d1|neon
// Requires X-Benchmark-Key header matching SIGNING_KEY.

import type { App } from "../types";
import { D1Engine } from "../storage/d1_driver";
import { NeonEngine } from "../storage/neon_driver";
import { HyperdriveEngine } from "../storage/hyperdrive_driver";
import { PgEngineBase, type QueryFn } from "../storage/pg_base";
import type { StorageEngine } from "../storage/engine";

interface BenchResult {
  op: string;
  n: number;
  avg_ms: number;
  p50_ms: number;
  p95_ms: number;
  ops_per_sec: number;
}

async function measure(
  name: string,
  fn: () => Promise<void>,
  iterations: number,
  warmup = 1,
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
    op: name,
    n: iterations,
    avg_ms: Math.round(latencies.reduce((a, b) => a + b, 0) / latencies.length),
    p50_ms: Math.round(latencies[Math.floor(latencies.length * 0.5)]),
    p95_ms: Math.round(latencies[Math.floor(latencies.length * 0.95)]),
    ops_per_sec: Math.round((iterations / totalMs) * 1000 * 10) / 10,
  };
}

export function register(app: App) {
  app.get("/benchmark", async (c) => {
    // Only available in DEV_MODE
    if (c.env.DEV_MODE !== "1") {
      return c.json({ error: "not_found", message: "Not found" }, 404);
    }

    // Auth: require BENCHMARK_KEY (or fall back to SIGNING_KEY)
    const key = c.req.header("X-Benchmark-Key");
    const validKey = c.env.BENCHMARK_KEY || c.env.SIGNING_KEY;
    if (!key || key !== validKey) {
      return c.json({ error: "unauthorized", message: "X-Benchmark-Key required" }, 401);
    }

    const driverName = c.req.query("driver") || c.env.STORAGE_DRIVER || "d1";
    const r2Config = {
      bucket: c.env.BUCKET,
      r2Endpoint: c.env.R2_ENDPOINT,
      r2AccessKeyId: c.env.R2_ACCESS_KEY_ID,
      r2SecretAccessKey: c.env.R2_SECRET_ACCESS_KEY,
      r2BucketName: c.env.R2_BUCKET_NAME,
    };

    let engine: StorageEngine;
    try {
      switch (driverName) {
        case "neon":
          if (!c.env.POSTGRES_DSN) return c.json({ error: "POSTGRES_DSN not configured" }, 400);
          engine = new NeonEngine({ connectionString: c.env.POSTGRES_DSN, ...r2Config });
          break;
        case "neon_eu":
          if (!c.env.POSTGRES_EC1_DSN) return c.json({ error: "POSTGRES_EC1_DSN not configured" }, 400);
          engine = new NeonEngine({ connectionString: c.env.POSTGRES_EC1_DSN, ...r2Config });
          break;
        case "hyperdrive":
          if (!c.env.HYPERDRIVE) return c.json({ error: "HYPERDRIVE not configured" }, 400);
          engine = new HyperdriveEngine({ connectionString: c.env.HYPERDRIVE.connectionString, ...r2Config });
          break;
        case "d1":
        default:
          engine = new D1Engine({ db: c.env.DB, ...r2Config });
          break;
      }
    } catch (e: any) {
      return c.json({ error: "engine_init_failed", message: e.message }, 500);
    }

    const actor = `__benchmark_${driverName}_${Date.now()}`;
    const results: BenchResult[] = [];
    const N_WRITE = 5;
    const N_READ = 8;

    try {
      // ── Write ──────────────────────────────────────────────────
      let wi = 0;
      results.push(
        await measure("write", async () => {
          const data = new TextEncoder().encode(`bench ${wi++} ${Date.now()}`);
          await engine.write(actor, `bench/f-${wi}.txt`, data.buffer as ArrayBuffer, "text/plain");
        }, N_WRITE),
      );

      // ── Write Meta (SQL only, no R2) ────────────────────────────
      const hasBenchMeta = "benchWriteMeta" in engine;
      if (hasBenchMeta) {
        let wmi = 0;
        results.push(
          await measure("write_meta", async () => {
            await (engine as any).benchWriteMeta(actor, `bench/wm-${wmi++}.txt`, 64, "text/plain");
          }, N_WRITE),
        );
      }

      // ── Head ───────────────────────────────────────────────────
      results.push(
        await measure("head", async () => {
          await engine.head(actor, "bench/f-1.txt");
        }, N_READ),
      );

      // ── List ───────────────────────────────────────────────────
      results.push(
        await measure("list", async () => {
          await engine.list(actor, { prefix: "bench/" });
        }, N_READ),
      );

      // ── Search ─────────────────────────────────────────────────
      results.push(
        await measure("search", async () => {
          await engine.search(actor, "f-");
        }, N_READ),
      );

      // ── Stats ──────────────────────────────────────────────────
      results.push(
        await measure("stats", async () => {
          await engine.stats(actor);
        }, N_READ),
      );

      // ── Move ───────────────────────────────────────────────────
      for (let i = 0; i < 3; i++) {
        const data = new TextEncoder().encode(`move-${i}`);
        await engine.write(actor, `bench/mv-${i}.txt`, data.buffer as ArrayBuffer, "text/plain");
      }
      let mi = 0;
      results.push(
        await measure("move", async () => {
          await engine.move(actor, `bench/mv-${mi}.txt`, `bench/mv-dst-${mi}.txt`);
          mi++;
        }, 3, 0),
      );

      // ── Delete ─────────────────────────────────────────────────
      let di = 0;
      results.push(
        await measure("delete", async () => {
          await engine.delete(actor, [`bench/f-${++di}.txt`]);
        }, Math.min(N_WRITE, 4)),
      );

      // ── Cleanup ────────────────────────────────────────────────
      await engine.delete(actor, ["bench/"]);
    } catch (e: any) {
      return c.json({
        error: "benchmark_failed",
        message: e.message,
        driver: driverName,
        partial_results: results,
      }, 500);
    }

    return c.json({
      driver: driverName,
      region: (c.req.raw as any).cf?.colo || "unknown",
      timestamp: new Date().toISOString(),
      results,
    });
  });
}
