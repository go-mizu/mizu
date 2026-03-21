/** Cloudflare Turnstile server-side validation. */

const SITEVERIFY_URL = "https://challenges.cloudflare.com/turnstile/v0/siteverify";

export interface TurnstileResult {
  success: boolean;
  "error-codes": string[];
  challenge_ts?: string;
  hostname?: string;
  action?: string;
}

/**
 * Validate a Turnstile token server-side.
 * Returns `{ success: true }` if valid, or `{ success: false, ... }` with error codes.
 * If `secretKey` is empty/undefined, returns success (graceful degradation).
 */
export async function validateTurnstile(
  token: string | null | undefined,
  secretKey: string | undefined,
  remoteIp?: string,
): Promise<TurnstileResult> {
  // Graceful degradation: skip if Turnstile is not configured
  if (!secretKey) {
    return { success: true, "error-codes": [] };
  }

  if (!token) {
    return { success: false, "error-codes": ["missing-input-response"] };
  }

  const body = new URLSearchParams();
  body.append("secret", secretKey);
  body.append("response", token);
  if (remoteIp) body.append("remoteip", remoteIp);

  try {
    const res = await fetch(SITEVERIFY_URL, {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: body.toString(),
    });
    return (await res.json()) as TurnstileResult;
  } catch {
    // Network error — fail open to avoid blocking legitimate users
    console.error("[turnstile] siteverify fetch failed");
    return { success: true, "error-codes": ["fetch-error"] };
  }
}
