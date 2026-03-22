/**
 * Bot signal heuristic middleware.
 *
 * Scores incoming requests based on freely available signals:
 * - Datacenter ASN (cloud providers)
 * - User-Agent quality
 * - Browser header presence
 * - cf.botManagement.score (if available)
 *
 * Applied only to unauthenticated browser-facing endpoints.
 */

import type { Context, Next } from "hono";
import type { Env, Variables } from "../types";

type C = Context<{ Bindings: Env; Variables: Variables }>;

// Known datacenter / cloud provider ASNs
const DATACENTER_ASNS = new Set([
  14061, // DigitalOcean
  16276, // OVH
  24940, // Hetzner
  63949, // Akamai/Linode
  14618, 16509, // AWS
  15169, 396982, // Google Cloud
  8075, // Microsoft Azure
  13238, // Yandex Cloud
  45102, // Alibaba Cloud
  20473, // Vultr
]);

// Known bot/automation User-Agent patterns
const BOT_UA_PATTERNS = /\b(curl|wget|python-requests|httpx|node-fetch|axios|scrapy|go-http-client|java\/|okhttp|libwww-perl|mechanize|httpclient|headlesschrome)\b/i;

export interface BotScore {
  score: number;
  reasons: string[];
}

/** Compute a bot suspicion score (0-100). Higher = more suspicious. */
export function computeBotScore(request: Request): BotScore {
  let score = 0;
  const reasons: string[] = [];
  const cf = (request as any).cf;

  // 1. Datacenter ASN
  if (cf?.asn && DATACENTER_ASNS.has(cf.asn)) {
    score += 30;
    reasons.push(`datacenter-asn:${cf.asn}`);
  }

  // 2. User-Agent
  const ua = request.headers.get("User-Agent") || "";
  if (!ua) {
    score += 25;
    reasons.push("no-user-agent");
  } else if (ua.length < 10) {
    score += 20;
    reasons.push("short-user-agent");
  } else if (BOT_UA_PATTERNS.test(ua)) {
    score += 20;
    reasons.push("bot-user-agent");
  }

  // 3. Browser headers
  if (!request.headers.get("Accept-Language")) {
    score += 15;
    reasons.push("no-accept-language");
  }
  if (!request.headers.get("Sec-Fetch-Mode")) {
    score += 10;
    reasons.push("no-sec-fetch-mode");
  }

  // 4. cf.botManagement.score (Enterprise/Bot Management add-on)
  const botMgmt = cf?.botManagement;
  if (botMgmt && typeof botMgmt.score === "number") {
    if (botMgmt.score < 10) {
      score += 30;
      reasons.push(`cf-bot-score:${botMgmt.score}`);
    } else if (botMgmt.score < 30) {
      score += 15;
      reasons.push(`cf-bot-score:${botMgmt.score}`);
    }
    // Skip if verified bot (search engines)
    if (botMgmt.verifiedBot) {
      return { score: 0, reasons: ["verified-bot"] };
    }
  }

  return { score: Math.min(score, 100), reasons };
}

/** Threshold above which requests are blocked. */
const BLOCK_THRESHOLD = 60;

/**
 * Hono middleware: block requests with high bot suspicion score.
 * Skips if the request already has a Bearer token (API/CLI callers).
 */
export async function botGuard(c: C, next: Next) {
  // Skip for authenticated API callers (they already passed key-based auth)
  const authHeader = c.req.header("Authorization");
  if (authHeader?.startsWith("Bearer ")) {
    return next();
  }

  const { score, reasons } = computeBotScore(c.req.raw);

  if (score >= BLOCK_THRESHOLD) {
    console.log(
      JSON.stringify({
        level: "warn",
        component: "bot-guard",
        action: "blocked",
        score,
        reasons,
        ip: c.req.header("CF-Connecting-IP") || "unknown",
        ua: c.req.header("User-Agent")?.slice(0, 80) || "",
        ts: Date.now(),
      }),
    );
    return c.json(
      { error: "forbidden", message: "Request blocked" },
      403,
    );
  }

  return next();
}
