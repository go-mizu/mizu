#!/usr/bin/env node

/**
 * Tests for @liteio/storage-cli
 *
 * Tests the upload/download flow with SHA-256 content-hash dedup,
 * using a mock HTTP server to simulate the storage API.
 *
 * Run: node test/cli.test.mjs
 */

import { createHash } from "node:crypto";
import { createServer } from "node:http";
import { readFileSync, writeFileSync, mkdirSync, rmSync, existsSync } from "node:fs";
import { join } from "node:path";
import { execFileSync, execFile } from "node:child_process";
import { tmpdir } from "node:os";

const CLI = join(import.meta.dirname, "..", "bin", "storage.mjs");
const TMP = join(tmpdir(), `storage-cli-test-${Date.now()}`);

let passed = 0;
let failed = 0;

function assert(condition, msg) {
  if (condition) {
    passed++;
    console.log(`  \x1b[32m✓\x1b[0m ${msg}`);
  } else {
    failed++;
    console.log(`  \x1b[31m✗\x1b[0m ${msg}`);
  }
}

function assertEqual(actual, expected, msg) {
  if (actual === expected) {
    passed++;
    console.log(`  \x1b[32m✓\x1b[0m ${msg}`);
  } else {
    failed++;
    console.log(`  \x1b[31m✗\x1b[0m ${msg}: expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
}

function assertIncludes(str, substr, msg) {
  if (str.includes(substr)) {
    passed++;
    console.log(`  \x1b[32m✓\x1b[0m ${msg}`);
  } else {
    failed++;
    console.log(`  \x1b[31m✗\x1b[0m ${msg}: ${JSON.stringify(str)} does not include ${JSON.stringify(substr)}`);
  }
}

// Sync run — for tests that don't need a mock server (version, help, auth, edge cases)
// Uses spawnSync to capture both stdout and stderr regardless of exit code.
import { spawnSync } from "node:child_process";

function run(args, env = {}) {
  const result = spawnSync(process.execPath, [CLI, ...args], {
    encoding: "utf8",
    env: { ...process.env, ...env, NO_COLOR: "1", XDG_CONFIG_HOME: join(TMP, ".config") },
    timeout: 10000,
  });
  return {
    stdout: result.stdout || "",
    stderr: result.stderr || "",
    code: result.status ?? 1,
  };
}

// Async run — for tests with mock servers (execFileSync blocks the event loop,
// preventing Node's HTTP server from responding to the CLI subprocess)
function runAsync(args, env = {}) {
  return new Promise((resolve) => {
    execFile(process.execPath, [CLI, ...args], {
      encoding: "utf8",
      env: { ...process.env, ...env, NO_COLOR: "1", XDG_CONFIG_HOME: join(TMP, ".config") },
      timeout: 10000,
    }, (err, stdout, stderr) => {
      if (err) {
        resolve({
          stdout: stdout || err.stdout || "",
          stderr: stderr || err.stderr || "",
          code: err.code === "ERR_CHILD_PROCESS_STDIO_MAXBUFFER" ? 1 : (err.code ?? err.status ?? 1),
        });
      } else {
        resolve({ stdout: stdout || "", stderr: stderr || "", code: 0 });
      }
    });
  });
}

// ── SHA-256 computation tests ──────────────────────────────────────

console.log("\n\x1b[1mSHA-256 Computation\x1b[0m");

{
  const empty = createHash("sha256").update(Buffer.from("")).digest("hex");
  assertEqual(empty, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "SHA-256 of empty string");
}

{
  const hello = createHash("sha256").update(Buffer.from("hello")).digest("hex");
  assertEqual(hello, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", "SHA-256 of 'hello'");
}

{
  const data = Buffer.from("test dedup content");
  const h1 = createHash("sha256").update(data).digest("hex");
  const h2 = createHash("sha256").update(data).digest("hex");
  assertEqual(h1, h2, "same content produces same hash");
  assertEqual(h1.length, 64, "hash is 64 hex chars");
}

{
  const h1 = createHash("sha256").update(Buffer.from("content A")).digest("hex");
  const h2 = createHash("sha256").update(Buffer.from("content B")).digest("hex");
  assert(h1 !== h2, "different content produces different hash");
}

// ── Mock server tests ──────────────────────────────────────────────

console.log("\n\x1b[1mCLI Version\x1b[0m");

{
  const result = run(["--version"]);
  assertIncludes(result.stdout.trim(), "2.1.0", "version shows 2.1.0");
}

console.log("\n\x1b[1mUpload with Dedup (Mock Server)\x1b[0m");

// Create a mock server that simulates the storage API
async function withMockServer(handler, fn) {
  const server = createServer(handler);
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  const port = server.address().port;
  try {
    await fn(`http://127.0.0.1:${port}`);
  } finally {
    server.close();
  }
}

// Setup temp dir
mkdirSync(TMP, { recursive: true });

// Test: upload with dedup (blob already exists)
await withMockServer((req, res) => {
  let body = "";
  req.on("data", (chunk) => { body += chunk; });
  req.on("end", () => {
    if (req.method === "POST" && req.url === "/files/uploads") {
      const parsed = JSON.parse(body);
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({
        deduplicated: true,
        path: parsed.path,
        name: parsed.path.split("/").pop(),
        size: 42,
        tx: 7,
        time: Date.now(),
      }));
    } else {
      res.writeHead(404);
      res.end(JSON.stringify({ error: "not_found" }));
    }
  });
}, async (endpoint) => {
  const testFile = join(TMP, "dedup-test.txt");
  writeFileSync(testFile, "hello dedup");

  const result = await runAsync(
    ["put", testFile, "docs/dedup.txt", "--json"],
    { STORAGE_ENDPOINT: endpoint, STORAGE_TOKEN: "test-token" },
  );

  assert(result.code === 0, "dedup upload exits with 0");
  if (result.stdout.trim()) {
    try {
      const json = JSON.parse(result.stdout.trim());
      assertEqual(json.deduplicated, true, "response has deduplicated=true");
      assertEqual(json.path, "docs/dedup.txt", "response has correct path");
      assert(json.content_hash && json.content_hash.length === 64, "response includes content_hash");
    } catch (e) {
      assertEqual(true, false, `parse dedup JSON output: ${e.message}`);
    }
  }
});

// Test: upload full flow (not deduplicated)
await withMockServer((req, res) => {
  let body = "";
  req.on("data", (chunk) => { body += chunk; });
  req.on("end", () => {
    if (req.method === "POST" && req.url === "/files/uploads") {
      const parsed = JSON.parse(body);
      // Not deduplicated - return presigned URL (pointing back to our mock)
      const port = req.socket.localPort;
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({
        url: `http://127.0.0.1:${port}/r2-upload`,
        content_type: parsed.content_type,
        content_hash: parsed.content_hash,
        expires_in: 3600,
      }));
    } else if (req.method === "PUT" && req.url === "/r2-upload") {
      res.writeHead(200);
      res.end();
    } else if (req.method === "POST" && req.url === "/files/uploads/complete") {
      const parsed = JSON.parse(body);
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({
        path: parsed.path,
        name: parsed.path.split("/").pop(),
        size: 15,
        tx: 1,
        time: Date.now(),
      }));
    } else {
      res.writeHead(404);
      res.end(JSON.stringify({ error: "not_found" }));
    }
  });
}, async (endpoint) => {
  const testFile = join(TMP, "full-upload.txt");
  writeFileSync(testFile, "full upload test");

  const result = await runAsync(
    ["put", testFile, "docs/full.txt", "--json"],
    { STORAGE_ENDPOINT: endpoint, STORAGE_TOKEN: "test-token" },
  );

  assert(result.code === 0, "full upload exits with 0");
  if (result.stdout.trim()) {
    try {
      const json = JSON.parse(result.stdout.trim());
      assert(!json.deduplicated, "full upload is not deduplicated");
      assertEqual(json.path, "docs/full.txt", "full upload has correct path");
      assert(json.content_hash && json.content_hash.length === 64, "full upload includes content_hash");
    } catch (e) {
      assertEqual(true, false, `parse full upload JSON output: ${e.message}`);
    }
  }
});

// Test: upload sends content_hash in initiate request
await withMockServer((req, res) => {
  let body = "";
  req.on("data", (chunk) => { body += chunk; });
  req.on("end", () => {
    if (req.method === "POST" && req.url === "/files/uploads") {
      const parsed = JSON.parse(body);
      // Verify content_hash was sent
      if (!parsed.content_hash || parsed.content_hash.length !== 64) {
        res.writeHead(400, { "Content-Type": "application/json" });
        res.end(JSON.stringify({ error: "bad_request", message: "missing or invalid content_hash" }));
        return;
      }
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({
        deduplicated: true,
        path: parsed.path,
        name: "test.txt",
        size: 10,
        tx: 1,
        time: Date.now(),
      }));
    } else {
      res.writeHead(404);
      res.end();
    }
  });
}, async (endpoint) => {
  const testFile = join(TMP, "hash-check.txt");
  writeFileSync(testFile, "hash check");

  const result = await runAsync(
    ["put", testFile, "test.txt", "--json"],
    { STORAGE_ENDPOINT: endpoint, STORAGE_TOKEN: "test-token" },
  );

  assert(result.code === 0, "upload with content_hash exits with 0 (server verified hash)");
});

// Test: download (list/get/cat flows)
console.log("\n\x1b[1mList / Stats (Mock Server)\x1b[0m");

await withMockServer((req, res) => {
  let body = "";
  req.on("data", (chunk) => { body += chunk; });
  req.on("end", () => {
    if (req.method === "GET" && req.url.startsWith("/files/stats")) {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ files: 42, bytes: 1048576 }));
    } else if (req.method === "GET" && req.url.startsWith("/files/search")) {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ query: "test", results: [{ path: "docs/test.txt", name: "test.txt" }] }));
    } else if (req.method === "GET" && req.url.startsWith("/files?") || req.url === "/files") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({
        prefix: "/",
        entries: [
          { name: "docs/", type: "directory" },
          { name: "hello.txt", type: "text/plain", size: 13, updated_at: Date.now() },
        ],
        truncated: false,
      }));
    } else {
      res.writeHead(404);
      res.end(JSON.stringify({ error: "not_found" }));
    }
  });
}, async (endpoint) => {
  const statResult = await runAsync(
    ["stat", "--json"],
    { STORAGE_ENDPOINT: endpoint, STORAGE_TOKEN: "test-token" },
  );
  assert(statResult.code === 0, "stat command exits with 0");
  if (statResult.stdout.trim()) {
    const json = JSON.parse(statResult.stdout.trim());
    assertEqual(json.files, 42, "stat shows 42 files");
    assertEqual(json.bytes, 1048576, "stat shows correct bytes");
  }

  const findResult = await runAsync(
    ["find", "test", "--json"],
    { STORAGE_ENDPOINT: endpoint, STORAGE_TOKEN: "test-token" },
  );
  assert(findResult.code === 0, "find command exits with 0");
  if (findResult.stdout.trim()) {
    const json = JSON.parse(findResult.stdout.trim());
    assert(json.results && json.results.length > 0, "find returns results");
  }

  const lsResult = await runAsync(
    ["ls", "--json"],
    { STORAGE_ENDPOINT: endpoint, STORAGE_TOKEN: "test-token" },
  );
  assert(lsResult.code === 0, "ls command exits with 0");
  if (lsResult.stdout.trim()) {
    const json = JSON.parse(lsResult.stdout.trim());
    assert(json.entries && json.entries.length > 0, "ls returns entries");
  }
});

// Test: auth required
console.log("\n\x1b[1mAuth Required\x1b[0m");

{
  const result = run(["ls"], { STORAGE_TOKEN: "", STORAGE_ENDPOINT: "http://localhost:1" });
  assert(result.code === 3, "ls without token exits with EXIT_AUTH (3)");
}

{
  const result = run(["put", "/dev/null", "test.txt"], { STORAGE_TOKEN: "", STORAGE_ENDPOINT: "http://localhost:1" });
  assert(result.code === 3, "put without token exits with EXIT_AUTH (3)");
}

{
  const result = run(["get", "test.txt"], { STORAGE_TOKEN: "", STORAGE_ENDPOINT: "http://localhost:1" });
  assert(result.code === 3, "get without token exits with EXIT_AUTH (3)");
}

// Test: help
console.log("\n\x1b[1mHelp & Version\x1b[0m");

{
  const result = run(["--help"]);
  const helpText = result.stderr + result.stdout;
  assertIncludes(helpText, "storage", "help output mentions storage");
  assertIncludes(helpText, "put", "help output mentions put command");
}

{
  const result = run(["--version"]);
  assertEqual(result.stdout.trim(), "storage 2.1.0", "version output is correct");
}

// Test: stdin upload requires path
console.log("\n\x1b[1mEdge Cases\x1b[0m");

{
  const result = run(["put", "-"], { STORAGE_TOKEN: "test", STORAGE_ENDPOINT: "http://localhost:1" });
  assert(result.code === 2, "put stdin without path exits with EXIT_USAGE (2)");
}

{
  const result = run(["put"], { STORAGE_TOKEN: "test", STORAGE_ENDPOINT: "http://localhost:1" });
  assert(result.code === 2, "put without args exits with EXIT_USAGE (2)");
}

{
  const result = run(["get"], { STORAGE_TOKEN: "test", STORAGE_ENDPOINT: "http://localhost:1" });
  assert(result.code === 2, "get without args exits with EXIT_USAGE (2)");
}

{
  const result = run(["put", "/nonexistent/file.txt", "test.txt"], { STORAGE_TOKEN: "test", STORAGE_ENDPOINT: "http://localhost:1" });
  assert(result.code === 4, "put nonexistent file exits with EXIT_NOT_FOUND (4)");
}

// ── Cleanup ────────────────────────────────────────────────────────

try { rmSync(TMP, { recursive: true }); } catch {}

// ── Summary ────────────────────────────────────────────────────────

console.log(`\n\x1b[1mResults: ${passed} passed, ${failed} failed\x1b[0m\n`);
process.exit(failed > 0 ? 1 : 0);
