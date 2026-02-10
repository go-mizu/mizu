import type { Env, StoredSession, LoginError } from './types'
import { InstagramClient } from './instagram'
import { webBaseURL, appID, webUserAgent } from './config'

const KV_SESSION = '_session'
const KV_LOGIN_ERROR = '_login_error'
const KV_LOGIN_LOCK = '_login_lock'

// Cooldowns in milliseconds
const COOLDOWN_CHALLENGE = 24 * 60 * 60 * 1000  // 24h for challenge/2FA
const MAX_BACKOFF = 60 * 60 * 1000               // 1h max for unknown errors

function extractCookies(headers: Headers): Record<string, string> {
  const cookies: Record<string, string> = {}
  // getSetCookie() exists in CF Workers runtime but may not be in type defs
  const setCookieHeaders = (headers as any).getSetCookie?.() as string[] | undefined
  if (setCookieHeaders) {
    for (const header of setCookieHeaders) {
      const match = header.match(/^([^=]+)=([^;]*)/)
      if (match) cookies[match[1].trim()] = match[2].trim()
    }
  }
  return cookies
}

export async function instagramLogin(email: string, password: string): Promise<StoredSession> {
  // Step 1: GET instagram.com to acquire CSRF token and mid cookie
  const initResp = await fetch(`${webBaseURL}/`, {
    headers: {
      'user-agent': webUserAgent,
      accept: 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
      'accept-language': 'en-US,en;q=0.8',
    },
    redirect: 'manual',
  })
  const initBody = await initResp.text()

  const initCookies = extractCookies(initResp.headers)
  let csrfToken = initCookies['csrftoken'] || ''
  const mid = initCookies['mid'] || ''

  // Fallback: extract csrf_token from HTML/JSON in page body
  if (!csrfToken) {
    const csrfMatch = initBody.match(/"csrf_token":"([^"]+)"/) || initBody.match(/csrftoken=([a-zA-Z0-9]+)/)
    if (csrfMatch) csrfToken = csrfMatch[1]
  }

  if (!csrfToken) {
    throw new Error('Failed to get CSRF token from Instagram')
  }

  // Step 2: POST login — forward ALL cookies from initial GET
  const ts = Math.floor(Date.now() / 1000).toString()
  const encPassword = `#PWD_INSTAGRAM_BROWSER:0:${ts}:${password}`

  const body = new URLSearchParams()
  body.set('enc_password', encPassword)
  body.set('username', email)
  body.set('queryParams', '{}')
  body.set('optIntoOneTap', 'false')
  body.set('trustedDeviceRecords', '{}')

  const cookieStr = Object.entries(initCookies).map(([k, v]) => `${k}=${v}`).join('; ')
  const igDid = initCookies['ig_did'] || ''
  const loginHeaders: Record<string, string> = {
    'user-agent': webUserAgent,
    accept: '*/*',
    'accept-language': 'en-US,en;q=0.8',
    'content-type': 'application/x-www-form-urlencoded',
    'x-csrftoken': csrfToken,
    'x-ig-app-id': appID,
    'x-requested-with': 'XMLHttpRequest',
    origin: 'https://www.instagram.com',
    referer: 'https://www.instagram.com/',
    cookie: cookieStr,
    'sec-ch-ua': '"Google Chrome";v="142", "Chromium";v="142"',
    'sec-ch-ua-mobile': '?0',
    'sec-ch-ua-platform': '"Windows"',
    'sec-fetch-dest': 'empty',
    'sec-fetch-mode': 'cors',
    'sec-fetch-site': 'same-origin',
  }
  if (igDid) loginHeaders['x-web-device-id'] = igDid

  const loginResp = await fetch(`${webBaseURL}/api/v1/web/accounts/login/ajax/`, {
    method: 'POST',
    headers: loginHeaders,
    body: body.toString(),
    redirect: 'manual',
  })

  const loginBody = await loginResp.text()
  let loginData: Record<string, unknown>
  try {
    loginData = JSON.parse(loginBody)
  } catch {
    throw new Error(`Login response not JSON: ${loginBody.slice(0, 200)}`)
  }

  // Check error conditions
  if (loginData.two_factor_required) {
    const err = new Error('Two-factor authentication required')
    ;(err as any).errorType = '2fa_required'
    throw err
  }
  if (loginData.checkpoint_url || loginData.error_type === 'ChallengeRequired') {
    const err = new Error('Challenge/checkpoint required: ' + (loginData.checkpoint_url || ''))
    ;(err as any).errorType = 'challenge_required'
    throw err
  }
  if (loginData.user === false) {
    const err = new Error('User not found')
    ;(err as any).errorType = 'wrong_password'
    throw err
  }
  if (!loginData.authenticated) {
    const msg = (loginData.message as string) || 'Unknown error'
    const err = new Error(`Authentication failed: ${msg}`)
    ;(err as any).errorType = loginData.user === true ? 'wrong_password' : 'unknown'
    throw err
  }

  // Extract cookies from login response
  const respCookies = extractCookies(loginResp.headers)

  const sessionId = respCookies['sessionid'] || ''
  const newCsrf = respCookies['csrftoken'] || csrfToken
  const dsUserId = respCookies['ds_user_id'] || (loginData.userId as string) || ''
  const newMid = respCookies['mid'] || mid
  const respIgDid = respCookies['ig_did'] || igDid

  if (!sessionId) {
    throw new Error('Login succeeded but no sessionid cookie returned')
  }

  return {
    sessionId,
    csrfToken: newCsrf,
    dsUserId,
    mid: newMid,
    igDid: respIgDid,
    loginAt: new Date().toISOString(),
    source: 'login',
  }
}

export class SessionManager {
  private env: Env

  constructor(env: Env) {
    this.env = env
  }

  async getClient(): Promise<InstagramClient> {
    // 1. Try KV session first (from auto-login)
    const kvSession = await this.env.KV.get(KV_SESSION, 'json') as StoredSession | null
    if (kvSession?.sessionId) {
      return new InstagramClient(kvSession.sessionId, kvSession.csrfToken, kvSession.dsUserId, kvSession.mid, kvSession.igDid)
    }

    // 2. Fall back to env secrets
    if (this.env.INSTA_SESSION_ID) {
      return new InstagramClient(this.env.INSTA_SESSION_ID, this.env.INSTA_CSRF_TOKEN, this.env.INSTA_DS_USER_ID, this.env.INSTA_MID, this.env.INSTA_IG_DID)
    }

    // 3. Try auto-login if credentials available
    if (this.canLoginSync()) {
      const loginErr = await this.env.KV.get(KV_LOGIN_ERROR, 'json') as LoginError | null
      if (!loginErr || this.cooldownExpired(loginErr)) {
        try {
          return await this.doLogin()
        } catch (e) {
          await this.recordLoginError(e)
        }
      }
    }

    // 4. Return empty client (will use public endpoints)
    return new InstagramClient('', '', '', '', '')
  }

  async refreshSession(): Promise<InstagramClient> {
    // Delete existing KV session
    await this.env.KV.delete(KV_SESSION)

    if (!this.canLoginSync()) {
      throw new Error('No login credentials configured (INSTA_EMAIL + INSTA_PWD)')
    }

    // Check login lock
    const lock = await this.env.KV.get(KV_LOGIN_LOCK)
    if (lock) {
      throw new Error('Login already in progress (locked for 60s)')
    }

    return this.doLogin()
  }

  async getStatus(): Promise<Record<string, unknown>> {
    const kvSession = await this.env.KV.get(KV_SESSION, 'json') as StoredSession | null
    const loginErr = await this.env.KV.get(KV_LOGIN_ERROR, 'json') as LoginError | null

    let source: string
    let sessionOk = false

    if (kvSession?.sessionId) {
      source = 'kv (auto-login)'
      sessionOk = true
    } else if (this.env.INSTA_SESSION_ID) {
      source = 'secrets (static)'
      sessionOk = true
    } else {
      source = 'none'
    }

    // Check session health
    let health: Record<string, unknown> | undefined
    if (sessionOk) {
      const client = kvSession?.sessionId
        ? new InstagramClient(kvSession.sessionId, kvSession.csrfToken, kvSession.dsUserId, kvSession.mid, kvSession.igDid)
        : new InstagramClient(this.env.INSTA_SESSION_ID, this.env.INSTA_CSRF_TOKEN, this.env.INSTA_DS_USER_ID, this.env.INSTA_MID, this.env.INSTA_IG_DID)
      health = await client.checkSession()
    }

    return {
      session: {
        source,
        active: sessionOk,
        loginAt: kvSession?.loginAt || null,
        health,
      },
      login: {
        hasCredentials: this.canLoginSync(),
        lastError: loginErr,
        canRetry: loginErr ? this.cooldownExpired(loginErr) : true,
      },
      timestamp: new Date().toISOString(),
    }
  }

  private canLoginSync(): boolean {
    return !!(this.env.INSTA_EMAIL && this.env.INSTA_PWD)
  }

  private cooldownExpired(err: LoginError): boolean {
    const elapsed = Date.now() - err.timestamp
    switch (err.errorType) {
      case 'challenge_required':
      case '2fa_required':
        return elapsed >= COOLDOWN_CHALLENGE
      case 'wrong_password':
        return false // never retry
      default: {
        // Exponential backoff: 1min, 2min, 4min, ... up to 1h
        const backoff = Math.min(60_000 * Math.pow(2, err.attempts - 1), MAX_BACKOFF)
        return elapsed >= backoff
      }
    }
  }

  private async doLogin(): Promise<InstagramClient> {
    // Acquire lock (60s TTL)
    await this.env.KV.put(KV_LOGIN_LOCK, '1', { expirationTtl: 60 })

    try {
      const session = await instagramLogin(this.env.INSTA_EMAIL, this.env.INSTA_PWD)

      // Store session in KV (no expiration — persistent until refreshed)
      await this.env.KV.put(KV_SESSION, JSON.stringify(session))

      // Clear any previous login errors
      await this.env.KV.delete(KV_LOGIN_ERROR)

      return new InstagramClient(session.sessionId, session.csrfToken, session.dsUserId, session.mid, session.igDid)
    } finally {
      // Release lock
      await this.env.KV.delete(KV_LOGIN_LOCK)
    }
  }

  private async recordLoginError(e: unknown): Promise<void> {
    const existing = await this.env.KV.get(KV_LOGIN_ERROR, 'json') as LoginError | null
    const errorType = (e as any)?.errorType || 'unknown'
    const loginErr: LoginError = {
      error: e instanceof Error ? e.message : String(e),
      errorType,
      timestamp: Date.now(),
      attempts: (existing?.attempts || 0) + 1,
    }
    await this.env.KV.put(KV_LOGIN_ERROR, JSON.stringify(loginErr))
  }
}
