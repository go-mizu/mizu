# chat-now CLI + TUI Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `chat-now`, an npm-publishable CLI + TUI that connects to `chat.go-mizu.workers.dev` with Ed25519 auth.

**Architecture:** ESM-only TypeScript project using Commander for CLI, Ink + React for TUI, @noble/ed25519 for crypto, Zustand for state. Connects to existing Cloudflare Worker via REST + polling.

**Tech Stack:** TypeScript, Node.js 18+ / Bun, Ink 4, React 18, Commander 12, @noble/ed25519 2, Zustand 4

**Spec:** `blueprints/now/spec/0474_chat_cli.md`
**Server:** `https://chat.go-mizu.workers.dev`

---

## File Structure

```
tools/chat-cli/
├── package.json              # npm package config
├── tsconfig.json             # TypeScript config
├── bin/
│   └── chat-now.js           # ESM shebang entry point
├── src/
│   ├── cli.tsx               # Commander setup, command dispatch, TUI launch
│   ├── auth/
│   │   ├── config.ts         # Config read/write/import from Go CLI
│   │   └── signer.ts         # Ed25519 signing, canonical request, auth header
│   ├── api/
│   │   ├── client.ts         # ChatClient — all REST endpoints
│   │   ├── types.ts          # Chat, Message, error types
│   │   └── transport.ts      # Transport interface + PollingTransport
│   ├── tui/
│   │   ├── App.tsx           # Root: three-panel layout + keybindings
│   │   ├── RoomList.tsx      # Left panel: room/DM list
│   │   ├── MessageView.tsx   # Center panel: scrollable messages
│   │   ├── MemberList.tsx    # Right panel: actors from messages
│   │   ├── InputBar.tsx      # Bottom: text input
│   │   ├── StatusBar.tsx     # Footer: identity, room, polling status
│   │   └── Prompt.tsx        # Inline prompts (create, join, switch)
│   ├── store/
│   │   └── chat.ts           # Zustand store: rooms, messages, UI state
│   └── utils/
│       ├── format.ts         # Time formatting, actor color hashing
│       └── keys.ts           # Keybinding map
└── test/
    ├── signer.test.ts        # Canonical request + signing tests
    ├── config.test.ts        # Config read/write/import tests
    ├── client.test.ts        # API client tests with mock fetch
    └── store.test.ts         # Zustand store state transition tests
```

---

## Task 1: Project Scaffold

**Files:**
- Create: `tools/chat-cli/package.json`
- Create: `tools/chat-cli/tsconfig.json`
- Create: `tools/chat-cli/bin/chat-now.js`

- [ ] **Step 1: Create package.json**

```json
{
  "name": "chat-now",
  "version": "0.1.0",
  "description": "TUI + CLI for chat.go-mizu.workers.dev",
  "type": "module",
  "bin": {
    "chat-now": "./bin/chat-now.js"
  },
  "exports": {
    ".": "./dist/cli.js"
  },
  "engines": {
    "node": ">=18"
  },
  "files": [
    "dist/",
    "bin/"
  ],
  "scripts": {
    "build": "tsc",
    "dev": "tsx src/cli.tsx",
    "test": "node --test --loader tsx test/*.test.ts",
    "prepublishOnly": "npm run build"
  },
  "dependencies": {
    "@inkjs/ui": "^2.0.0",
    "@noble/ed25519": "^2.2.0",
    "commander": "^13.0.0",
    "ink": "^5.2.0",
    "react": "^18.3.0",
    "zustand": "^5.0.0"
  },
  "devDependencies": {
    "@types/react": "^18.3.0",
    "tsx": "^4.19.0",
    "typescript": "^5.7.0"
  }
}
```

- [ ] **Step 2: Create tsconfig.json**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "jsx": "react-jsx",
    "outDir": "dist",
    "rootDir": "src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "declaration": true
  },
  "include": ["src"],
  "exclude": ["node_modules", "dist", "test"]
}
```

- [ ] **Step 3: Create bin/chat-now.js**

```javascript
#!/usr/bin/env node
import "../dist/cli.js";
```

- [ ] **Step 4: Create minimal src/cli.tsx**

```tsx
import { Command } from "commander";

const program = new Command()
  .name("chat-now")
  .description("TUI + CLI for chat.go-mizu.workers.dev")
  .version("0.1.0");

program.parse();
```

- [ ] **Step 5: Install dependencies**

Run: `cd tools/chat-cli && npm install`
Expected: `node_modules/` created, `package-lock.json` generated

- [ ] **Step 6: Verify build**

Run: `cd tools/chat-cli && npx tsc --noEmit`
Expected: No errors

- [ ] **Step 7: Commit**

```bash
git add -f tools/chat-cli/
git commit -m "feat(chat-cli): project scaffold with deps"
```

---

## Task 2: Auth — Config Management

**Files:**
- Create: `tools/chat-cli/src/auth/config.ts`
- Create: `tools/chat-cli/test/config.test.ts`

- [ ] **Step 1: Write config types and tests**

`test/config.test.ts`:
```typescript
import { describe, it, before, after } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, rmSync, readFileSync, writeFileSync, mkdirSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { loadConfig, saveConfig, importGoConfig, type Config } from "../src/auth/config.ts";

describe("config", () => {
  let dir: string;

  before(() => {
    dir = mkdtempSync(join(tmpdir(), "chat-now-test-"));
  });

  after(() => {
    rmSync(dir, { recursive: true });
  });

  it("returns null when config does not exist", async () => {
    const cfg = await loadConfig(join(dir, "nonexistent.json"));
    assert.equal(cfg, null);
  });

  it("saves and loads config", async () => {
    const path = join(dir, "config.json");
    const config: Config = {
      actor: "u/alice",
      public_key: "dGVzdHB1YmtleQ",
      private_key: "dGVzdHByaXZrZXk",
      fingerprint: "a1b2c3d4e5f67890",
      server: "https://chat.go-mizu.workers.dev",
    };
    await saveConfig(path, config);
    const loaded = await loadConfig(path);
    assert.deepEqual(loaded, config);
  });

  it("imports Go CLI config stripping base64url padding", async () => {
    const goPath = join(dir, "go-config.json");
    writeFileSync(goPath, JSON.stringify({
      actor: "u/bob",
      public_key: "dGVzdA==",
      private_key: "cHJpdg==",
      fingerprint: "deadbeef12345678",
    }));
    const cfg = await importGoConfig(goPath);
    assert.equal(cfg!.actor, "u/bob");
    assert.equal(cfg!.public_key, "dGVzdA");
    assert.equal(cfg!.private_key, "cHJpdg");
    assert.equal(cfg!.server, "https://chat.go-mizu.workers.dev");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd tools/chat-cli && npx tsx --test test/config.test.ts`
Expected: FAIL — module not found

- [ ] **Step 3: Implement config.ts**

`src/auth/config.ts`:
```typescript
import { readFile, writeFile, mkdir } from "node:fs/promises";
import { dirname, join } from "node:path";
import { homedir } from "node:os";

export interface Config {
  actor: string;
  public_key: string;
  private_key: string;
  fingerprint: string;
  server: string;
}

const DEFAULT_SERVER = "https://chat.go-mizu.workers.dev";
const DEFAULT_PATH = join(homedir(), ".config", "chat-now", "config.json");
const GO_CLI_PATH = join(homedir(), ".config", "now", "config.json");

export function defaultConfigPath(): string {
  return DEFAULT_PATH;
}

export function goCliConfigPath(): string {
  return GO_CLI_PATH;
}

export async function loadConfig(path: string): Promise<Config | null> {
  try {
    const raw = await readFile(path, "utf-8");
    return JSON.parse(raw) as Config;
  } catch {
    return null;
  }
}

export async function saveConfig(path: string, config: Config): Promise<void> {
  await mkdir(dirname(path), { recursive: true, mode: 0o700 });
  await writeFile(path, JSON.stringify(config, null, 2) + "\n", { mode: 0o600 });
}

function stripPadding(s: string): string {
  return s.replace(/=+$/, "");
}

export async function importGoConfig(path: string): Promise<Config | null> {
  const raw = await loadConfig(path);
  if (!raw) return null;
  return {
    actor: raw.actor,
    public_key: stripPadding(raw.public_key),
    private_key: stripPadding(raw.private_key),
    fingerprint: raw.fingerprint,
    server: raw.server || DEFAULT_SERVER,
  };
}

export async function resolveConfig(overridePath?: string): Promise<Config | null> {
  if (overridePath) return loadConfig(overridePath);
  const cfg = await loadConfig(DEFAULT_PATH);
  if (cfg) return cfg;
  return importGoConfig(GO_CLI_PATH);
}
```

- [ ] **Step 4: Run tests**

Run: `cd tools/chat-cli && npx tsx --test test/config.test.ts`
Expected: All pass

- [ ] **Step 5: Commit**

```bash
git add -f tools/chat-cli/src/auth/config.ts tools/chat-cli/test/config.test.ts
git commit -m "feat(chat-cli): config load/save/import with Go CLI compat"
```

---

## Task 3: Auth — Ed25519 Request Signing

**Files:**
- Create: `tools/chat-cli/src/auth/signer.ts`
- Create: `tools/chat-cli/test/signer.test.ts`

- [ ] **Step 1: Write signing tests**

`test/signer.test.ts`:
```typescript
import { describe, it } from "node:test";
import assert from "node:assert/strict";
import {
  buildCanonicalRequest,
  buildStringToSign,
  sha256hex,
  base64url,
  base64urlDecode,
  signRequest,
  buildAuthHeader,
  generateKeypair,
} from "../src/auth/signer.ts";

describe("base64url", () => {
  it("encodes without padding", () => {
    const buf = new TextEncoder().encode("test");
    const encoded = base64url(buf);
    assert.ok(!encoded.includes("="));
    assert.ok(!encoded.includes("+"));
    assert.ok(!encoded.includes("/"));
  });

  it("round-trips", () => {
    const original = new Uint8Array([0, 1, 2, 255, 254]);
    const encoded = base64url(original);
    const decoded = base64urlDecode(encoded);
    assert.deepEqual(decoded, original);
  });
});

describe("sha256hex", () => {
  it("hashes empty string correctly", async () => {
    const hash = await sha256hex("");
    assert.equal(hash, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855");
  });

  it("hashes content", async () => {
    const hash = await sha256hex("hello");
    assert.equal(hash, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824");
  });
});

describe("buildCanonicalRequest", () => {
  it("builds GET with no query or body", async () => {
    const cr = await buildCanonicalRequest("GET", "/api/chat", "", "");
    const emptyHash = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855";
    assert.equal(cr, `GET\n/api/chat\n\n${emptyHash}`);
  });

  it("builds POST with body", async () => {
    const body = '{"text":"hello"}';
    const cr = await buildCanonicalRequest("POST", "/api/chat/c_123/messages", "", body);
    const bodyHash = await sha256hex(body);
    assert.equal(cr, `POST\n/api/chat/c_123/messages\n\n${bodyHash}`);
  });

  it("sorts query params", async () => {
    const cr = await buildCanonicalRequest("GET", "/api/chat", "limit=10&before=m_abc", "");
    const emptyHash = await sha256hex("");
    assert.equal(cr, `GET\n/api/chat\nbefore=m_abc&limit=10\n${emptyHash}`);
  });
});

describe("buildStringToSign", () => {
  it("builds correctly", async () => {
    const sts = await buildStringToSign(1710000000, "u/alice", "canonical-hash-hex");
    assert.equal(sts, "CHAT-ED25519\n1710000000\nu/alice\ncanonical-hash-hex");
  });
});

describe("signRequest", () => {
  it("generates valid keypair and signs a request", async () => {
    const { publicKey, privateKey } = await generateKeypair();
    assert.equal(publicKey.length, 32);
    assert.equal(privateKey.length, 64);

    const header = await signRequest({
      actor: "u/test",
      privateKey,
      method: "GET",
      path: "/api/chat",
      query: "",
      body: "",
    });
    assert.ok(header.startsWith("CHAT-ED25519 Credential=u/test, Timestamp="));
    assert.ok(header.includes("Signature="));
  });
});

describe("buildAuthHeader", () => {
  it("formats correctly", () => {
    const header = buildAuthHeader("u/alice", 1710000000, "c2lnbmF0dXJl");
    assert.equal(header, "CHAT-ED25519 Credential=u/alice, Timestamp=1710000000, Signature=c2lnbmF0dXJl");
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd tools/chat-cli && npx tsx --test test/signer.test.ts`
Expected: FAIL — module not found

- [ ] **Step 3: Implement signer.ts**

`src/auth/signer.ts`:
```typescript
import { ed25519 } from "@noble/ed25519";
import { sha512 } from "@noble/hashes/sha512";

// @noble/ed25519 v2 needs sha512
ed25519.etc.sha512Sync = (...m: Uint8Array[]) => {
  const h = sha512.create();
  for (const msg of m) h.update(msg);
  return h.digest();
};

const encoder = new TextEncoder();

export function base64url(buf: Uint8Array): string {
  let s = "";
  for (const b of buf) s += String.fromCharCode(b);
  return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function base64urlDecode(s: string): Uint8Array {
  const padded = s.replace(/-/g, "+").replace(/_/g, "/");
  const bin = atob(padded);
  const bytes = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
  return bytes;
}

export async function sha256hex(data: string | Uint8Array): Promise<string> {
  const input = typeof data === "string" ? encoder.encode(data) : data;
  const hash = await crypto.subtle.digest("SHA-256", input);
  return Array.from(new Uint8Array(hash), (b) => b.toString(16).padStart(2, "0")).join("");
}

export function fingerprint(publicKey: Uint8Array): string {
  // Synchronous SHA-256 not available, use async wrapper
  // For fingerprint we need sync — use noble hashes
  const { sha256 } = require("@noble/hashes/sha256") as { sha256: (data: Uint8Array) => Uint8Array };
  const hash = sha256(publicKey);
  return Array.from(hash.slice(0, 8), (b: number) => b.toString(16).padStart(2, "0")).join("");
}

export async function fingerprintAsync(publicKey: Uint8Array): Promise<string> {
  const hash = await sha256hex(publicKey);
  return hash.slice(0, 16);
}

export async function buildCanonicalRequest(
  method: string,
  path: string,
  query: string,
  body: string,
): Promise<string> {
  const sortedQuery = query
    ? query.split("&").sort().join("&")
    : "";
  const bodyHash = await sha256hex(body);
  return `${method}\n${path}\n${sortedQuery}\n${bodyHash}`;
}

export async function buildStringToSign(
  timestamp: number,
  actor: string,
  canonicalHash: string,
): Promise<string> {
  return `CHAT-ED25519\n${timestamp}\n${actor}\n${canonicalHash}`;
}

export function buildAuthHeader(actor: string, timestamp: number, signatureB64: string): string {
  return `CHAT-ED25519 Credential=${actor}, Timestamp=${timestamp}, Signature=${signatureB64}`;
}

export async function generateKeypair(): Promise<{ publicKey: Uint8Array; privateKey: Uint8Array }> {
  const privateKey = ed25519.utils.randomPrivateKey();
  const publicKey = ed25519.getPublicKey(privateKey);
  // ed25519 private key is 32 bytes seed; we store the 64-byte expanded form (seed + pubkey)
  const full = new Uint8Array(64);
  full.set(privateKey, 0);
  full.set(publicKey, 32);
  return { publicKey, privateKey: full };
}

interface SignRequestOpts {
  actor: string;
  privateKey: Uint8Array;
  method: string;
  path: string;
  query: string;
  body: string;
}

export async function signRequest(opts: SignRequestOpts): Promise<string> {
  const { actor, privateKey, method, path, query, body } = opts;
  const timestamp = Math.floor(Date.now() / 1000);

  const canonical = await buildCanonicalRequest(method, path, query, body);
  const canonicalHash = await sha256hex(canonical);
  const stringToSign = await buildStringToSign(timestamp, actor, canonicalHash);

  // Use first 32 bytes (seed) for signing
  const seed = privateKey.slice(0, 32);
  const signature = ed25519.sign(encoder.encode(stringToSign), seed);
  const sigB64 = base64url(signature);

  return buildAuthHeader(actor, timestamp, sigB64);
}
```

Note: The `fingerprint` sync function uses a `require` call. We'll replace this in step 4 with an async-only approach and use `@noble/hashes` as a dependency.

- [ ] **Step 4: Add @noble/hashes dependency and fix fingerprint**

Update `package.json` to add `@noble/hashes` to dependencies. Then refactor `signer.ts` to remove the sync `require` and use only the async `fingerprintAsync`:

```typescript
// Remove the sync fingerprint function entirely.
// Export fingerprintAsync as fingerprint.
```

- [ ] **Step 5: Run tests**

Run: `cd tools/chat-cli && npx tsx --test test/signer.test.ts`
Expected: All pass

- [ ] **Step 6: Commit**

```bash
git add -f tools/chat-cli/src/auth/signer.ts tools/chat-cli/test/signer.test.ts tools/chat-cli/package.json
git commit -m "feat(chat-cli): Ed25519 request signing with canonical format"
```

---

## Task 4: API Client + Types

**Files:**
- Create: `tools/chat-cli/src/api/types.ts`
- Create: `tools/chat-cli/src/api/client.ts`
- Create: `tools/chat-cli/test/client.test.ts`

- [ ] **Step 1: Create types.ts**

`src/api/types.ts`:
```typescript
export interface Chat {
  id: string;
  kind: string;
  title: string;
  creator: string;
  peer?: string;
  created_at: string;
}

export interface Message {
  id: string;
  chat: string;
  actor: string;
  text: string;
  created_at: string;
}

export interface RegisterResponse {
  actor: string;
  recovery_code: string;
}

export interface ListResponse<T> {
  items: T[];
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public body: string,
  ) {
    super(`HTTP ${status}: ${body}`);
    this.name = "ApiError";
  }
}

export class AuthError extends ApiError {
  constructor(status: number, body: string) {
    super(status, body);
    this.name = "AuthError";
  }
}

export class RateLimitError extends ApiError {
  public retryAfter: number;
  constructor(status: number, body: string, retryAfter: number) {
    super(status, body);
    this.name = "RateLimitError";
    this.retryAfter = retryAfter;
  }
}
```

- [ ] **Step 2: Write client tests**

`test/client.test.ts`:
```typescript
import { describe, it, beforeEach } from "node:test";
import assert from "node:assert/strict";
import { ChatClient } from "../src/api/client.ts";
import type { Config } from "../src/auth/config.ts";

// Mock signer that returns a fixed header
const mockSigner = async () => "CHAT-ED25519 Credential=u/test, Timestamp=1710000000, Signature=dGVzdA";

function mockFetch(response: { status: number; body: unknown; headers?: Record<string, string> }) {
  return async (url: string | URL | Request, init?: RequestInit) => {
    return {
      ok: response.status >= 200 && response.status < 300,
      status: response.status,
      headers: new Headers(response.headers || {}),
      json: async () => response.body,
      text: async () => JSON.stringify(response.body),
    } as Response;
  };
}

const testConfig: Config = {
  actor: "u/test",
  public_key: "dGVzdA",
  private_key: "dGVzdA",
  fingerprint: "1234567890abcdef",
  server: "https://chat.go-mizu.workers.dev",
};

describe("ChatClient", () => {
  it("register sends correct request", async () => {
    let capturedUrl = "";
    let capturedBody = "";
    const client = new ChatClient(testConfig, mockSigner, async (url, init) => {
      capturedUrl = url as string;
      capturedBody = init?.body as string;
      return { ok: true, status: 201, json: async () => ({ actor: "u/test", recovery_code: "abc" }), text: async () => "", headers: new Headers() } as Response;
    });
    const result = await client.register("u/test", new Uint8Array([1, 2, 3]));
    assert.ok(capturedUrl.endsWith("/api/register"));
    assert.equal(result.actor, "u/test");
  });

  it("createChat sends POST /api/chat", async () => {
    let capturedInit: RequestInit | undefined;
    const client = new ChatClient(testConfig, mockSigner, async (url, init) => {
      capturedInit = init;
      return { ok: true, status: 201, json: async () => ({ id: "chat_abc", kind: "room", title: "test", creator: "u/test", created_at: "2026-03-17T00:00:00Z" }), text: async () => "", headers: new Headers() } as Response;
    });
    const chat = await client.createChat({ title: "test" });
    assert.equal(chat.id, "chat_abc");
    assert.equal(capturedInit?.method, "POST");
  });

  it("listChats unwraps items envelope", async () => {
    const client = new ChatClient(testConfig, mockSigner, async () => {
      return { ok: true, status: 200, json: async () => ({ items: [{ id: "chat_1" }, { id: "chat_2" }] }), text: async () => "", headers: new Headers() } as Response;
    });
    const chats = await client.listChats();
    assert.equal(chats.length, 2);
  });

  it("throws AuthError on 401", async () => {
    const client = new ChatClient(testConfig, mockSigner, async () => {
      return { ok: false, status: 401, json: async () => ({ error: "unauthorized" }), text: async () => "unauthorized", headers: new Headers() } as Response;
    });
    await assert.rejects(() => client.listChats(), { name: "AuthError" });
  });

  it("throws RateLimitError on 429", async () => {
    const client = new ChatClient(testConfig, mockSigner, async () => {
      return { ok: false, status: 429, json: async () => ({ error: "rate limited" }), text: async () => "rate limited", headers: new Headers({ "retry-after": "60" }) } as Response;
    });
    await assert.rejects(() => client.listChats(), { name: "RateLimitError" });
  });
});
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd tools/chat-cli && npx tsx --test test/client.test.ts`
Expected: FAIL — module not found

- [ ] **Step 4: Implement client.ts**

`src/api/client.ts`:
```typescript
import type { Config } from "../auth/config.ts";
import { base64url } from "../auth/signer.ts";
import type { Chat, Message, RegisterResponse, ListResponse } from "./types.ts";
import { ApiError, AuthError, RateLimitError } from "./types.ts";

type Signer = (method: string, path: string, query: string, body: string) => Promise<string>;
type Fetcher = typeof globalThis.fetch;

export class ChatClient {
  private server: string;
  private sign: Signer;
  private fetch: Fetcher;

  constructor(
    private config: Config,
    signFn: Signer,
    fetchFn?: Fetcher,
  ) {
    this.server = config.server.replace(/\/$/, "");
    this.sign = signFn;
    this.fetch = fetchFn || globalThis.fetch;
  }

  private async request<T>(method: string, path: string, opts?: { query?: string; body?: string; noAuth?: boolean }): Promise<T> {
    const query = opts?.query || "";
    const body = opts?.body || "";
    const url = `${this.server}${path}${query ? "?" + query : ""}`;

    const headers: Record<string, string> = { "Content-Type": "application/json" };
    if (!opts?.noAuth) {
      headers["Authorization"] = await this.sign(method, path, query, body);
    }

    const res = await this.fetch(url, {
      method,
      headers,
      body: body || undefined,
    });

    if (!res.ok) {
      const text = await res.text();
      if (res.status === 401 || res.status === 403) throw new AuthError(res.status, text);
      if (res.status === 429) {
        const retryAfter = parseInt(res.headers.get("retry-after") || "60", 10);
        throw new RateLimitError(res.status, text, retryAfter);
      }
      throw new ApiError(res.status, text);
    }

    if (res.status === 204) return undefined as T;
    return res.json() as Promise<T>;
  }

  async register(actor: string, publicKey: Uint8Array): Promise<RegisterResponse> {
    return this.request<RegisterResponse>("POST", "/api/register", {
      body: JSON.stringify({ actor, public_key: base64url(publicKey) }),
      noAuth: true,
    });
  }

  async createChat(opts: { title?: string; visibility?: string } = {}): Promise<Chat> {
    return this.request<Chat>("POST", "/api/chat", {
      body: JSON.stringify({ kind: "room", ...opts }),
    });
  }

  async getChat(id: string): Promise<Chat> {
    return this.request<Chat>("GET", `/api/chat/${id}`);
  }

  async listChats(opts?: { limit?: number }): Promise<Chat[]> {
    const params = new URLSearchParams();
    if (opts?.limit) params.set("limit", String(opts.limit));
    const res = await this.request<ListResponse<Chat>>("GET", "/api/chat", { query: params.toString() });
    return res.items;
  }

  async joinChat(id: string): Promise<void> {
    await this.request<void>("POST", `/api/chat/${id}/join`);
  }

  async startDm(peer: string): Promise<Chat> {
    return this.request<Chat>("POST", "/api/chat/dm", {
      body: JSON.stringify({ peer }),
    });
  }

  async listDms(opts?: { limit?: number }): Promise<Chat[]> {
    const params = new URLSearchParams();
    if (opts?.limit) params.set("limit", String(opts.limit));
    const res = await this.request<ListResponse<Chat>>("GET", "/api/chat/dm", { query: params.toString() });
    return res.items;
  }

  async sendMessage(chatId: string, text: string): Promise<Message> {
    return this.request<Message>("POST", `/api/chat/${chatId}/messages`, {
      body: JSON.stringify({ text }),
    });
  }

  async listMessages(chatId: string, opts?: { limit?: number; before?: string }): Promise<Message[]> {
    const params = new URLSearchParams();
    if (opts?.limit) params.set("limit", String(opts.limit));
    if (opts?.before) params.set("before", opts.before);
    const res = await this.request<ListResponse<Message>>("GET", `/api/chat/${chatId}/messages`, { query: params.toString() });
    return res.items;
  }
}
```

- [ ] **Step 5: Run tests**

Run: `cd tools/chat-cli && npx tsx --test test/client.test.ts`
Expected: All pass

- [ ] **Step 6: Commit**

```bash
git add -f tools/chat-cli/src/api/ tools/chat-cli/test/client.test.ts
git commit -m "feat(chat-cli): API client with typed errors and DM support"
```

---

## Task 5: Polling Transport

**Files:**
- Create: `tools/chat-cli/src/api/transport.ts`

- [ ] **Step 1: Implement transport.ts**

`src/api/transport.ts`:
```typescript
import type { ChatClient } from "./client.ts";
import type { Message, Chat } from "./types.ts";

export type Unsubscribe = () => void;

export interface Transport {
  subscribeMessages(chatId: string, onMessages: (msgs: Message[]) => void): Unsubscribe;
  subscribeRooms(onRooms: (rooms: Chat[]) => void): Unsubscribe;
}

export class PollingTransport implements Transport {
  private timers = new Map<string, ReturnType<typeof setInterval>>();

  constructor(
    private client: ChatClient,
    private messageInterval = 3000,
    private roomInterval = 30000,
  ) {}

  subscribeMessages(chatId: string, onMessages: (msgs: Message[]) => void): Unsubscribe {
    const key = `msg:${chatId}`;
    this.clearTimer(key);

    const poll = async () => {
      try {
        const msgs = await this.client.listMessages(chatId, { limit: 50 });
        onMessages(msgs);
      } catch {
        // Swallow — TUI handles via status bar
      }
    };

    poll(); // Immediate first fetch
    this.timers.set(key, setInterval(poll, this.messageInterval));

    return () => this.clearTimer(key);
  }

  subscribeRooms(onRooms: (rooms: Chat[]) => void): Unsubscribe {
    const key = "rooms";
    this.clearTimer(key);

    const poll = async () => {
      try {
        const rooms = await this.client.listChats();
        const dms = await this.client.listDms();
        onRooms([...rooms, ...dms]);
      } catch {
        // Swallow
      }
    };

    poll();
    this.timers.set(key, setInterval(poll, this.roomInterval));

    return () => this.clearTimer(key);
  }

  private clearTimer(key: string) {
    const existing = this.timers.get(key);
    if (existing) {
      clearInterval(existing);
      this.timers.delete(key);
    }
  }

  destroy() {
    for (const [key] of this.timers) this.clearTimer(key);
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add -f tools/chat-cli/src/api/transport.ts
git commit -m "feat(chat-cli): polling transport with swappable interface"
```

---

## Task 6: Zustand Store

**Files:**
- Create: `tools/chat-cli/src/store/chat.ts`
- Create: `tools/chat-cli/test/store.test.ts`

- [ ] **Step 1: Write store tests**

`test/store.test.ts`:
```typescript
import { describe, it, beforeEach } from "node:test";
import assert from "node:assert/strict";
import { createChatStore } from "../src/store/chat.ts";

describe("ChatStore", () => {
  it("sets active room", () => {
    const store = createChatStore();
    store.getState().setActiveRoom("chat_1");
    assert.equal(store.getState().activeRoomId, "chat_1");
  });

  it("cycles focus", () => {
    const store = createChatStore();
    assert.equal(store.getState().focusedPanel, "input");
    store.getState().cycleFocus();
    assert.equal(store.getState().focusedPanel, "rooms");
    store.getState().cycleFocus();
    assert.equal(store.getState().focusedPanel, "messages");
    store.getState().cycleFocus();
    assert.equal(store.getState().focusedPanel, "members");
    store.getState().cycleFocus();
    assert.equal(store.getState().focusedPanel, "input");
  });

  it("adds messages deduplicating by ID", () => {
    const store = createChatStore();
    store.getState().setMessages("chat_1", [
      { id: "m_1", chat: "chat_1", actor: "u/alice", text: "hi", created_at: "2026-03-17T00:00:00Z" },
    ]);
    store.getState().setMessages("chat_1", [
      { id: "m_1", chat: "chat_1", actor: "u/alice", text: "hi", created_at: "2026-03-17T00:00:00Z" },
      { id: "m_2", chat: "chat_1", actor: "u/bob", text: "hey", created_at: "2026-03-17T00:01:00Z" },
    ]);
    assert.equal(store.getState().messages["chat_1"].length, 2);
  });

  it("derives members from messages", () => {
    const store = createChatStore();
    store.getState().setMessages("chat_1", [
      { id: "m_1", chat: "chat_1", actor: "u/alice", text: "hi", created_at: "2026-03-17T00:00:00Z" },
      { id: "m_2", chat: "chat_1", actor: "u/bob", text: "hey", created_at: "2026-03-17T00:01:00Z" },
      { id: "m_3", chat: "chat_1", actor: "u/alice", text: "yo", created_at: "2026-03-17T00:02:00Z" },
    ]);
    const members = store.getState().membersFor("chat_1");
    assert.deepEqual(members.sort(), ["u/alice", "u/bob"]);
  });

  it("sets rooms", () => {
    const store = createChatStore();
    store.getState().setRooms([
      { id: "chat_1", kind: "room", title: "general", creator: "u/alice", created_at: "2026-03-17T00:00:00Z" },
    ]);
    assert.equal(store.getState().rooms.length, 1);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd tools/chat-cli && npx tsx --test test/store.test.ts`
Expected: FAIL — module not found

- [ ] **Step 3: Implement store**

`src/store/chat.ts`:
```typescript
import { createStore } from "zustand/vanilla";
import type { Chat, Message } from "../api/types.ts";

export type Panel = "rooms" | "messages" | "members" | "input";
const PANELS: Panel[] = ["input", "rooms", "messages", "members"];

export interface ChatState {
  // Rooms
  rooms: Chat[];
  activeRoomId: string | null;
  setActiveRoom: (id: string) => void;
  setRooms: (rooms: Chat[]) => void;

  // Messages keyed by chat ID
  messages: Record<string, Message[]>;
  setMessages: (chatId: string, msgs: Message[]) => void;

  // Members derived from messages
  membersFor: (chatId: string) => string[];

  // UI
  focusedPanel: Panel;
  cycleFocus: () => void;
  setFocus: (panel: Panel) => void;

  // Status
  connected: boolean;
  error: string | null;
  setConnected: (v: boolean) => void;
  setError: (e: string | null) => void;
}

export function createChatStore() {
  return createStore<ChatState>((set, get) => ({
    rooms: [],
    activeRoomId: null,
    messages: {},
    focusedPanel: "input" as Panel,
    connected: false,
    error: null,

    setActiveRoom: (id) => set({ activeRoomId: id }),

    setRooms: (rooms) => set({ rooms }),

    setMessages: (chatId, msgs) =>
      set((state) => {
        const existing = state.messages[chatId] || [];
        const seen = new Set(existing.map((m) => m.id));
        const merged = [...existing];
        for (const m of msgs) {
          if (!seen.has(m.id)) {
            merged.push(m);
            seen.add(m.id);
          }
        }
        merged.sort((a, b) => a.created_at.localeCompare(b.created_at));
        return { messages: { ...state.messages, [chatId]: merged } };
      }),

    membersFor: (chatId) => {
      const msgs = get().messages[chatId] || [];
      return [...new Set(msgs.map((m) => m.actor))];
    },

    cycleFocus: () =>
      set((state) => {
        const idx = PANELS.indexOf(state.focusedPanel);
        return { focusedPanel: PANELS[(idx + 1) % PANELS.length] };
      }),

    setFocus: (panel) => set({ focusedPanel: panel }),
    setConnected: (v) => set({ connected: v }),
    setError: (e) => set({ error: e }),
  }));
}
```

- [ ] **Step 4: Run tests**

Run: `cd tools/chat-cli && npx tsx --test test/store.test.ts`
Expected: All pass

- [ ] **Step 5: Commit**

```bash
git add -f tools/chat-cli/src/store/chat.ts tools/chat-cli/test/store.test.ts
git commit -m "feat(chat-cli): zustand store with dedup and derived members"
```

---

## Task 7: Utilities — Formatting & Keybindings

**Files:**
- Create: `tools/chat-cli/src/utils/format.ts`
- Create: `tools/chat-cli/src/utils/keys.ts`

- [ ] **Step 1: Create format.ts**

`src/utils/format.ts`:
```typescript
const COLORS = ["cyan", "green", "yellow", "blue", "magenta", "red", "gray", "white"] as const;

export type ActorColor = (typeof COLORS)[number];

export function actorColor(actor: string): ActorColor {
  let hash = 0;
  for (let i = 0; i < actor.length; i++) {
    hash = ((hash << 5) - hash + actor.charCodeAt(i)) | 0;
  }
  return COLORS[Math.abs(hash) % COLORS.length];
}

export function formatTime(iso: string): string {
  const date = new Date(iso);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  const time = date.toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });

  if (diffDays === 0) return time;
  if (diffDays === 1) return `yesterday ${time}`;
  if (diffDays < 7) {
    const day = date.toLocaleDateString(undefined, { weekday: "short" });
    return `${day} ${time}`;
  }
  const short = date.toLocaleDateString(undefined, { month: "short", day: "numeric" });
  return `${short} ${time}`;
}

export function roomIcon(kind: string): string {
  return kind === "direct" ? "@" : "#";
}

export function roomLabel(chat: { kind: string; title: string; peer?: string }): string {
  if (chat.kind === "direct" && chat.peer) return `@${chat.peer.replace(/^[ua]\//, "")}`;
  return `#${chat.title || "untitled"}`;
}
```

- [ ] **Step 2: Create keys.ts**

`src/utils/keys.ts`:
```typescript
export const KEYBINDINGS = {
  quit: { key: "q", ctrl: true },
  cycleFocus: { key: "tab" },
  cycleFocusReverse: { key: "tab", shift: true },
  createRoom: { key: "n", ctrl: true },
  joinRoom: { key: "j", ctrl: true },
  quickSwitch: { key: "k", ctrl: true },
  refresh: { key: "r", ctrl: true },
} as const;
```

- [ ] **Step 3: Commit**

```bash
git add -f tools/chat-cli/src/utils/
git commit -m "feat(chat-cli): formatting utils and keybinding map"
```

---

## Task 8: CLI Commands

**Files:**
- Modify: `tools/chat-cli/src/cli.tsx`

- [ ] **Step 1: Implement all CLI commands**

`src/cli.tsx`:
```tsx
import { Command } from "commander";
import { resolveConfig, saveConfig, defaultConfigPath, goCliConfigPath, importGoConfig } from "./auth/config.ts";
import { generateKeypair, base64url, signRequest, fingerprintAsync } from "./auth/signer.ts";
import { ChatClient } from "./api/client.ts";
import type { Config } from "./auth/config.ts";

const program = new Command()
  .name("chat-now")
  .description("TUI + CLI for chat.go-mizu.workers.dev")
  .version("0.1.0")
  .option("--server <url>", "Server URL", "https://chat.go-mizu.workers.dev")
  .option("--config <path>", "Config file path")
  .option("--pretty", "Pretty-print JSON output");

function output(data: unknown, pretty: boolean) {
  console.log(pretty ? JSON.stringify(data, null, 2) : JSON.stringify(data));
}

async function getConfigOrDie(opts: { config?: string }): Promise<Config> {
  const cfg = await resolveConfig(opts.config);
  if (!cfg) {
    console.error('No identity found. Run "chat-now init" first.');
    process.exit(1);
  }
  return cfg;
}

function makeClient(cfg: Config, serverOverride?: string): ChatClient {
  const config = serverOverride ? { ...cfg, server: serverOverride } : cfg;
  const signer = (method: string, path: string, query: string, body: string) =>
    signRequest({ actor: config.actor, privateKey: new Uint8Array(Buffer.from(config.private_key, "base64url")), method, path, query, body });
  return new ChatClient(config, signer);
}

// init
program
  .command("init")
  .description("Generate keypair or import from Go CLI")
  .option("--actor <name>", "Actor name (u/alice or a/bot1)")
  .option("--import", "Import from Go CLI config")
  .action(async (opts) => {
    const parentOpts = program.opts();
    const configPath = parentOpts.config || defaultConfigPath();

    // Try import
    if (opts.import) {
      const goConfig = await importGoConfig(goCliConfigPath());
      if (!goConfig) {
        console.error("Go CLI config not found at", goCliConfigPath());
        process.exit(1);
      }
      await saveConfig(configPath, goConfig);
      console.log(`Imported identity: ${goConfig.actor}`);
      console.log(`Fingerprint: ${goConfig.fingerprint}`);
      console.log(`Config: ${configPath}`);
      return;
    }

    // Generate new
    const actor = opts.actor;
    if (!actor) {
      console.error("--actor is required (e.g. u/alice)");
      process.exit(1);
    }
    if (!/^[ua]\/[\w.@-]{1,64}$/.test(actor)) {
      console.error("Invalid actor format. Use u/<name> or a/<name> (letters, digits, . @ - _)");
      process.exit(1);
    }

    const { publicKey, privateKey } = await generateKeypair();
    const fp = await fingerprintAsync(publicKey);
    const server = parentOpts.server;

    const config: Config = {
      actor,
      public_key: base64url(publicKey),
      private_key: base64url(privateKey),
      fingerprint: fp,
      server,
    };

    // Register with server
    const client = new ChatClient(config, async () => "", globalThis.fetch);
    try {
      const res = await client.register(actor, publicKey);
      console.log(`Registered: ${res.actor}`);
      console.log(`Recovery code: ${res.recovery_code}`);
      console.log("⚠ Save your recovery code — it cannot be retrieved later.");
    } catch (e: any) {
      console.error(`Registration failed: ${e.message}`);
      process.exit(1);
    }

    await saveConfig(configPath, config);
    console.log(`Fingerprint: ${fp}`);
    console.log(`Config: ${configPath}`);
  });

// whoami
program
  .command("whoami")
  .description("Show current identity")
  .action(async () => {
    const opts = program.opts();
    const cfg = await getConfigOrDie(opts);
    output({ actor: cfg.actor, fingerprint: cfg.fingerprint, server: cfg.server }, !!opts.pretty);
  });

// create
program
  .command("create")
  .description("Create a room")
  .option("--title <title>", "Room title")
  .option("--visibility <v>", "public or private", "public")
  .action(async (opts) => {
    const parentOpts = program.opts();
    const cfg = await getConfigOrDie(parentOpts);
    const client = makeClient(cfg, parentOpts.server);
    const chat = await client.createChat({ title: opts.title, visibility: opts.visibility });
    output(chat, !!parentOpts.pretty);
  });

// dm
program
  .command("dm <peer>")
  .description("Start or resume DM with peer")
  .action(async (peer) => {
    const opts = program.opts();
    const cfg = await getConfigOrDie(opts);
    const client = makeClient(cfg, opts.server);
    const chat = await client.startDm(peer);
    output(chat, !!opts.pretty);
  });

// join
program
  .command("join <id>")
  .description("Join a chat")
  .action(async (id) => {
    const opts = program.opts();
    const cfg = await getConfigOrDie(opts);
    const client = makeClient(cfg, opts.server);
    await client.joinChat(id);
  });

// get
program
  .command("get <id>")
  .description("Get chat details")
  .action(async (id) => {
    const opts = program.opts();
    const cfg = await getConfigOrDie(opts);
    const client = makeClient(cfg, opts.server);
    const chat = await client.getChat(id);
    output(chat, !!opts.pretty);
  });

// list
program
  .command("list")
  .description("List chats")
  .option("--limit <n>", "Limit results", "50")
  .action(async (opts) => {
    const parentOpts = program.opts();
    const cfg = await getConfigOrDie(parentOpts);
    const client = makeClient(cfg, parentOpts.server);
    const chats = await client.listChats({ limit: parseInt(opts.limit) });
    output(chats, !!parentOpts.pretty);
  });

// dms
program
  .command("dms")
  .description("List DM conversations")
  .option("--limit <n>", "Limit results", "50")
  .action(async (opts) => {
    const parentOpts = program.opts();
    const cfg = await getConfigOrDie(parentOpts);
    const client = makeClient(cfg, parentOpts.server);
    const dms = await client.listDms({ limit: parseInt(opts.limit) });
    output(dms, !!parentOpts.pretty);
  });

// send
program
  .command("send <id> <text>")
  .description("Send a message")
  .action(async (id, text) => {
    const opts = program.opts();
    const cfg = await getConfigOrDie(opts);
    const client = makeClient(cfg, opts.server);
    const msg = await client.sendMessage(id, text);
    output(msg, !!opts.pretty);
  });

// messages
program
  .command("messages <id>")
  .description("List messages in a chat")
  .option("--limit <n>", "Limit results", "50")
  .option("--before <id>", "Cursor for pagination")
  .action(async (id, opts) => {
    const parentOpts = program.opts();
    const cfg = await getConfigOrDie(parentOpts);
    const client = makeClient(cfg, parentOpts.server);
    const msgs = await client.listMessages(id, { limit: parseInt(opts.limit), before: opts.before });
    output(msgs, !!parentOpts.pretty);
  });

// Default: launch TUI when no command given
program.action(async () => {
  const opts = program.opts();
  const cfg = await getConfigOrDie(opts);
  // Dynamic import to avoid loading Ink for CLI-only usage
  const { launchTui } = await import("./tui/App.tsx");
  await launchTui(cfg, opts.server);
});

program.parseAsync();
```

- [ ] **Step 2: Verify CLI compiles**

Run: `cd tools/chat-cli && npx tsc --noEmit`
Expected: No errors (TUI import will fail — that's ok, we'll create it next)

- [ ] **Step 3: Commit**

```bash
git add -f tools/chat-cli/src/cli.tsx
git commit -m "feat(chat-cli): full CLI command surface"
```

---

## Task 9: TUI — App Root & Layout

**Files:**
- Create: `tools/chat-cli/src/tui/App.tsx`
- Create: `tools/chat-cli/src/tui/StatusBar.tsx`

- [ ] **Step 1: Create StatusBar.tsx**

`src/tui/StatusBar.tsx`:
```tsx
import React from "react";
import { Box, Text } from "ink";

interface Props {
  actor: string;
  room: string | null;
  memberCount: number;
  connected: boolean;
  error: string | null;
}

export function StatusBar({ actor, room, memberCount, connected, error }: Props) {
  return (
    <Box borderStyle="single" borderTop borderBottom={false} borderLeft={false} borderRight={false} paddingX={1}>
      <Text dimColor>{actor}</Text>
      {room && <Text dimColor> · {room}</Text>}
      <Text dimColor> · {memberCount} members</Text>
      <Text dimColor> · </Text>
      {error ? (
        <Text color="red">{error}</Text>
      ) : connected ? (
        <Text color="green">connected</Text>
      ) : (
        <Text color="yellow">connecting...</Text>
      )}
    </Box>
  );
}
```

- [ ] **Step 2: Create App.tsx with three-panel layout**

`src/tui/App.tsx`:
```tsx
import React, { useEffect, useCallback } from "react";
import { render, Box, Text, useApp, useInput } from "ink";
import type { Config } from "../auth/config.ts";
import { signRequest } from "../auth/signer.ts";
import { base64urlDecode } from "../auth/signer.ts";
import { ChatClient } from "../api/client.ts";
import { PollingTransport } from "../api/transport.ts";
import { createChatStore, type ChatState, type Panel } from "../store/chat.ts";
import { RoomList } from "./RoomList.tsx";
import { MessageView } from "./MessageView.tsx";
import { MemberList } from "./MemberList.tsx";
import { InputBar } from "./InputBar.tsx";
import { StatusBar } from "./StatusBar.tsx";
import { roomLabel } from "../utils/format.ts";

interface AppProps {
  config: Config;
  serverOverride?: string;
}

function App({ config, serverOverride }: AppProps) {
  const { exit } = useApp();
  const storeRef = React.useRef(createChatStore());
  const store = storeRef.current;
  const [state, setState] = React.useState<ChatState>(store.getState());

  React.useEffect(() => {
    return store.subscribe(setState);
  }, [store]);

  // Create client and transport
  const clientRef = React.useRef<ChatClient | null>(null);
  const transportRef = React.useRef<PollingTransport | null>(null);

  useEffect(() => {
    const cfg = serverOverride ? { ...config, server: serverOverride } : config;
    const signer = (method: string, path: string, query: string, body: string) =>
      signRequest({
        actor: cfg.actor,
        privateKey: base64urlDecode(cfg.private_key),
        method,
        path,
        query,
        body,
      });
    const client = new ChatClient(cfg, signer);
    clientRef.current = client;

    const transport = new PollingTransport(client);
    transportRef.current = transport;

    const unsubRooms = transport.subscribeRooms((rooms) => {
      store.getState().setRooms(rooms);
      store.getState().setConnected(true);
      store.getState().setError(null);
      // Auto-select first room if none selected
      if (!store.getState().activeRoomId && rooms.length > 0) {
        store.getState().setActiveRoom(rooms[0].id);
      }
    });

    return () => {
      unsubRooms();
      transport.destroy();
    };
  }, [config, serverOverride]);

  // Subscribe to active room messages
  const unsubMsgRef = React.useRef<(() => void) | null>(null);
  useEffect(() => {
    if (unsubMsgRef.current) unsubMsgRef.current();
    if (!state.activeRoomId || !transportRef.current) return;

    const chatId = state.activeRoomId;
    unsubMsgRef.current = transportRef.current.subscribeMessages(chatId, (msgs) => {
      store.getState().setMessages(chatId, msgs);
    });

    return () => {
      if (unsubMsgRef.current) unsubMsgRef.current();
    };
  }, [state.activeRoomId]);

  // Keybindings
  useInput((input, key) => {
    if (key.ctrl && input === "q") { exit(); return; }
    if (key.ctrl && input === "c") { exit(); return; }
    if (key.tab && key.shift) {
      // Reverse cycle
      const panels: Panel[] = ["input", "rooms", "messages", "members"];
      const idx = panels.indexOf(state.focusedPanel);
      store.getState().setFocus(panels[(idx - 1 + panels.length) % panels.length]);
      return;
    }
    if (key.tab) { store.getState().cycleFocus(); return; }
  });

  const activeRoom = state.rooms.find((r) => r.id === state.activeRoomId);
  const activeMessages = state.activeRoomId ? state.messages[state.activeRoomId] || [] : [];
  const members = state.activeRoomId ? state.membersFor(state.activeRoomId) : [];

  const handleSend = useCallback(async (text: string) => {
    if (!clientRef.current || !state.activeRoomId) return;
    try {
      await clientRef.current.sendMessage(state.activeRoomId, text);
    } catch (e: any) {
      store.getState().setError(e.message);
    }
  }, [state.activeRoomId]);

  const handleSelectRoom = useCallback((id: string) => {
    store.getState().setActiveRoom(id);
    store.getState().setFocus("input");
  }, []);

  return (
    <Box flexDirection="column" width="100%">
      <Box flexGrow={1}>
        <Box width={20} flexShrink={0} borderStyle="single" borderRight>
          <RoomList
            rooms={state.rooms}
            activeId={state.activeRoomId}
            focused={state.focusedPanel === "rooms"}
            onSelect={handleSelectRoom}
          />
        </Box>
        <Box flexGrow={1} flexDirection="column">
          <MessageView
            messages={activeMessages}
            currentActor={config.actor}
            focused={state.focusedPanel === "messages"}
          />
        </Box>
        <Box width={16} flexShrink={0} borderStyle="single" borderLeft>
          <MemberList
            members={members}
            currentActor={config.actor}
            focused={state.focusedPanel === "members"}
          />
        </Box>
      </Box>
      <InputBar
        focused={state.focusedPanel === "input"}
        onSubmit={handleSend}
      />
      <StatusBar
        actor={config.actor}
        room={activeRoom ? roomLabel(activeRoom) : null}
        memberCount={members.length}
        connected={state.connected}
        error={state.error}
      />
    </Box>
  );
}

export async function launchTui(config: Config, serverOverride?: string) {
  const { waitUntilExit } = render(<App config={config} serverOverride={serverOverride} />);
  await waitUntilExit();
}
```

- [ ] **Step 3: Commit**

```bash
git add -f tools/chat-cli/src/tui/App.tsx tools/chat-cli/src/tui/StatusBar.tsx
git commit -m "feat(chat-cli): TUI root layout with three panels"
```

---

## Task 10: TUI — Room List Panel

**Files:**
- Create: `tools/chat-cli/src/tui/RoomList.tsx`

- [ ] **Step 1: Implement RoomList**

`src/tui/RoomList.tsx`:
```tsx
import React, { useState } from "react";
import { Box, Text, useInput } from "ink";
import type { Chat } from "../api/types.ts";
import { roomLabel } from "../utils/format.ts";

interface Props {
  rooms: Chat[];
  activeId: string | null;
  focused: boolean;
  onSelect: (id: string) => void;
}

export function RoomList({ rooms, activeId, focused, onSelect }: Props) {
  const [cursor, setCursor] = useState(0);

  useInput(
    (input, key) => {
      if (!focused) return;
      if (key.upArrow) setCursor((c) => Math.max(0, c - 1));
      if (key.downArrow) setCursor((c) => Math.min(rooms.length - 1, c + 1));
      if (key.return && rooms[cursor]) onSelect(rooms[cursor].id);
    },
  );

  if (rooms.length === 0) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text dimColor>No rooms</Text>
        <Text dimColor>Ctrl+N to create</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" paddingX={0}>
      <Box paddingX={1} marginBottom={1}>
        <Text bold>Rooms</Text>
      </Box>
      {rooms.map((room, i) => {
        const isActive = room.id === activeId;
        const isCursor = i === cursor && focused;
        return (
          <Box key={room.id} paddingX={1}>
            <Text
              bold={isActive}
              inverse={isCursor}
              color={isActive ? "cyan" : undefined}
            >
              {roomLabel(room)}
            </Text>
          </Box>
        );
      })}
    </Box>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add -f tools/chat-cli/src/tui/RoomList.tsx
git commit -m "feat(chat-cli): TUI room list panel"
```

---

## Task 11: TUI — Message View Panel

**Files:**
- Create: `tools/chat-cli/src/tui/MessageView.tsx`

- [ ] **Step 1: Implement MessageView**

`src/tui/MessageView.tsx`:
```tsx
import React, { useState, useEffect } from "react";
import { Box, Text, useInput } from "ink";
import type { Message } from "../api/types.ts";
import { actorColor, formatTime } from "../utils/format.ts";

interface Props {
  messages: Message[];
  currentActor: string;
  focused: boolean;
}

export function MessageView({ messages, currentActor, focused }: Props) {
  const [scrollOffset, setScrollOffset] = useState(0);

  // Auto-scroll to bottom on new messages
  useEffect(() => {
    setScrollOffset(0);
  }, [messages.length]);

  useInput((input, key) => {
    if (!focused) return;
    if (key.upArrow) setScrollOffset((o) => Math.min(messages.length - 1, o + 1));
    if (key.downArrow) setScrollOffset((o) => Math.max(0, o - 1));
    if (key.pageUp) setScrollOffset((o) => Math.min(messages.length - 1, o + 10));
    if (key.pageDown) setScrollOffset((o) => Math.max(0, o - 10));
  });

  if (messages.length === 0) {
    return (
      <Box flexDirection="column" justifyContent="center" alignItems="center" flexGrow={1}>
        <Text dimColor>No messages yet</Text>
      </Box>
    );
  }

  // Show messages from bottom, applying scroll offset
  const visibleMessages = messages.slice(
    Math.max(0, messages.length - 30 - scrollOffset),
    messages.length - scrollOffset,
  );

  return (
    <Box flexDirection="column" paddingX={1} flexGrow={1}>
      {visibleMessages.map((msg) => {
        const isMe = msg.actor === currentActor;
        const color = actorColor(msg.actor);
        return (
          <Box key={msg.id} flexDirection="column" marginBottom={0}>
            <Box gap={1}>
              <Text color={color} bold={isMe}>
                {msg.actor}
              </Text>
              <Text dimColor>{formatTime(msg.created_at)}</Text>
            </Box>
            <Box paddingLeft={2}>
              <Text>{msg.text}</Text>
            </Box>
          </Box>
        );
      })}
      {scrollOffset > 0 && (
        <Text dimColor>↓ {scrollOffset} more below</Text>
      )}
    </Box>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add -f tools/chat-cli/src/tui/MessageView.tsx
git commit -m "feat(chat-cli): TUI message view with scrolling"
```

---

## Task 12: TUI — Member List + Input Bar

**Files:**
- Create: `tools/chat-cli/src/tui/MemberList.tsx`
- Create: `tools/chat-cli/src/tui/InputBar.tsx`

- [ ] **Step 1: Create MemberList.tsx**

`src/tui/MemberList.tsx`:
```tsx
import React from "react";
import { Box, Text } from "ink";
import { actorColor } from "../utils/format.ts";

interface Props {
  members: string[];
  currentActor: string;
  focused: boolean;
}

export function MemberList({ members, currentActor, focused }: Props) {
  return (
    <Box flexDirection="column" paddingX={0}>
      <Box paddingX={1} marginBottom={1}>
        <Text bold>Members</Text>
      </Box>
      {members.length === 0 ? (
        <Box paddingX={1}>
          <Text dimColor>—</Text>
        </Box>
      ) : (
        members.map((actor) => (
          <Box key={actor} paddingX={1}>
            <Text
              color={actorColor(actor)}
              bold={actor === currentActor}
            >
              {actor}
            </Text>
          </Box>
        ))
      )}
    </Box>
  );
}
```

- [ ] **Step 2: Create InputBar.tsx**

`src/tui/InputBar.tsx`:
```tsx
import React, { useState } from "react";
import { Box, Text, useInput } from "ink";

interface Props {
  focused: boolean;
  onSubmit: (text: string) => void;
}

export function InputBar({ focused, onSubmit }: Props) {
  const [value, setValue] = useState("");

  useInput(
    (input, key) => {
      if (!focused) return;

      if (key.return && value.trim()) {
        onSubmit(value.trim());
        setValue("");
        return;
      }

      if (key.backspace || key.delete) {
        setValue((v) => v.slice(0, -1));
        return;
      }

      // Ignore control characters
      if (key.ctrl || key.meta) return;

      if (input && !key.upArrow && !key.downArrow && !key.leftArrow && !key.rightArrow && !key.tab) {
        setValue((v) => v + input);
      }
    },
  );

  return (
    <Box borderStyle="single" borderTop borderBottom={false} borderLeft={false} borderRight={false} paddingX={1}>
      <Text color={focused ? "green" : "gray"}>{"> "}</Text>
      <Text>{value}</Text>
      {focused && <Text color="green">▎</Text>}
    </Box>
  );
}
```

- [ ] **Step 3: Commit**

```bash
git add -f tools/chat-cli/src/tui/MemberList.tsx tools/chat-cli/src/tui/InputBar.tsx
git commit -m "feat(chat-cli): TUI member list and input bar"
```

---

## Task 13: TUI — Inline Prompts

**Files:**
- Create: `tools/chat-cli/src/tui/Prompt.tsx`

- [ ] **Step 1: Create Prompt.tsx**

`src/tui/Prompt.tsx`:
```tsx
import React, { useState } from "react";
import { Box, Text, useInput } from "ink";

interface Props {
  label: string;
  onSubmit: (value: string) => void;
  onCancel: () => void;
}

export function Prompt({ label, onSubmit, onCancel }: Props) {
  const [value, setValue] = useState("");

  useInput((input, key) => {
    if (key.escape) { onCancel(); return; }
    if (key.return) { onSubmit(value); return; }
    if (key.backspace || key.delete) { setValue((v) => v.slice(0, -1)); return; }
    if (key.ctrl || key.meta) return;
    if (input && !key.upArrow && !key.downArrow && !key.leftArrow && !key.rightArrow && !key.tab) {
      setValue((v) => v + input);
    }
  });

  return (
    <Box borderStyle="round" paddingX={1} borderColor="yellow">
      <Text color="yellow">{label}: </Text>
      <Text>{value}</Text>
      <Text color="yellow">▎</Text>
    </Box>
  );
}
```

- [ ] **Step 2: Wire prompts into App.tsx**

Add Ctrl+N (create room) and Ctrl+J (join room) prompt handling to `App.tsx`:
- When Ctrl+N pressed: show `<Prompt label="Room title" />`, on submit call `client.createChat({ title })`, refresh rooms.
- When Ctrl+J pressed: show `<Prompt label="Room ID" />`, on submit call `client.joinChat(id)`, refresh rooms.
- Escape dismisses prompt.

Add `promptMode` state to App: `null | "create" | "join"`. When set, render `<Prompt>` over the input bar.

- [ ] **Step 3: Commit**

```bash
git add -f tools/chat-cli/src/tui/Prompt.tsx tools/chat-cli/src/tui/App.tsx
git commit -m "feat(chat-cli): TUI inline prompts for create/join"
```

---

## Task 14: Build & Publish Setup

**Files:**
- Verify: `tools/chat-cli/bin/chat-now.js`
- Verify: `tools/chat-cli/package.json`

- [ ] **Step 1: Build the project**

Run: `cd tools/chat-cli && npm run build`
Expected: `dist/` directory created with compiled JS

- [ ] **Step 2: Test CLI entry point**

Run: `cd tools/chat-cli && node bin/chat-now.js --help`
Expected: Shows help with all commands listed

- [ ] **Step 3: Test with npx**

Run: `cd tools/chat-cli && npx . --help`
Expected: Same help output

- [ ] **Step 4: Verify with Bun**

Run: `cd tools/chat-cli && bun bin/chat-now.js --help`
Expected: Same help output (if Bun installed)

- [ ] **Step 5: Commit**

```bash
git add -f tools/chat-cli/dist/ tools/chat-cli/bin/
git commit -m "feat(chat-cli): build and publish-ready package"
```

---

## Task 15: Integration Test

- [ ] **Step 1: Test init with a fresh identity**

Run: `cd tools/chat-cli && node bin/chat-now.js init --actor u/testcli`
Expected: Registers with worker, shows recovery code, writes config.

- [ ] **Step 2: Test create room**

Run: `cd tools/chat-cli && node bin/chat-now.js create --title "test-room" --pretty`
Expected: JSON with chat id, kind=room, title.

- [ ] **Step 3: Test send + messages**

Run:
```bash
CHAT_ID=$(node bin/chat-now.js create --title test | jq -r .id)
node bin/chat-now.js send "$CHAT_ID" "hello from chat-now"
node bin/chat-now.js messages "$CHAT_ID" --pretty
```
Expected: Message appears in list.

- [ ] **Step 4: Test TUI launches**

Run: `cd tools/chat-cli && node bin/chat-now.js`
Expected: Three-panel TUI renders. Ctrl+Q exits.

- [ ] **Step 5: Final commit**

```bash
git add -f tools/chat-cli/
git commit -m "feat(chat-cli): chat-now v0.1.0 — TUI + CLI ready for npm"
```
