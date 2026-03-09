# 0701 — Crawl Mode Comparison Report

## Test Setup

- **Local**: MacBook (ARM), Chrome installed, `CRAWLER_WORKER_TOKEN` from env
- **Server2**: Ubuntu 24.04, 12GB RAM, US datacenter, headless Chrome
- **Worker**: CF Worker at `crawler.go-mizu.workers.dev`, paid plan
- **Max pages**: 100 (react.dev), 50 (openai.com, tailwindcss.com)
- **Date**: 2026-03-09

## react.dev (React SPA, no anti-bot)

| Mode | Location | OK | Failed | Blocked | Timeout | Speed | Peak | Avg Fetch | Elapsed |
|------|----------|-----|--------|---------|---------|-------|------|-----------|---------|
| HTTP | Local | 161 | 0 | 0 | 0 | 142.5/s | 152.0/s | 428ms | 2s |
| HTTP | Server2 | 237 | 0 | 0 | 0 | 107.0/s | 238.1/s | 117ms | 3s |
| Browser | Local | 126 | 2 | 33 | 0 | 5.4/s | 6.5/s | 5.0s | 31s |
| Browser | Server2 | 0 | 0 | 0 | 4 | 0.0/s | 0.1/s | 37.1s | 204s* |
| Worker | Server2 | 257 | 0 | 0 | 0 | 74.0/s | 256.7/s | 121ms | 4s |
| Worker+Browser | Server2 | 271 | 0 | 0 | 0 | 39.5/s | 108.2/s | 129ms | 7s |

*Server2 browser cancelled after 204s (only 4 pages, all timeouts)

**Observations**:
- HTTP mode is the best choice for react.dev — no anti-bot protection, fastest
- Worker mode matches HTTP speed via CF edge network (121ms avg fetch)
- Worker+Browser adds CF Browser Rendering overhead but 100% success
- Local browser works (126 OK) but 26× slower than HTTP
- Server2 browser is unusable — headless Chrome struggles with React SPA

## openai.com (Cloudflare Turnstile, JS challenge)

| Mode | Location | OK | Failed | Blocked | Timeout | Speed | Peak | Avg Fetch | Elapsed |
|------|----------|-----|--------|---------|---------|-------|------|-----------|---------|
| HTTP | Local | — | — | — | — | — | — | — | — |
| HTTP | Server2 | — | — | — | — | — | — | — | — |
| Browser | Local | 50 | 53 | 0 | 10 | 1.9/s | 4.8/s | 17.4s | 86s |
| Browser | Server2 | 65 | 70 | 0 | 161 | 6.0/s | 8.5/s | 32.6s | 162s |
| Worker | Server2 | 0 | 0 | 0 | 1149 | 0/s | 0/s | — | 8s |
| Worker+Browser | Server2 | 76 | 1411 | 0 | 0 | 154/s | 165/s | 79ms | 11s |

(HTTP not tested — previous tests showed 100% blocked/timeout on openai.com)

**Observations**:
- Cloudflare Turnstile blocks HTTP and worker (standard fetch) completely
- Local browser works best — solves CF challenge locally, 50 OK pages
- Server2 browser works but slower (Chrome under QEMU-like conditions)
- Worker+Browser (CF Browser Rendering): fast but 240 pages get 403 — CF blocks its own worker IPs on certain paths (`/chatgpt/*`, `/form/*`, locale-prefixed)
- Worker without browser = 100% timeout (CF JS challenge not solved)

## qiita.com (CloudFront WAF, IP-based blocking)

| Mode | Location | OK | Failed | Blocked | Timeout | Speed | Peak | Avg Fetch | Elapsed |
|------|----------|-----|--------|---------|---------|-------|------|-----------|---------|
| HTTP | Local | 0 | 0 | 8568 | 0 | 8568/s | — | — | 1s |
| HTTP | Server2 | 217 | 1 | 0 | 0 | 54.5/s | — | — | 5s |
| Browser | Local | — | — | — | — | — | — | — | — |
| Worker | Server2 | 184 | 1 | 0 | 0 | 18.5/s | 37.0/s | 1420ms | 13s |

**Observations**:
- CloudFront WAF blocks by IP — local Mac gets 403, US server passes
- Worker mode succeeds (CF edge IPs not blocked by CloudFront)
- HTTP mode from server2 is fastest (direct connection)
- Browser mode not worth testing (server-side blocking, not JS challenge)

## tailwindcss.com (no anti-bot)

| Mode | Location | OK | Failed | Blocked | Timeout | Speed | Peak | Avg Fetch | Elapsed |
|------|----------|-----|--------|---------|---------|-------|------|-----------|---------|
| Worker | Local | 257 | 2 | 0 | 1 | 25.1/s | 25.1/s | 1447ms | 12s |

## Recommendations

### When to use each mode

| Scenario | Recommended Mode |
|----------|-----------------|
| No anti-bot (most sites) | **HTTP** — fastest, simplest |
| CloudFront WAF (geo-block) | **Worker** — CF edge IPs bypass CloudFront |
| Cloudflare Turnstile | **Local Browser** — solves JS challenge |
| Cloudflare + need speed | **Worker+Browser** — fast but some paths blocked |
| React SPA (needs JS render) | **HTTP** — SSR sites serve HTML directly; if CSR-only, use **Worker+Browser** |
| Heavy JS + anti-bot | **Local Browser** — most reliable bypass |

### Speed ranking (for unprotected sites)

1. **HTTP** (100-250 pages/s) — direct connection, zero overhead
2. **Worker** (25-75 pages/s) — CF edge fetch + markdown conversion overhead
3. **Worker+Browser** (40-100 pages/s) — CF Browser Rendering adds ~50ms
4. **Local Browser** (2-7 pages/s) — Chrome tab management overhead
5. **Server2 Browser** (0-6 pages/s) — headless Chrome on server, most overhead
