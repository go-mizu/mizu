#!/usr/bin/env node

// storage — CLI for Storage API
// https://storage.liteio.dev/cli
//
// Zero dependencies. Works with Node 18+, Bun, Deno.
// Install: npx @liteio/storage-cli

import { createHash, randomBytes } from "node:crypto";
import { readFileSync, writeFileSync, mkdirSync, unlinkSync, statSync, createWriteStream } from "node:fs";
import { join, basename, dirname, extname } from "node:path";
import { homedir } from "node:os";
import { createServer } from "node:http";
import { execFile } from "node:child_process";
import { Buffer } from "node:buffer";
import process from "node:process";

// ── Constants ──────────────────────────────────────────────────────────

const VERSION = "1.1.0";
const DEFAULT_ENDPOINT = "https://storage.liteio.dev";
const OAUTH_CLIENT_ID = "storage-cli";
const OAUTH_SCOPE = "storage:read storage:write storage:admin";

const EXIT_ERROR = 1;
const EXIT_USAGE = 2;
const EXIT_AUTH = 3;
const EXIT_NOT_FOUND = 4;
const EXIT_CONFLICT = 5;
const EXIT_PERMISSION = 6;
const EXIT_NETWORK = 7;

// ── Config ─────────────────────────────────────────────────────────────

function configDir() {
  const xdg = process.env.XDG_CONFIG_HOME;
  return join(xdg || join(homedir(), ".config"), "storage");
}
function tokenFile() { return join(configDir(), "token"); }

function loadConfig(flagToken, flagEndpoint) {
  const cfg = { endpoint: DEFAULT_ENDPOINT, token: "" };

  // 1. Config file
  try {
    const raw = readFileSync(join(configDir(), "config"), "utf8");
    for (const line of raw.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#")) continue;
      const eq = trimmed.indexOf("=");
      if (eq < 0) continue;
      const k = trimmed.slice(0, eq).trim();
      const v = trimmed.slice(eq + 1).trim();
      if (k === "endpoint") cfg.endpoint = v;
    }
  } catch {}

  // 2. Environment
  if (process.env.STORAGE_ENDPOINT) cfg.endpoint = process.env.STORAGE_ENDPOINT;
  if (process.env.STORAGE_TOKEN) cfg.token = process.env.STORAGE_TOKEN;

  // 3. Token file
  if (!cfg.token) {
    try { cfg.token = readFileSync(tokenFile(), "utf8").trim(); } catch {}
  }

  // 4. Flags (highest priority)
  if (flagEndpoint) cfg.endpoint = flagEndpoint;
  if (flagToken) cfg.token = flagToken;

  return cfg;
}

function saveToken(token) {
  const dir = configDir();
  mkdirSync(dir, { recursive: true });
  writeFileSync(tokenFile(), token, { mode: 0o600 });
}

function removeToken() {
  try { unlinkSync(tokenFile()); return true; } catch { return false; }
}

// ── Output ─────────────────────────────────────────────────────────────

const isTTY = process.stderr.isTTY ?? false;
let noColor = !isTTY || !!process.env.NO_COLOR || process.env.TERM === "dumb";
let quiet = false;
let jsonOutput = false;

const esc = (code) => noColor ? "" : `\x1b[${code}m`;
const fmt = {
  bold:   (s) => `${esc(1)}${s}${esc(0)}`,
  dim:    (s) => `${esc(2)}${s}${esc(0)}`,
  red:    (s) => `${esc(31)}${s}${esc(0)}`,
  green:  (s) => `${esc(32)}${s}${esc(0)}`,
  yellow: (s) => `${esc(33)}${s}${esc(0)}`,
  cyan:   (s) => `${esc(36)}${s}${esc(0)}`,
};

function info(action, detail) {
  if (quiet) return;
  process.stderr.write(`  ${fmt.green(action)} ${detail}\n`);
}

function warn(msg) { process.stderr.write(`${fmt.yellow("warning:")} ${msg}\n`); }

function printError(msg, reason, hint) {
  process.stderr.write(`${fmt.red("error:")} ${msg}\n`);
  if (reason) process.stderr.write(`  ${reason}\n`);
  if (hint) process.stderr.write(`  ${hint}\n`);
}

function die(msg, reason, hint, code = EXIT_ERROR) {
  printError(msg, reason, hint);
  process.exit(code);
}

function humanSize(bytes) {
  if (bytes >= 1 << 30) return `${(bytes / (1 << 30)).toFixed(1)} GB`;
  if (bytes >= 1 << 20) return `${(bytes / (1 << 20)).toFixed(1)} MB`;
  if (bytes >= 1 << 10) return `${(bytes / (1 << 10)).toFixed(1)} KB`;
  return `${bytes} B`;
}

function relativeTime(epochMs) {
  if (!epochMs) return "-";
  const d = Date.now() - epochMs;
  if (d < 60000) return "just now";
  if (d < 3600000) return `${Math.floor(d / 60000)}m ago`;
  if (d < 86400000) return `${Math.floor(d / 3600000)}h ago`;
  if (d < 604800000) return `${Math.floor(d / 86400000)}d ago`;
  return `${Math.floor(d / 604800000)}w ago`;
}

function printJSON(data) {
  process.stdout.write(JSON.stringify(data, null, 2) + "\n");
}

// ── HTTP Client ────────────────────────────────────────────────────────

class APIError extends Error {
  constructor(status, code, message) {
    super(message || `HTTP ${status}`);
    this.status = status;
    this.code = code;
  }
  get exitCode() {
    if (this.status === 401) return EXIT_AUTH;
    if (this.status === 403) return EXIT_PERMISSION;
    if (this.status === 404) return EXIT_NOT_FOUND;
    if (this.status === 409) return EXIT_CONFLICT;
    return EXIT_ERROR;
  }
}

class CLIError extends Error {
  constructor(msg, hint, code = EXIT_ERROR) {
    super(msg);
    this.hint = hint;
    this.exitCode = code;
  }
}

async function request(cfg, method, path, opts = {}) {
  const url = cfg.endpoint + path;
  const headers = { ...opts.headers };
  if (cfg.token) headers["Authorization"] = `Bearer ${cfg.token}`;

  const fetchOpts = { method, headers };
  if (opts.body !== undefined) {
    if (typeof opts.body === "string" || opts.body instanceof Buffer) {
      fetchOpts.body = opts.body;
    } else if (opts.body && typeof opts.body.pipe === "function") {
      const chunks = [];
      for await (const chunk of opts.body) chunks.push(chunk);
      fetchOpts.body = Buffer.concat(chunks);
    } else if (opts.body instanceof ReadableStream) {
      fetchOpts.body = opts.body;
    } else {
      fetchOpts.body = JSON.stringify(opts.body);
      if (!headers["Content-Type"]) headers["Content-Type"] = "application/json";
    }
  }

  let res;
  try {
    res = await fetch(url, fetchOpts);
  } catch {
    throw new CLIError("network error", `Could not reach ${cfg.endpoint}\nCheck your internet connection and try again`, EXIT_NETWORK);
  }

  if (opts.raw) return res;
  if (res.ok) {
    if (opts.stream) return res;
    const text = await res.text();
    try { return JSON.parse(text); } catch { return text; }
  }

  let errMsg = `HTTP ${res.status}`;
  try {
    const body = await res.json();
    if (body.message) errMsg = body.message;
    else if (body.error) errMsg = body.error;
  } catch {}
  throw new APIError(res.status, res.status, errMsg);
}

async function download(cfg, filePath, writable, range) {
  // Get presigned URL then stream directly from R2
  const data = await request(cfg, "GET", `/presign/read/${filePath}`);
  if (!data.url) throw new CLIError("presigned URL not available", "Server did not return a presigned URL", EXIT_ERROR);

  const fetchOpts = {};
  if (range) {
    const rangeHeader = range.startsWith("bytes=") ? range : `bytes=${range}`;
    fetchOpts.headers = { Range: rangeHeader };
  }

  let res;
  try {
    res = await fetch(data.url, fetchOpts);
  } catch {
    throw new CLIError("download failed", "Could not reach R2 storage endpoint", EXIT_NETWORK);
  }
  if (!res.ok && res.status !== 206) throw new APIError(res.status, res.status, `R2 download failed: HTTP ${res.status}`);

  const reader = res.body.getReader();
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;
      writable.write(value);
    }
  } finally {
    reader.releaseLock();
  }
  if (writable !== process.stdout) {
    writable.end();
    await new Promise((resolve) => writable.on("finish", resolve));
  }
}

// ── Auth helpers ───────────────────────────────────────────────────────

function requireToken(cfg) {
  if (!cfg.token) {
    throw new CLIError(
      "not authenticated",
      `No token found in --token flag, $STORAGE_TOKEN, or ${tokenFile()}\nRun 'storage login' to authenticate`,
      EXIT_AUTH,
    );
  }
}

// ── MIME detection ─────────────────────────────────────────────────────

const MIME_MAP = {
  ".json": "application/json", ".md": "text/markdown", ".html": "text/html",
  ".htm": "text/html", ".css": "text/css", ".js": "application/javascript",
  ".mjs": "application/javascript", ".xml": "application/xml", ".csv": "text/csv",
  ".yaml": "application/yaml", ".yml": "application/yaml", ".pdf": "application/pdf",
  ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
  ".gif": "image/gif", ".svg": "image/svg+xml", ".webp": "image/webp",
  ".mp4": "video/mp4", ".webm": "video/webm", ".mp3": "audio/mpeg",
  ".wav": "audio/wav", ".zip": "application/zip", ".gz": "application/gzip",
  ".tar": "application/x-tar", ".txt": "text/plain", ".go": "text/x-go",
  ".ts": "text/typescript", ".tsx": "text/typescript", ".rs": "text/x-rust",
  ".py": "text/x-python", ".rb": "text/x-ruby", ".sh": "text/x-shellscript",
};

function detectMime(filepath) {
  return MIME_MAP[extname(filepath).toLowerCase()] || "application/octet-stream";
}

// ── Duration parsing ───────────────────────────────────────────────────

function parseDuration(s) {
  if (!s) return 3600;
  const m = s.match(/^(\d+)([smhd]?)$/);
  if (!m) throw new CLIError(`invalid duration: ${s}`, "Use a number with optional unit: 30s, 15m, 2h, 7d, or plain seconds", EXIT_USAGE);
  const n = parseInt(m[1], 10);
  const u = m[2] || "s";
  return n * ({ s: 1, m: 60, h: 3600, d: 86400 }[u] || 1);
}

// ── Commands ───────────────────────────────────────────────────────────

// ─── login ─────────────────────────────────────────────────────────────

async function cmdLogin(cfg) {
  const codeVerifier = randomBytes(32).toString("base64url");
  const codeChallenge = createHash("sha256").update(codeVerifier).digest("base64url");
  const state = randomBytes(16).toString("hex");

  const { port, codePromise, server } = await startCallbackServer(state);
  const redirectUri = `http://localhost:${port}/callback`;

  const params = new URLSearchParams({
    response_type: "code",
    client_id: OAUTH_CLIENT_ID,
    redirect_uri: redirectUri,
    code_challenge: codeChallenge,
    code_challenge_method: "S256",
    scope: OAUTH_SCOPE,
    state,
  });
  const authUrl = `${cfg.endpoint}/oauth/authorize?${params}`;

  info("Opening", "browser for authentication...");
  const opened = openBrowser(authUrl);
  if (!opened) {
    process.stderr.write(`\n  Open this URL in your browser:\n\n  ${authUrl}\n\n`);
  }
  process.stderr.write(`  ${fmt.dim("Waiting for authentication...")}\n`);

  let code;
  try {
    code = await Promise.race([
      codePromise,
      new Promise((_, rej) => setTimeout(() => rej(new Error("timeout")), 300000)),
    ]);
  } catch {
    server.close();
    throw new CLIError("authentication timed out", "Try again: storage login");
  }
  server.close();

  let tokenRes;
  try {
    tokenRes = await fetch(`${cfg.endpoint}/oauth/token`, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        grant_type: "authorization_code",
        code,
        redirect_uri: redirectUri,
        client_id: OAUTH_CLIENT_ID,
        code_verifier: codeVerifier,
      }),
      signal: AbortSignal.timeout(30000),
    });
  } catch {
    throw new CLIError("token exchange failed", `Could not reach ${cfg.endpoint}\nCheck your connection and try again`, EXIT_NETWORK);
  }
  const tokenData = await tokenRes.json();
  if (!tokenData.access_token) {
    throw new CLIError("failed to get access token", tokenData.error_description || "Token exchange failed", EXIT_AUTH);
  }

  saveToken(tokenData.access_token);
  const days = tokenData.expires_in ? Math.floor(tokenData.expires_in / 86400) : 0;
  const expiresMsg = days > 0 ? ` (expires in ${days} days)` : "";
  info("Authenticated", `token saved to ${tokenFile()}${expiresMsg}`);
}

function startCallbackServer(expectedState) {
  return new Promise((resolve) => {
    let resolveCode, rejectCode;
    const codePromise = new Promise((res, rej) => { resolveCode = res; rejectCode = rej; });

    const server = createServer((req, res) => {
      const url = new URL(req.url, `http://localhost`);
      if (url.pathname !== "/callback") { res.writeHead(404); res.end(); return; }

      const q = url.searchParams;
      if (q.get("state") !== expectedState) {
        res.writeHead(400, { "Content-Type": "text/html" });
        res.end(errorHTML("State mismatch", "This request may have been tampered with."));
        rejectCode(new Error("state mismatch"));
        return;
      }
      if (q.get("error")) {
        const desc = q.get("error_description") || q.get("error");
        res.writeHead(400, { "Content-Type": "text/html" });
        res.end(errorHTML("Authorization failed", desc));
        rejectCode(new Error(desc));
        return;
      }
      const code = q.get("code");
      if (!code) {
        res.writeHead(400, { "Content-Type": "text/html" });
        res.end(errorHTML("Missing code", "No authorization code received."));
        rejectCode(new Error("no code"));
        return;
      }

      res.writeHead(200, { "Content-Type": "text/html" });
      res.end(successHTML());
      resolveCode(code);
    });

    server.listen(0, "127.0.0.1", () => {
      resolve({ port: server.address().port, codePromise, server });
    });
  });
}

function openBrowser(url) {
  const platform = process.platform;
  const cmd = platform === "darwin" ? "open" : platform === "win32" ? "cmd" : "xdg-open";
  const args = platform === "win32" ? ["/c", "start", "", url] : [url];
  try { execFile(cmd, args, { stdio: "ignore" }); return true; } catch { return false; }
}

function successHTML() {
  return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Authenticated — storage.now</title>
<style>*{margin:0;padding:0;box-sizing:border-box}body{font-family:system-ui,sans-serif;background:#fafafa;color:#111;display:flex;align-items:center;justify-content:center;min-height:100vh}@media(prefers-color-scheme:dark){body{background:#111;color:#eee}.card{background:#1a1a1a;border-color:#333}.sub{color:#999}}.card{background:#fff;border:1px solid #ddd;padding:2.5rem;max-width:420px;text-align:center}h1{font-size:1.1rem;font-weight:600;margin-bottom:0.5rem}.sub{font-size:0.85rem;color:#666;line-height:1.5}.check{font-size:2rem;margin-bottom:1rem}.brand{font-size:0.75rem;text-transform:uppercase;letter-spacing:0.08em;color:#999;margin-bottom:1rem}</style></head><body>
<div class="card"><div class="brand">storage.now</div><div class="check">&#10003;</div><h1>Authentication successful</h1><p class="sub">You can close this tab and return to your terminal.</p></div>
<script>setTimeout(()=>window.close(),2000)</script></body></html>`;
}

function escHtml(s) {
  return String(s).replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/"/g, "&quot;");
}

function errorHTML(title, message) {
  const t = escHtml(title), m = escHtml(message);
  return `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>${t} — storage.now</title>
<style>*{margin:0;padding:0;box-sizing:border-box}body{font-family:system-ui,sans-serif;background:#fafafa;color:#111;display:flex;align-items:center;justify-content:center;min-height:100vh}@media(prefers-color-scheme:dark){body{background:#111;color:#eee}.card{background:#1a1a1a;border-color:#333}.sub{color:#999}}.card{background:#fff;border:1px solid #ddd;padding:2.5rem;max-width:420px;text-align:center}h1{font-size:1.1rem;font-weight:600;margin-bottom:0.5rem}.sub{font-size:0.85rem;color:#666;line-height:1.5}.brand{font-size:0.75rem;text-transform:uppercase;letter-spacing:0.08em;color:#999;margin-bottom:1rem}</style></head><body>
<div class="card"><div class="brand">storage.now</div><h1>${t}</h1><p class="sub">${m}</p></div></body></html>`;
}

// ─── logout / token ────────────────────────────────────────────────────

async function cmdLogout() {
  if (removeToken()) info("Logged out", `token removed from ${tokenFile()}`);
  else info("Already", "logged out");
}

async function cmdToken(cfg, args) {
  if (args.length === 0) {
    if (!cfg.token) throw new CLIError("no token configured", "Run 'storage login' to authenticate\nOr set directly: storage token <token>", EXIT_AUTH);
    let source = "unknown";
    if (process.env.STORAGE_TOKEN) source = "$STORAGE_TOKEN";
    else { try { statSync(tokenFile()); source = tokenFile(); } catch {} }
    const truncated = cfg.token.length > 12 ? cfg.token.slice(0, 12) + "..." : cfg.token;
    console.log(`${fmt.dim("source:")} ${source}`);
    console.log(`${fmt.dim("token:")}  ${truncated}`);
    return;
  }
  saveToken(args[0]);
  info("Saved", `token stored in ${tokenFile()}`);
}

// ─── ls ────────────────────────────────────────────────────────────────

async function cmdLs(cfg, args, flags) {
  requireToken(cfg);
  const prefix = args[0] || "";
  const params = new URLSearchParams();
  if (flags.limit) params.set("limit", flags.limit);
  if (flags.offset) params.set("offset", flags.offset);
  const qs = params.toString();

  const data = await request(cfg, "GET", `/ls/${prefix}${qs ? "?" + qs : ""}`);
  if (jsonOutput) return printJSON(data);

  const entries = data.entries || [];
  if (entries.length === 0) {
    info("Empty", prefix ? `no files in ${prefix}` : "no files yet — upload with: storage put <file>");
    return;
  }

  console.log(`${"NAME".padEnd(36)} ${"SIZE".padStart(10)}  ${"TYPE".padEnd(24)}  MODIFIED`);
  for (const e of entries) {
    const size = e.type === "directory" ? "-" : humanSize(e.size || 0);
    const type = e.type === "directory" ? fmt.cyan("directory/") : (e.type || "");
    const mod = e.updated_at ? relativeTime(e.updated_at) : "-";
    console.log(`${(e.name || "").padEnd(36)} ${size.padStart(10)}  ${type.padEnd(24)}  ${mod}`);
  }
  if (data.truncated) console.log(fmt.dim(`\n  ... more results. Use --limit and --offset to paginate.`));
}

// ─── multipart upload ───────────────────────────────────────────────────

async function multipartUpload(cfg, destPath, body, contentType) {
  const PART_SIZE = 50 * 1024 * 1024; // 50 MB
  const partCount = Math.ceil(body.length / PART_SIZE);

  // 1. Initiate multipart upload
  const init = await request(cfg, "POST", "/presign/multipart/create", {
    body: { path: destPath, content_type: contentType, part_count: partCount },
  });
  if (!init.upload_id) throw new CLIError("multipart init failed", "Server did not return an upload ID", EXIT_ERROR);

  info("Multipart", `${partCount} parts, ${humanSize(PART_SIZE)} each`);

  // 2. Upload parts (4 concurrent)
  const CONCURRENCY = 4;
  const parts = [];
  let uploaded = 0;

  for (let batch = 0; batch < partCount; batch += CONCURRENCY) {
    const batchEnd = Math.min(batch + CONCURRENCY, partCount);
    const promises = [];
    for (let i = batch; i < batchEnd; i++) {
      const start = i * PART_SIZE;
      const end = Math.min(start + PART_SIZE, body.length);
      const chunk = body.slice(start, end);
      const url = init.part_urls[i];

      promises.push(
        fetch(url, { method: "PUT", body: chunk }).then((res) => {
          if (!res.ok) throw new APIError(res.status, res.status, `Part ${i + 1} upload failed: HTTP ${res.status}`);
          uploaded++;
          if (!quiet) process.stderr.write(`  ${fmt.dim(`part ${uploaded}/${partCount}`)}\r`);
          return { part_number: i + 1, etag: res.headers.get("ETag")?.replace(/"/g, "") || "" };
        }),
      );
    }
    const batchParts = await Promise.all(promises);
    parts.push(...batchParts);
  }

  if (!quiet) process.stderr.write("\n");

  // 3. Complete multipart upload
  parts.sort((a, b) => a.part_number - b.part_number);
  const data = await request(cfg, "POST", "/presign/multipart/complete", {
    body: { path: destPath, upload_id: init.upload_id, parts },
  });

  return data;
}

// ─── put ───────────────────────────────────────────────────────────────

async function cmdPut(cfg, args, flags) {
  requireToken(cfg);
  if (args.length < 1) throw new CLIError("file required", "Usage: storage put <file> [path]\n  storage put report.pdf docs/report.pdf\n  echo data | storage put - notes/data.txt", EXIT_USAGE);

  const file = args[0];
  let destPath = args[1] || "";

  // If no dest path given, use the filename
  if (!destPath) {
    if (file === "-") throw new CLIError("path required", "When reading from stdin, specify a destination path\nUsage: echo data | storage put - path/to/file.txt", EXIT_USAGE);
    destPath = basename(file);
  }

  // If dest looks like a directory (ends with /), append the filename
  if (destPath.endsWith("/") && file !== "-") {
    destPath += basename(file);
  }

  // Strip leading slash if present
  if (destPath.startsWith("/")) destPath = destPath.slice(1);

  const ct = flags.type || (file !== "-" ? detectMime(file) : "application/octet-stream");

  let body;
  if (file === "-") {
    const chunks = [];
    for await (const chunk of process.stdin) chunks.push(chunk);
    body = Buffer.concat(chunks);
  } else {
    try { body = readFileSync(file); } catch (err) {
      if (err.code === "ENOENT") throw new CLIError("file not found", `${file} does not exist`, EXIT_NOT_FOUND);
      throw err;
    }
  }

  const MULTIPART_THRESHOLD = 100 * 1024 * 1024; // 100 MB
  const PART_SIZE = 50 * 1024 * 1024; // 50 MB

  if (body.length > MULTIPART_THRESHOLD) {
    // Multipart upload for large files
    const data = await multipartUpload(cfg, destPath, body, ct);
    if (jsonOutput) return printJSON(data);
    info("Uploaded", `${destPath} (${humanSize(body.length)}, multipart)`);
    return;
  }

  // 1. Get presigned URL
  const presign = await request(cfg, "POST", "/presign/upload", {
    body: { path: destPath, content_type: ct },
  });
  if (!presign.url) throw new CLIError("presigned URL not available", "Server did not return a presigned URL", EXIT_ERROR);

  // 2. Upload directly to R2
  let r2Res;
  try {
    r2Res = await fetch(presign.url, {
      method: "PUT",
      headers: { "Content-Type": presign.content_type || ct },
      body,
    });
  } catch {
    throw new CLIError("upload failed", "Could not reach R2 storage endpoint", EXIT_NETWORK);
  }
  if (!r2Res.ok) throw new APIError(r2Res.status, r2Res.status, `R2 upload failed: HTTP ${r2Res.status}`);

  // 3. Confirm upload (updates DB index)
  const data = await request(cfg, "POST", "/presign/complete", {
    body: { path: destPath },
  });

  if (jsonOutput) return printJSON(data);
  const sizeStr = humanSize(body.length);
  info("Uploaded", `${destPath} (${sizeStr})`);
}

// ─── get ───────────────────────────────────────────────────────────────

async function cmdGet(cfg, args, flags) {
  requireToken(cfg);
  if (args.length < 1) throw new CLIError("path required", "Usage: storage get <path> [local-path]\n  storage get docs/report.pdf\n  storage get docs/report.pdf ~/Downloads/", EXIT_USAGE);

  let filePath = args[0];
  if (filePath.startsWith("/")) filePath = filePath.slice(1);

  let dest = args[1] || basename(filePath);

  if (dest === "-") {
    await download(cfg, filePath, process.stdout, flags.range);
    return;
  }

  // If dest is a directory, append the filename
  try {
    if (statSync(dest).isDirectory()) dest = join(dest, basename(filePath));
  } catch {}

  const dir = dirname(dest);
  if (dir !== ".") mkdirSync(dir, { recursive: true });
  const ws = createWriteStream(dest);
  await download(cfg, filePath, ws, flags.range);
  if (!quiet) {
    try {
      const st = statSync(dest);
      info("Downloaded", `${basename(filePath)} (${humanSize(st.size)})`);
    } catch {
      info("Downloaded", basename(filePath));
    }
  }
}

// ─── cat ───────────────────────────────────────────────────────────────

async function cmdCat(cfg, args, flags) {
  requireToken(cfg);
  if (args.length < 1) throw new CLIError("path required", "Usage: storage cat <path>", EXIT_USAGE);
  let filePath = args[0];
  if (filePath.startsWith("/")) filePath = filePath.slice(1);
  await download(cfg, filePath, process.stdout, flags.range);
}

// ─── rm ────────────────────────────────────────────────────────────────

async function cmdRm(cfg, args, flags) {
  requireToken(cfg);
  if (args.length < 1) throw new CLIError("path required", "Usage: storage rm <path...>\n  storage rm docs/old.pdf\n  storage rm logs/ --recursive", EXIT_USAGE);

  for (let path of args) {
    if (path.startsWith("/")) path = path.slice(1);

    // For recursive delete, ensure trailing slash
    if (flags.recursive && !path.endsWith("/")) path += "/";

    if (path.endsWith("/") && isTTY && !quiet && !flags.force) {
      // Confirm recursive delete
      process.stderr.write(`Delete everything under ${fmt.bold(path)}? [y/N] `);
      const answer = await readLine();
      if (!answer.toLowerCase().startsWith("y")) {
        info("Skipped", path);
        continue;
      }
    }

    const data = await request(cfg, "DELETE", `/f/${path}`);
    if (jsonOutput) { printJSON(data); continue; }

    if (path.endsWith("/")) {
      info("Deleted", `${path} (${data.deleted || 0} files)`);
    } else {
      info("Deleted", path);
    }
  }
}

// ─── mv ────────────────────────────────────────────────────────────────

async function cmdMv(cfg, args) {
  requireToken(cfg);
  if (args.length !== 2) throw new CLIError("usage: storage mv <from> <to>", "  storage mv drafts/post.md published/post.md", EXIT_USAGE);
  let from = args[0], to = args[1];
  if (from.startsWith("/")) from = from.slice(1);
  if (to.startsWith("/")) to = to.slice(1);

  const data = await request(cfg, "POST", "/mv", { body: { from, to } });
  if (jsonOutput) return printJSON(data);
  info("Moved", `${from} → ${to}`);
}

// ─── share ─────────────────────────────────────────────────────────────

async function cmdShare(cfg, args, flags) {
  requireToken(cfg);
  if (args.length < 1) throw new CLIError("path required", "Usage: storage share <path> [--expires 7d]", EXIT_USAGE);
  let filePath = args[0];
  if (filePath.startsWith("/")) filePath = filePath.slice(1);

  const ttl = parseDuration(flags.expires || "1h");
  const data = await request(cfg, "POST", "/share", { body: { path: filePath, ttl } });
  if (jsonOutput) return printJSON(data);
  console.log(data.url);
  info("Expires", `in ${flags.expires || "1h"}`);
}

// ─── find ──────────────────────────────────────────────────────────────

async function cmdFind(cfg, args, flags) {
  requireToken(cfg);
  const query = args[0] || "";
  if (!query) throw new CLIError("query required", "Usage: storage find <query>\n  storage find report\n  storage find \"*.pdf\"", EXIT_USAGE);

  const params = new URLSearchParams({ q: query });
  if (flags.limit) params.set("limit", flags.limit);

  const data = await request(cfg, "GET", `/find?${params}`);
  if (jsonOutput) return printJSON(data);

  const results = data.results || [];
  if (results.length === 0) { info("No results", `for "${query}"`); return; }
  console.log(`${"PATH".padEnd(40)} NAME`);
  for (const r of results) {
    console.log(`${(r.path || "").padEnd(40)} ${r.name || ""}`);
  }
}

// ─── stat ──────────────────────────────────────────────────────────────

async function cmdStat(cfg) {
  requireToken(cfg);
  const data = await request(cfg, "GET", "/stat");
  if (jsonOutput) return printJSON(data);

  const w = (label, value) => console.log(`${fmt.dim(label.padEnd(14))} ${value}`);
  w("Files:", String(data.files || 0));
  w("Total size:", humanSize(data.bytes || 0));
}

// ─── key management ────────────────────────────────────────────────────

async function cmdKeyCreate(cfg, args, flags) {
  requireToken(cfg);
  if (args.length < 1) throw new CLIError("name required", "Usage: storage key create <name> [--prefix <path>]", EXIT_USAGE);

  const body = { name: args[0] };
  if (flags.prefix) body.prefix = flags.prefix;
  if (flags.expires) body.expires_in = parseDuration(flags.expires);

  const data = await request(cfg, "POST", "/auth/keys", { body });
  if (jsonOutput) return printJSON(data);

  console.log();
  console.log(`${fmt.bold("API Key:")} ${data.token}`);
  console.log();
  console.log(fmt.dim("Save this key — it won't be shown again."));
  console.log(`Use it with: ${fmt.cyan(`STORAGE_TOKEN=${data.token} storage ls`)}`);
}

async function cmdKeyList(cfg) {
  requireToken(cfg);
  const data = await request(cfg, "GET", "/auth/keys");
  if (jsonOutput) return printJSON(data);

  const keys = data.keys || [];
  if (keys.length === 0) { info("No API keys", "Create one with: storage key create <name>"); return; }

  console.log(`${"ID".padEnd(28)} ${"NAME".padEnd(20)} ${"PREFIX".padEnd(16)}  CREATED`);
  for (const k of keys) {
    console.log(`${(k.id || "").padEnd(28)} ${(k.name || "").padEnd(20)} ${(k.prefix || "*").padEnd(16)}  ${relativeTime(k.created_at)}`);
  }
}

async function cmdKeyRevoke(cfg, args) {
  requireToken(cfg);
  if (args.length < 1) throw new CLIError("id required", "Usage: storage key rm <id>", EXIT_USAGE);
  await request(cfg, "DELETE", `/auth/keys/${args[0]}`);
  info("Revoked", `API key ${args[0]}`);
}

// ── Readline helper ────────────────────────────────────────────────────

function readLine() {
  return new Promise((resolve) => {
    let buf = "";
    const onData = (chunk) => {
      buf += chunk;
      if (buf.includes("\n")) {
        cleanup();
        resolve(buf.split("\n")[0].trim());
      }
    };
    const onEnd = () => { cleanup(); resolve(buf.trim()); };
    const cleanup = () => {
      process.stdin.removeListener("data", onData);
      process.stdin.removeListener("end", onEnd);
      process.stdin.pause();
    };
    process.stdin.setEncoding("utf8");
    process.stdin.resume();
    process.stdin.on("data", onData);
    process.stdin.on("end", onEnd);
  });
}

// ── Arg parser ─────────────────────────────────────────────────────────

function parseArgs(argv) {
  const flags = {};
  const args = [];
  let i = 0;
  while (i < argv.length) {
    const a = argv[i];
    if (a === "--") { args.push(...argv.slice(i + 1)); break; }
    if (a.startsWith("--")) {
      const eq = a.indexOf("=");
      if (eq > 0) {
        flags[a.slice(2, eq)] = a.slice(eq + 1);
      } else if (["--json", "--quiet", "--no-color", "--help", "--version", "--force", "--recursive"].includes(a)) {
        flags[a.slice(2)] = true;
      } else {
        flags[a.slice(2)] = argv[++i] || "";
      }
    } else if (a.startsWith("-") && a.length === 2) {
      const map = { j: "json", q: "quiet", v: "version", h: "help", t: "token", e: "endpoint",
        l: "limit", x: "expires", f: "force", r: "recursive", p: "prefix", T: "type" };
      const key = map[a[1]];
      if (key) {
        if (["json", "quiet", "version", "help", "force", "recursive"].includes(key)) flags[key] = true;
        else flags[key] = argv[++i] || "";
      }
    } else {
      args.push(a);
    }
    i++;
  }
  return { flags, args };
}

// ── Help ───────────────────────────────────────────────────────────────

const HELP = `${fmt.bold("storage")} — CLI for Storage API
https://storage.liteio.dev

Upload, download, and share files from your terminal.
Zero dependencies, pipe-friendly, works with Node/Bun/Deno.

${fmt.bold("Usage:")}
  storage <command> [options] [args]

${fmt.bold("Commands:")}
  login                      Authenticate via browser (OAuth)
  logout                     Remove saved credentials
  token [<token>]            Show or set authentication token

  ls [path]                  List files and directories
  put <file> [path]          Upload a file (or stdin with -)
  get <path> [dest]          Download a file (or stdout with -)
  cat <path>                 Print file contents to stdout
  rm <path...>               Delete files or directories
  mv <from> <to>             Move or rename a file

  share <path>               Create a public share link
  find <query>               Search files by name
  stat                       Show storage usage

  key create <name>          Create an API key
  key list                   List API keys
  key rm <id>                Delete an API key

${fmt.bold("Global flags:")}
  -j, --json              JSON output
  -q, --quiet             Suppress non-essential output
  -t, --token <token>     Bearer token or API key
  -e, --endpoint <url>    API base URL
      --no-color          Disable colors
  -v, --version           Show version
  -h, --help              Show help
      --range <range>     Byte range for partial downloads (e.g. bytes=0-1023)

${fmt.bold("Examples:")}
  storage login
  storage put report.pdf docs/
  storage ls docs/
  storage get docs/report.pdf
  storage share docs/report.pdf --expires 7d
  echo "hello" | storage put - notes/hello.txt
  storage cat docs/data.json | jq '.items'
  storage find quarterly --json | jq '.results[].path'

${fmt.bold("Install:")}
  npx @liteio/storage-cli            # run once
  npm install -g @liteio/storage-cli  # install globally
`;

// ── Main ───────────────────────────────────────────────────────────────

async function main() {
  const raw = process.argv.slice(2);
  const { flags, args } = parseArgs(raw);

  if (flags["no-color"]) noColor = true;
  if (flags.quiet) quiet = true;
  if (flags.json) jsonOutput = true;

  if (flags.version) { console.log(`storage ${VERSION}`); return; }
  if (flags.help || args.length === 0) { process.stderr.write(HELP); return; }

  const cfg = loadConfig(flags.token, flags.endpoint);
  const cmd = args[0];
  const cmdArgs = args.slice(1);

  try {
    switch (cmd) {
      case "login":       await cmdLogin(cfg); break;
      case "logout":      await cmdLogout(); break;
      case "token":       await cmdToken(cfg, cmdArgs); break;
      case "ls": case "list": await cmdLs(cfg, cmdArgs, flags); break;
      case "put": case "upload": case "push": await cmdPut(cfg, cmdArgs, flags); break;
      case "get": case "download": case "pull": await cmdGet(cfg, cmdArgs, flags); break;
      case "cat":         await cmdCat(cfg, cmdArgs, flags); break;
      case "rm": case "delete": case "del": await cmdRm(cfg, cmdArgs, flags); break;
      case "mv": case "move": case "rename": await cmdMv(cfg, cmdArgs); break;
      case "share":       await cmdShare(cfg, cmdArgs, flags); break;
      case "find": case "search": await cmdFind(cfg, cmdArgs, flags); break;
      case "stat": case "stats": await cmdStat(cfg); break;
      case "key": case "keys":
        if (cmdArgs[0] === "create" || cmdArgs[0] === "new") await cmdKeyCreate(cfg, cmdArgs.slice(1), flags);
        else if (cmdArgs[0] === "list" || cmdArgs[0] === "ls") await cmdKeyList(cfg);
        else if (cmdArgs[0] === "rm" || cmdArgs[0] === "revoke" || cmdArgs[0] === "delete") await cmdKeyRevoke(cfg, cmdArgs.slice(1));
        else die("unknown key subcommand", `Available: create, list, rm`, "", EXIT_USAGE);
        break;
      default:
        die(`unknown command: ${cmd}`, "Run 'storage --help' for usage", "", EXIT_USAGE);
    }
  } catch (err) {
    if (err instanceof CLIError) {
      printError(err.message, err.hint, "");
      process.exit(err.exitCode);
    }
    if (err instanceof APIError) {
      const status = err.status;
      if (status === 401) printError("authentication failed", err.message, "Run 'storage login' to re-authenticate");
      else if (status === 403) printError("permission denied", err.message, "Check your API key prefix restrictions");
      else if (status === 404) printError("not found", err.message, "");
      else if (status === 409) printError("conflict", err.message, "");
      else printError(`request failed (${status})`, err.message, "");
      process.exit(err.exitCode);
    }
    printError(err.message, "", "");
    process.exit(EXIT_ERROR);
  }
}

main();
