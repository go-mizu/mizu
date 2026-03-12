/**
 * Proxy layer (L3) for Cloudflare Browser Rendering REST API.
 * When CF_ACCOUNT_ID and CF_API_TOKEN secrets are set, requests are
 * forwarded to api.cloudflare.com and the response is returned verbatim.
 */

export interface CfProxyResult {
  ok: boolean;
  rateLimited: boolean;
  status: number;
  body: unknown;
  blob: Blob | null;
  browserMsUsed: string | null;
}

export function cfAvailable(env: { CF_ACCOUNT_ID?: string; CF_API_TOKEN?: string }): boolean {
  return Boolean(env.CF_ACCOUNT_ID && env.CF_API_TOKEN);
}

export async function proxyCF(
  endpoint: string,
  requestBody: unknown,
  env: { CF_ACCOUNT_ID: string; CF_API_TOKEN: string },
  binary = false
): Promise<CfProxyResult> {
  const url = `https://api.cloudflare.com/client/v4/accounts/${env.CF_ACCOUNT_ID}/browser-rendering/${endpoint}`;

  const res = await fetch(url, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${env.CF_API_TOKEN}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(requestBody),
  });

  const browserMsUsed = res.headers.get("X-Browser-Ms-Used");

  if (!res.ok) {
    let body: unknown = null;
    try { body = await res.json(); } catch { /* ignore */ }
    return {
      ok: false,
      rateLimited: res.status === 429,
      status: res.status,
      body,
      blob: null,
      browserMsUsed,
    };
  }

  if (binary) {
    return {
      ok: true,
      rateLimited: false,
      status: res.status,
      body: null,
      blob: await res.blob(),
      browserMsUsed,
    };
  }

  const body = await res.json();
  return { ok: true, rateLimited: false, status: res.status, body, blob: null, browserMsUsed };
}
