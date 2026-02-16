/**
 * Disposable email providers for auto-registration.
 * Ported from pkg/dcrawler/perplexity/email_*.go
 *
 * 5 providers across 3 tiers (private → session → public).
 * Each implements TempEmailClient interface.
 */

export interface TempEmailClient {
  email(): string
  password(): string  // stored password (empty for no-auth providers)
  waitForMessage(matchSubject: string, timeoutMs: number): Promise<string>
}

interface EmailProvider {
  name: string
  tier: number // 1=private, 2=session, 3=public
  create: () => Promise<TempEmailClient>
}

function randomString(n: number): string {
  const chars = 'abcdefghijklmnopqrstuvwxyz0123456789'
  let result = ''
  for (let i = 0; i < n; i++) {
    result += chars[Math.floor(Math.random() * chars.length)]
  }
  return result
}

function sleep(ms: number): Promise<void> {
  return new Promise(r => setTimeout(r, ms))
}

// --- mail.tm / mail.gw (Hydra API, JWT auth) ---

class MailTMClient implements TempEmailClient {
  private baseURL: string
  private _email = ''
  private _password = ''
  private token = ''

  constructor(baseURL: string) {
    this.baseURL = baseURL
  }

  async init(): Promise<void> {
    // Get available domain
    const domResp = await fetch(`${this.baseURL}/domains`)
    if (!domResp.ok) throw new Error(`domains HTTP ${domResp.status}`)
    const domData = await domResp.json() as { 'hydra:member': Array<{ domain: string; isActive: boolean }> }
    const domain = domData['hydra:member']?.find(d => d.isActive)?.domain
    if (!domain) throw new Error('no active domains')

    // Create account
    const user = `pplx${Date.now() % 100000}${randomString(5)}`
    this._email = `${user}@${domain}`
    const password = randomString(16)
    this._password = password

    const acctResp = await fetch(`${this.baseURL}/accounts`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ address: this._email, password }),
    })
    if (!acctResp.ok && acctResp.status !== 201) {
      const body = await acctResp.text()
      throw new Error(`create account HTTP ${acctResp.status}: ${body.slice(0, 200)}`)
    }
    await acctResp.text() // consume body

    // Get JWT token
    const tokenResp = await fetch(`${this.baseURL}/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ address: this._email, password }),
    })
    if (!tokenResp.ok) throw new Error(`token HTTP ${tokenResp.status}`)
    const tokenData = await tokenResp.json() as { token: string }
    this.token = tokenData.token
    if (!this.token) throw new Error('empty token')
  }

  email(): string { return this._email }
  password(): string { return this._password }

  async waitForMessage(matchSubject: string, timeoutMs: number): Promise<string> {
    const deadline = Date.now() + timeoutMs
    while (Date.now() < deadline) {
      const resp = await fetch(`${this.baseURL}/messages`, {
        headers: { Authorization: `Bearer ${this.token}` },
      })
      if (resp.ok) {
        const data = await resp.json() as { 'hydra:member': Array<{ id: string; subject: string }> }
        for (const msg of data['hydra:member'] || []) {
          if (msg.subject.includes(matchSubject)) {
            const fullResp = await fetch(`${this.baseURL}/messages/${msg.id}`, {
              headers: { Authorization: `Bearer ${this.token}` },
            })
            if (fullResp.ok) {
              const full = await fullResp.json() as { html: string[]; text: string }
              return full.html?.length ? full.html.join('') : full.text
            }
          }
        }
      }
      await sleep(3000)
    }
    throw new Error(`timeout waiting for email with subject "${matchSubject}"`)
  }
}

async function createMailTM(): Promise<TempEmailClient> {
  const c = new MailTMClient('https://api.mail.tm')
  await c.init()
  return c
}

async function createMailGW(): Promise<TempEmailClient> {
  const c = new MailTMClient('https://api.mail.gw')
  await c.init()
  return c
}

// --- Guerrilla Mail (session-based) ---

class GuerrillaClient implements TempEmailClient {
  private _email = ''
  private sidToken = ''

  async init(): Promise<void> {
    const resp = await fetch('https://api.guerrillamail.com/ajax.php?f=get_email_address', {
      headers: { 'User-Agent': 'Mozilla/5.0' },
    })
    if (!resp.ok) throw new Error(`guerrilla HTTP ${resp.status}`)
    const data = await resp.json() as { email_addr: string; sid_token: string }
    this._email = data.email_addr
    this.sidToken = data.sid_token
    if (!this._email || !this.sidToken) throw new Error('empty guerrilla response')
  }

  email(): string { return this._email }
  password(): string { return '' }

  async waitForMessage(matchSubject: string, timeoutMs: number): Promise<string> {
    const deadline = Date.now() + timeoutMs
    while (Date.now() < deadline) {
      const resp = await fetch(
        `https://api.guerrillamail.com/ajax.php?f=check_email&sid_token=${this.sidToken}&seq=0`,
        { headers: { 'User-Agent': 'Mozilla/5.0' } },
      )
      if (resp.ok) {
        const data = await resp.json() as { list: Array<{ mail_id: string; mail_subject: string }> }
        for (const msg of data.list || []) {
          if (msg.mail_subject?.includes(matchSubject)) {
            const fetchResp = await fetch(
              `https://api.guerrillamail.com/ajax.php?f=fetch_email&sid_token=${this.sidToken}&email_id=${msg.mail_id}`,
              { headers: { 'User-Agent': 'Mozilla/5.0' } },
            )
            if (fetchResp.ok) {
              const full = await fetchResp.json() as { mail_body: string }
              return full.mail_body || ''
            }
          }
        }
      }
      await sleep(3000)
    }
    throw new Error(`timeout waiting for email with subject "${matchSubject}"`)
  }
}

async function createGuerrilla(): Promise<TempEmailClient> {
  const c = new GuerrillaClient()
  await c.init()
  return c
}

// --- DropMail (GraphQL, session-based) ---

class DropMailClient implements TempEmailClient {
  private _email = ''
  private sessionID = ''

  async init(): Promise<void> {
    const resp = await fetch('https://dropmail.me/api/graphql/web-test-2', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ query: 'mutation { introduceSession { id, expiresAt, addresses { address } } }' }),
    })
    if (!resp.ok) throw new Error(`dropmail HTTP ${resp.status}`)
    const data = await resp.json() as { data: { introduceSession: { id: string; addresses: Array<{ address: string }> } } }
    const session = data.data?.introduceSession
    if (!session?.id || !session.addresses?.length) throw new Error('empty dropmail session')
    this._email = session.addresses[0].address
    this.sessionID = session.id
  }

  email(): string { return this._email }
  password(): string { return '' }

  async waitForMessage(matchSubject: string, timeoutMs: number): Promise<string> {
    const deadline = Date.now() + timeoutMs
    while (Date.now() < deadline) {
      const resp = await fetch('https://dropmail.me/api/graphql/web-test-2', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query: `query { session(id: "${this.sessionID}") { mails { rawSize, headerSubject, text, html } } }`,
        }),
      })
      if (resp.ok) {
        const data = await resp.json() as { data: { session: { mails: Array<{ headerSubject: string; text: string; html: string }> } } }
        for (const msg of data.data?.session?.mails || []) {
          if (msg.headerSubject?.includes(matchSubject)) {
            return msg.html || msg.text || ''
          }
        }
      }
      await sleep(3000)
    }
    throw new Error(`timeout waiting for email with subject "${matchSubject}"`)
  }
}

async function createDropMail(): Promise<TempEmailClient> {
  const c = new DropMailClient()
  await c.init()
  return c
}

// --- tempmail.lol (token auth) ---

class TempMailLolClient implements TempEmailClient {
  private _email = ''
  private token = ''

  async init(): Promise<void> {
    const resp = await fetch('https://api.tempmail.lol/generate')
    if (!resp.ok) throw new Error(`tempmail.lol HTTP ${resp.status}`)
    const data = await resp.json() as { address: string; token: string }
    if (!data.address || !data.token) throw new Error('empty tempmail.lol response')
    this._email = data.address
    this.token = data.token
  }

  email(): string { return this._email }
  password(): string { return '' }

  async waitForMessage(matchSubject: string, timeoutMs: number): Promise<string> {
    const deadline = Date.now() + timeoutMs
    while (Date.now() < deadline) {
      const resp = await fetch(`https://api.tempmail.lol/auth/${this.token}`)
      if (resp.ok) {
        const data = await resp.json() as { email: Array<{ subject: string; body: string; html: string }> }
        for (const msg of data.email || []) {
          if (msg.subject?.includes(matchSubject)) {
            return msg.html || msg.body || ''
          }
        }
      }
      await sleep(3000)
    }
    throw new Error(`timeout waiting for email with subject "${matchSubject}"`)
  }
}

async function createTempMailLol(): Promise<TempEmailClient> {
  const c = new TempMailLolClient()
  await c.init()
  return c
}

// --- tempmail.plus (public, no auth) ---

class TempMailPlusClient implements TempEmailClient {
  private _email: string
  private user: string

  constructor() {
    this.user = `pplx${Date.now() % 100000}${randomString(5)}`
    this._email = `${this.user}@mailto.plus`
  }

  email(): string { return this._email }
  password(): string { return '' }

  async waitForMessage(matchSubject: string, timeoutMs: number): Promise<string> {
    const deadline = Date.now() + timeoutMs
    while (Date.now() < deadline) {
      const resp = await fetch(`https://tempmail.plus/api/mails?email=${this._email}&epin=`)
      if (resp.ok) {
        const data = await resp.json() as { mail_list: Array<{ mail_id: number; subject: string }> }
        for (const msg of data.mail_list || []) {
          if (msg.subject?.includes(matchSubject)) {
            const fullResp = await fetch(`https://tempmail.plus/api/mails/${msg.mail_id}?email=${this._email}&epin=`)
            if (fullResp.ok) {
              const full = await fullResp.json() as { html: string; text: string }
              return full.html || full.text || ''
            }
          }
        }
      }
      await sleep(3000)
    }
    throw new Error(`timeout waiting for email with subject "${matchSubject}"`)
  }
}

async function createTempMailPlus(): Promise<TempEmailClient> {
  return new TempMailPlusClient()
}

// --- Provider registry with tiered fallback ---

const ALL_PROVIDERS: EmailProvider[] = [
  // Tier 1: Private (JWT/token auth)
  { name: 'mail.tm', tier: 1, create: createMailTM },
  { name: 'mail.gw', tier: 1, create: createMailGW },
  { name: 'tempmail.lol', tier: 1, create: createTempMailLol },
  // Tier 2: Session-based
  { name: 'guerrillamail', tier: 2, create: createGuerrilla },
  { name: 'dropmail', tier: 2, create: createDropMail },
  // Tier 3: Public (no auth)
  { name: 'tempmail.plus', tier: 3, create: createTempMailPlus },
]

/** Shuffle array in-place. */
function shuffle<T>(arr: T[]): T[] {
  for (let i = arr.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1));
    [arr[i], arr[j]] = [arr[j], arr[i]]
  }
  return arr
}

/**
 * Create a disposable email client.
 * Tries providers in tier order (private → session → public), shuffled within each tier.
 * Returns the first successful provider.
 */
export async function createTempEmail(): Promise<{ client: TempEmailClient; provider: string }> {
  // Group by tier, shuffle within each, then concatenate
  const byTier = new Map<number, EmailProvider[]>()
  for (const p of ALL_PROVIDERS) {
    const group = byTier.get(p.tier) || []
    group.push(p)
    byTier.set(p.tier, group)
  }

  const ordered: EmailProvider[] = []
  for (const tier of [1, 2, 3]) {
    const group = byTier.get(tier)
    if (group) ordered.push(...shuffle([...group]))
  }

  const errors: string[] = []
  for (const p of ordered) {
    try {
      const client = await p.create()
      return { client, provider: p.name }
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e)
      errors.push(`${p.name}: ${msg}`)
    }
  }

  throw new Error(`All ${ordered.length} email providers failed:\n  ${errors.join('\n  ')}`)
}
