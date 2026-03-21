/**
 * PostgreSQL Driver Integration Tests
 *
 * Tests the NeonEngine (and by extension PgEngineBase) against a real
 * Neon PostgreSQL database. The SQL logic is shared with HyperdriveEngine
 * via PgEngineBase, so these tests cover both drivers' SQL layer.
 *
 * Requires POSTGRES_DSN in vitest config bindings (loaded from ~/.local.env).
 * Skips gracefully if not configured.
 */
import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { env } from "cloudflare:test";
import { NeonEngine } from "../storage/neon_driver";
import { resetSchemaFlag } from "../storage/pg_base";
import type { StorageEngine } from "../storage/engine";

// ── Setup ────────────────────────────────────────────────────────────

const DSN = (env as any).POSTGRES_DSN as string;
const SKIP = !DSN;
const actor = `pg-test-${Date.now()}`;

function createEngine(): StorageEngine {
  return new NeonEngine({
    connectionString: DSN,
    bucket: env.BUCKET,
    r2Endpoint: "https://test.r2.cloudflarestorage.com",
    r2AccessKeyId: "test-key-id",
    r2SecretAccessKey: "test-secret-key",
  });
}

function skipIf(condition: boolean, name: string, fn: () => Promise<void>) {
  if (condition) {
    it.skip(name, fn);
  } else {
    it(name, fn);
  }
}

// ── Cleanup ──────────────────────────────────────────────────────────

afterAll(async () => {
  if (SKIP) return;
  // Clean up test data from Postgres
  const { neon } = await import("@neondatabase/serverless");
  const sql = neon(DSN);
  await sql.query("DELETE FROM stg_files WHERE owner = $1", [actor]);
  await sql.query("DELETE FROM stg_events WHERE actor = $1", [actor]);
  await sql.query("DELETE FROM stg_blobs WHERE actor = $1", [actor]);
  await sql.query("DELETE FROM stg_tx WHERE actor = $1", [actor]);
});

// ── Schema ──────────────────────────────────────────────────────────

describe("PostgreSQL driver (NeonEngine)", () => {
  beforeAll(() => {
    resetSchemaFlag(); // Force schema creation for test isolation
  });

  skipIf(SKIP, "creates schema on first access", async () => {
    const engine = createEngine();
    // ensureSchema is called implicitly by any operation
    const stats = await engine.stats(actor);
    expect(stats).toEqual({ files: 0, bytes: 0 });
  });

  // ── Write ─────────────────────────────────────────────────────────

  skipIf(SKIP, "writes a file", async () => {
    const engine = createEngine();
    const data = new TextEncoder().encode("hello postgres");
    const result = await engine.write(actor, "test/hello.txt", data.buffer as ArrayBuffer, "text/plain");

    expect(result.tx).toBeGreaterThan(0);
    expect(result.time).toBeGreaterThan(0);
    expect(result.size).toBe(14);
  });

  skipIf(SKIP, "overwrites a file (same path)", async () => {
    const engine = createEngine();
    const data = new TextEncoder().encode("updated content");
    const result = await engine.write(actor, "test/hello.txt", data.buffer as ArrayBuffer, "text/plain");

    expect(result.tx).toBeGreaterThan(1);
    expect(result.size).toBe(15);
  });

  skipIf(SKIP, "deduplicates identical content", async () => {
    const engine = createEngine();
    const data = new TextEncoder().encode("shared content");
    await engine.write(actor, "test/file-a.txt", data.buffer as ArrayBuffer, "text/plain");
    await engine.write(actor, "test/file-b.txt", data.buffer as ArrayBuffer, "text/plain");

    const statsResult = await engine.stats(actor);
    expect(statsResult.files).toBeGreaterThanOrEqual(3);
  });

  // ── Head ──────────────────────────────────────────────────────────

  skipIf(SKIP, "returns file metadata", async () => {
    const engine = createEngine();
    const meta = await engine.head(actor, "test/hello.txt");

    expect(meta).not.toBeNull();
    expect(meta!.name).toBe("hello.txt");
    expect(meta!.type).toBe("text/plain");
    expect(meta!.size).toBe(15); // Updated content
    expect(meta!.tx).toBeGreaterThan(0);
  });

  skipIf(SKIP, "returns null for missing file", async () => {
    const engine = createEngine();
    const meta = await engine.head(actor, "test/nonexistent.txt");
    expect(meta).toBeNull();
  });

  // ── List ──────────────────────────────────────────────────────────

  skipIf(SKIP, "lists files under prefix", async () => {
    const engine = createEngine();
    const { entries, truncated } = await engine.list(actor, { prefix: "test/" });

    expect(truncated).toBe(false);
    expect(entries.length).toBeGreaterThanOrEqual(3);

    const names = entries.map((e) => e.name);
    expect(names).toContain("hello.txt");
    expect(names).toContain("file-a.txt");
    expect(names).toContain("file-b.txt");
  });

  skipIf(SKIP, "lists with pagination", async () => {
    const engine = createEngine();
    const { entries, truncated } = await engine.list(actor, { prefix: "test/", limit: 1 });

    expect(entries.length).toBe(1);
    expect(truncated).toBe(true);
  });

  skipIf(SKIP, "lists directories", async () => {
    const engine = createEngine();
    // Write a nested file
    const data = new TextEncoder().encode("nested");
    await engine.write(actor, "test/sub/nested.txt", data.buffer as ArrayBuffer, "text/plain");

    const { entries } = await engine.list(actor, { prefix: "test/" });
    const dirEntry = entries.find((e) => e.name === "sub/");
    expect(dirEntry).toBeDefined();
    expect(dirEntry!.type).toBe("directory");
  });

  // ── Search ────────────────────────────────────────────────────────

  skipIf(SKIP, "searches by filename", async () => {
    const engine = createEngine();
    const results = await engine.search(actor, "hello");

    expect(results.length).toBeGreaterThanOrEqual(1);
    expect(results[0].name).toBe("hello.txt");
  });

  skipIf(SKIP, "searches with prefix filter", async () => {
    const engine = createEngine();
    const results = await engine.search(actor, "nested", { prefix: "test/sub/" });

    expect(results.length).toBe(1);
    expect(results[0].path).toBe("test/sub/nested.txt");
  });

  // ── Stats ─────────────────────────────────────────────────────────

  skipIf(SKIP, "returns storage stats", async () => {
    const engine = createEngine();
    const stats = await engine.stats(actor);

    expect(stats.files).toBeGreaterThanOrEqual(4); // hello, file-a, file-b, sub/nested
    expect(stats.bytes).toBeGreaterThan(0);
  });

  // ── Move ──────────────────────────────────────────────────────────

  skipIf(SKIP, "moves a file", async () => {
    const engine = createEngine();
    const result = await engine.move(actor, "test/file-a.txt", "test/file-a-moved.txt");

    expect(result.tx).toBeGreaterThan(0);

    const oldMeta = await engine.head(actor, "test/file-a.txt");
    expect(oldMeta).toBeNull();

    const newMeta = await engine.head(actor, "test/file-a-moved.txt");
    expect(newMeta).not.toBeNull();
    expect(newMeta!.name).toBe("file-a-moved.txt");
  });

  skipIf(SKIP, "throws on moving nonexistent file", async () => {
    const engine = createEngine();
    await expect(
      engine.move(actor, "test/nonexistent.txt", "test/dest.txt"),
    ).rejects.toThrow("Source not found");
  });

  // ── Delete ────────────────────────────────────────────────────────

  skipIf(SKIP, "deletes a single file", async () => {
    const engine = createEngine();
    const result = await engine.delete(actor, ["test/file-b.txt"]);

    expect(result.deleted).toBe(1);
    expect(result.tx).toBeGreaterThan(0);

    const meta = await engine.head(actor, "test/file-b.txt");
    expect(meta).toBeNull();
  });

  skipIf(SKIP, "deletes a folder recursively", async () => {
    const engine = createEngine();
    // Write extra files in subfolder
    const data = new TextEncoder().encode("x");
    await engine.write(actor, "test/sub/a.txt", data.buffer as ArrayBuffer, "text/plain");
    await engine.write(actor, "test/sub/b.txt", data.buffer as ArrayBuffer, "text/plain");

    const result = await engine.delete(actor, ["test/sub/"]);
    expect(result.deleted).toBeGreaterThanOrEqual(2);

    const { entries } = await engine.list(actor, { prefix: "test/sub/" });
    expect(entries.length).toBe(0);
  });

  // ── Log ───────────────────────────────────────────────────────────

  skipIf(SKIP, "returns event log", async () => {
    const engine = createEngine();
    const events = await engine.log(actor);

    expect(events.length).toBeGreaterThan(0);
    // Most recent event should be the folder delete
    const actions = events.map((e) => e.action);
    expect(actions).toContain("write");
    expect(actions).toContain("delete");
  });

  skipIf(SKIP, "filters log by path", async () => {
    const engine = createEngine();
    const events = await engine.log(actor, { path: "test/hello.txt" });

    expect(events.length).toBeGreaterThan(0);
    expect(events.every((e) => e.path === "test/hello.txt")).toBe(true);
  });

  // ── AllNames ──────────────────────────────────────────────────────

  skipIf(SKIP, "returns all file names", async () => {
    const engine = createEngine();
    const names = await engine.allNames(actor);

    expect(names.length).toBeGreaterThan(0);
    expect(names.some((n) => n.name === "hello.txt")).toBe(true);
  });
});
