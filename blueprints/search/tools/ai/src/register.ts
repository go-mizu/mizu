/**
 * Background auto-registration for Perplexity accounts.
 * Ported from pkg/dcrawler/perplexity/register.go
 *
 * Runs via ctx.waitUntil() — no public API exposure.
 * Full step-by-step logging to KV (accounts:log) for debugging.
 *
 * Flow:
 *   1. Acquire lock (prevent concurrent registration)
 *   2. Init anonymous session (CSRF + cookies)
 *   3. Create temp email (tries providers in tier order)
 *   4. POST signin request (sends magic link to email)
 *   5. Poll email inbox for magic link (25s timeout)
 *   6. GET magic link callback (complete auth + follow redirects)
 *   7. Save authenticated session to KV as account
 *   8. Release lock
 */

import { createTempEmail } from './email'
import { AccountManager } from './accounts'
import { ENDPOINTS, CHROME_HEADERS, MAGIC_LINK_REGEX, SIGNIN_SUBJECT } from './config'
import type { AccountStore } from './storage'
import type { SessionState } from './types'

const EMAIL_TIMEOUT_MS = 25000

/** Extract cookies from Set-Cookie headers. */
function extractCookies(resp: Response, existing: string = ''): string {
  const cookies = new Map<string, string>()
  if (existing) {
    for (const part of existing.split('; ')) {
      const eq = part.indexOf('=')
      if (eq > 0) cookies.set(part.slice(0, eq), part.slice(eq + 1))
    }
  }
  const setCookies = resp.headers.getAll?.('set-cookie') ?? []
  for (const sc of setCookies) {
    const nameVal = sc.split(';')[0]
    const eq = nameVal.indexOf('=')
    if (eq > 0) cookies.set(nameVal.slice(0, eq).trim(), nameVal.slice(eq + 1).trim())
  }
  return Array.from(cookies.entries()).map(([k, v]) => `${k}=${v}`).join('; ')
}

/** Extract CSRF token from cookies or JSON response. */
function extractCSRF(cookies: string, responseBody?: string): string {
  if (responseBody) {
    try {
      const json = JSON.parse(responseBody)
      if (json.csrfToken) return json.csrfToken
    } catch { /* not JSON */ }
  }
  const match = cookies.match(/next-auth\.csrf-token=([^;]+)/)
  if (match) {
    const val = match[1]
    const parts = val.split('%')
    if (parts.length > 1) return parts[0]
    try {
      const decoded = decodeURIComponent(val)
      const pipeParts = decoded.split('|')
      if (pipeParts.length > 1) return pipeParts[0]
      return decoded
    } catch { return val }
  }
  return ''
}

/**
 * Run background auto-registration.
 * Call this via ctx.waitUntil(backgroundRegister(kv)).
 * Logs every step to KV. Does nothing if lock is held or accounts are sufficient.
 */
export async function backgroundRegister(store: AccountStore, accountSecret: string = 'default-dev-secret'): Promise<void> {
  const am = new AccountManager(store, accountSecret)
  const t0 = Date.now()

  // Check if we actually need registration
  const needsReg = await am.needsRegistration()
  if (!needsReg) return

  // Try to acquire lock
  const locked = await am.tryLock()
  if (!locked) {
    await am.log({
      timestamp: new Date().toISOString(),
      event: 'start',
      message: 'Registration skipped — lock held by another request',
    })
    return
  }

  try {
    await am.log({
      timestamp: new Date().toISOString(),
      event: 'start',
      message: 'Background registration starting...',
    })

    // Step 1: Init anonymous session
    let cookies = ''
    const sessionResp = await fetch(ENDPOINTS.session, {
      headers: { ...CHROME_HEADERS },
      redirect: 'manual',
    })
    cookies = extractCookies(sessionResp, cookies)

    const csrfResp = await fetch(ENDPOINTS.csrf, {
      headers: { ...CHROME_HEADERS, Cookie: cookies },
      redirect: 'manual',
    })
    cookies = extractCookies(csrfResp, cookies)
    const csrfBody = await csrfResp.text()
    const csrfToken = extractCSRF(cookies, csrfBody)

    if (!csrfToken) {
      throw new Error(`CSRF extraction failed. Session status: ${sessionResp.status}, CSRF status: ${csrfResp.status}, cookies: ${cookies.length} chars`)
    }

    // Step 2: Create temp email
    const { client: emailClient, provider } = await createTempEmail()
    const email = emailClient.email()

    await am.log({
      timestamp: new Date().toISOString(),
      event: 'email_created',
      message: `Temp email created via ${provider}: ${email}`,
      provider,
      email,
      durationMs: Date.now() - t0,
    })

    // Step 3: Request magic link
    const formData = `email=${encodeURIComponent(email)}&csrfToken=${encodeURIComponent(csrfToken)}&callbackUrl=${encodeURIComponent('https://www.perplexity.ai/')}&json=true`

    const signinResp = await fetch(ENDPOINTS.signin, {
      method: 'POST',
      headers: {
        ...CHROME_HEADERS,
        'Content-Type': 'application/x-www-form-urlencoded',
        'Cookie': cookies,
        'Accept': '*/*',
        'Sec-Fetch-Dest': 'empty',
        'Sec-Fetch-Mode': 'cors',
        'Sec-Fetch-Site': 'same-origin',
        'Origin': 'https://www.perplexity.ai',
        'Referer': 'https://www.perplexity.ai/',
      },
      body: formData,
    })

    const signinBody = await signinResp.text()
    if (!signinResp.ok) {
      throw new Error(`Signin failed: HTTP ${signinResp.status} — ${signinBody.slice(0, 300)}`)
    }

    await am.log({
      timestamp: new Date().toISOString(),
      event: 'signin_sent',
      message: `Signin request sent (HTTP ${signinResp.status}). Waiting for email...`,
      email,
      durationMs: Date.now() - t0,
    })

    // Step 4: Wait for magic link email
    const emailBody = await emailClient.waitForMessage(SIGNIN_SUBJECT, EMAIL_TIMEOUT_MS)

    const match = emailBody.match(MAGIC_LINK_REGEX)
    if (!match?.[1]) {
      throw new Error(`Magic link not found in email body (${emailBody.length} chars). Body preview: ${emailBody.slice(0, 200)}`)
    }
    const magicLink = match[1]

    await am.log({
      timestamp: new Date().toISOString(),
      event: 'email_received',
      message: `Magic link received. Link length: ${magicLink.length}`,
      email,
      durationMs: Date.now() - t0,
    })

    // Step 5: Complete auth via magic link (follow redirects)
    let authCookies = cookies
    const authResp = await fetch(magicLink, {
      headers: { ...CHROME_HEADERS, Cookie: authCookies },
      redirect: 'manual',
    })
    authCookies = extractCookies(authResp, authCookies)

    let location = authResp.headers.get('location')
    let redirectCount = 0
    for (let i = 0; i < 5 && location; i++) {
      const url = location.startsWith('http') ? location : `https://www.perplexity.ai${location}`
      const redirectResp = await fetch(url, {
        headers: { ...CHROME_HEADERS, Cookie: authCookies },
        redirect: 'manual',
      })
      authCookies = extractCookies(redirectResp, authCookies)
      location = redirectResp.headers.get('location')
      redirectCount++
    }

    const finalCsrf = extractCSRF(authCookies)

    await am.log({
      timestamp: new Date().toISOString(),
      event: 'auth_complete',
      message: `Auth complete. Redirects: ${redirectCount}. Cookies: ${authCookies.length} chars. CSRF: ${finalCsrf ? 'yes' : 'no'}`,
      email,
      durationMs: Date.now() - t0,
    })

    // Step 6: Save account
    const authSession: SessionState = {
      csrfToken: finalCsrf || csrfToken,
      cookies: authCookies,
      createdAt: new Date().toISOString(),
    }

    const accountId = await am.addAccount(email, provider, emailClient.password(), authSession, 5)

    await am.log({
      timestamp: new Date().toISOString(),
      event: 'account_saved',
      message: `Account registered: ${email} (id: ${accountId}, proQueries: 5)`,
      provider,
      email,
      accountId,
      durationMs: Date.now() - t0,
    })
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    const stack = e instanceof Error ? e.stack : undefined
    await am.log({
      timestamp: new Date().toISOString(),
      event: 'error',
      message: `Registration failed: ${msg}`,
      error: stack || msg,
      durationMs: Date.now() - t0,
    })
  } finally {
    await am.unlock()
  }
}
