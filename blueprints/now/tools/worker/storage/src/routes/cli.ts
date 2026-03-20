/**
 * CLI binary distribution routes.
 *
 * GET /cli/releases/:version/:filename
 *   → resolves "latest" to actual version
 *   → logs download (D1 + structured console)
 *   → streams binary from R2
 *
 * GET /cli/dl?t=<token>&f=<filename>
 *   → verifies pre-signed token (HMAC-SHA256)
 *   → streams binary from R2
 *   → use for shareable download links
 *
 * Binaries stored in R2 at: _cli/{version}/{filename}
 * Latest version pointer:   _cli/latest-version
 */

import type { App } from "../types";

// ── HMAC signing ─────────────────────────────────────────────────────

async function hmac(data: string, secret: string): Promise<string> {
  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const sig = await crypto.subtle.sign(
    "HMAC",
    key,
    new TextEncoder().encode(data),
  );
  return Array.from(new Uint8Array(sig), (b) =>
    b.toString(16).padStart(2, "0"),
  ).join("");
}

function parseTarget(filename: string): [string, string] {
  const m = filename.match(/storage-(\w+)-(\w+)/);
  return m ? [m[1], m[2].replace(/\.exe$/, "")] : ["unknown", "unknown"];
}

// ── Routes ───────────────────────────────────────────────────────────

export function register(app: App) {
  /**
   * Primary download endpoint — logs and streams directly.
   */
  app.get("/cli/releases/:version/:filename", async (c) => {
    const version = c.req.param("version")!;
    const filename = c.req.param("filename")!;

    // Resolve "latest" to actual version
    let resolved = version;
    if (version === "latest") {
      const meta = await c.env.BUCKET.get("_cli/latest-version");
      resolved = meta ? (await meta.text()).trim() : "1.0.0";
    }

    // Fetch binary from R2
    const key = `_cli/${resolved}/${filename}`;
    const obj = await c.env.BUCKET.get(key);
    if (!obj) {
      return c.json(
        { error: "not_found", message: `CLI binary not found: ${filename} (v${resolved})` },
        404,
      );
    }

    // Log download to D1 (fire-and-forget)
    const [os, arch] = parseTarget(filename);
    const ip = c.req.header("CF-Connecting-IP") || c.req.header("X-Forwarded-For") || "";
    const country = c.req.header("CF-IPCountry") || "";
    const ua = c.req.header("User-Agent") || "";
    const ts = Date.now();

    c.executionCtx.waitUntil(
      c.env.DB.prepare(
        `INSERT INTO cli_downloads
           (version, filename, os, arch, ip, country, user_agent, referrer, ts)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
      )
        .bind(resolved, filename, os, arch, ip, country, ua, c.req.header("Referer") || "", ts)
        .run()
        .catch(() => {}),
    );

    // Structured log for real-time Workers Logs / Logpush
    console.log(
      JSON.stringify({ event: "cli_download", version: resolved, filename, os, arch, ip, country, ua, ts }),
    );

    // Stream binary
    return new Response(obj.body, {
      headers: {
        "Content-Type": "application/octet-stream",
        "Content-Length": obj.size.toString(),
        "Content-Disposition": `attachment; filename="${filename}"`,
        "Cache-Control": "public, max-age=3600",
        "X-Version": resolved,
      },
    });
  });

  /**
   * Upload CLI binary to R2.
   * Auth: X-Admin-Key header must match CLI_UPLOAD_KEY (or SIGNING_KEY).
   */
  app.put("/cli/upload/:version/:filename", async (c) => {
    const authKey = c.req.header("X-Admin-Key");
    const validKey = c.env.CLI_UPLOAD_KEY || c.env.SIGNING_KEY;
    if (!authKey || authKey !== validKey) {
      return c.json({ error: "unauthorized" }, 401);
    }

    const version = c.req.param("version")!;
    const filename = c.req.param("filename")!;
    const key = `_cli/${version}/${filename}`;

    const body = await c.req.arrayBuffer();
    await c.env.BUCKET.put(key, body, {
      httpMetadata: { contentType: "application/octet-stream" },
    });

    // If this is a versioned upload, also update latest-version
    if (version !== "latest-version") {
      await c.env.BUCKET.put("_cli/latest-version", version, {
        httpMetadata: { contentType: "text/plain" },
      });
    }

    return c.json({ ok: true, key, size: body.byteLength, version });
  });

  /**
   * Pre-signed download URL endpoint.
   *
   * Generate a link: POST /cli/sign { key, ttl? }  (admin only, future)
   * Or create tokens manually for external distribution.
   */
  app.get("/cli/dl", async (c) => {
    const token = c.req.query("t");
    const filename = c.req.query("f") || "storage";

    if (!token) {
      return c.json({ error: "bad_request", message: "Missing download token" }, 400);
    }

    const dot = token.indexOf(".");
    if (dot === -1) {
      return c.json({ error: "unauthorized", message: "Invalid token" }, 401);
    }

    const payload = token.slice(0, dot);
    const sig = token.slice(dot + 1);
    const expected = await hmac(payload, c.env.SIGNING_KEY);
    if (sig !== expected) {
      return c.json({ error: "unauthorized", message: "Invalid signature" }, 401);
    }

    let claims: { k: string; x: number };
    try {
      claims = JSON.parse(atob(payload.replace(/-/g, "+").replace(/_/g, "/")));
    } catch {
      return c.json({ error: "unauthorized", message: "Malformed token" }, 401);
    }

    if (claims.x < Date.now()) {
      return c.json({ error: "unauthorized", message: "Download link expired" }, 401);
    }

    const obj = await c.env.BUCKET.get(claims.k);
    if (!obj) {
      return c.json({ error: "not_found", message: "Binary not found" }, 404);
    }

    return new Response(obj.body, {
      headers: {
        "Content-Type": "application/octet-stream",
        "Content-Length": obj.size.toString(),
        "Content-Disposition": `attachment; filename="${filename}"`,
        "Cache-Control": "private, no-store",
      },
    });
  });
}
