import { webBaseURL, iPhoneBaseURL, appID, webUserAgent, iPhoneUserAgent } from './config'

export class InstagramClient {
  private sessionId: string
  private csrfToken: string
  private dsUserId: string
  private mid: string
  private igDid: string

  constructor(sessionId: string, csrfToken: string, dsUserId: string, mid: string, igDid: string) {
    this.sessionId = sessionId
    this.csrfToken = csrfToken
    this.dsUserId = dsUserId
    this.mid = mid || ''
    this.igDid = igDid || ''
  }

  private cookieString(): string {
    const parts = [
      `sessionid=${this.sessionId}`,
      `csrftoken=${this.csrfToken}`,
      `ds_user_id=${this.dsUserId}`,
    ]
    if (this.mid) parts.push(`mid=${this.mid}`)
    if (this.igDid) parts.push(`ig_did=${this.igDid}`)
    return parts.join('; ')
  }

  private webHeaders(): Record<string, string> {
    return {
      accept: '*/*',
      'accept-language': 'en-US,en;q=0.8',
      'user-agent': webUserAgent,
      'x-ig-app-id': appID,
      'x-instagram-ajax': '1',
      'x-requested-with': 'XMLHttpRequest',
      'x-csrftoken': this.csrfToken,
      origin: 'https://www.instagram.com',
      referer: 'https://www.instagram.com/',
      cookie: this.cookieString(),
      'sec-ch-ua': '"Google Chrome";v="142", "Chromium";v="142"',
      'sec-ch-ua-mobile': '?0',
      'sec-ch-ua-platform': '"Windows"',
      'sec-fetch-dest': 'empty',
      'sec-fetch-mode': 'cors',
      'sec-fetch-site': 'same-origin',
    }
  }

  private iphoneHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      accept: '*/*',
      'accept-language': 'en-US,en;q=0.8',
      'user-agent': iPhoneUserAgent,
      'x-ig-app-id': '124024574287414',
      'x-ig-capabilities': '36r/F/8=',
      'x-ig-connection-type': 'WiFi',
      'x-ig-app-locale': 'en-US',
      'x-ig-device-locale': 'en-US',
      'x-ig-mapped-locale': 'en-US',
      'x-fb-http-engine': 'Liger',
      cookie: this.cookieString(),
    }
    if (this.dsUserId) headers['ig-intended-user-id'] = this.dsUserId
    if (this.mid) headers['x-mid'] = this.mid
    if (this.igDid) {
      headers['x-ig-device-id'] = this.igDid
      headers['x-ig-family-device-id'] = this.igDid
    }
    if (this.dsUserId) headers['ig-u-ds-user-id'] = this.dsUserId
    if (this.csrfToken) headers['x-csrftoken'] = this.csrfToken
    return headers
  }

  // Fetch profile via web_profile_info
  async getProfileInfo(username: string): Promise<Record<string, unknown>> {
    const url = `${webBaseURL}/api/v1/users/web_profile_info/?username=${encodeURIComponent(username)}`
    const resp = await fetch(url, { headers: this.webHeaders() })
    await this.checkResponse(resp)
    return await resp.json() as Record<string, unknown>
  }

  // Check if the session is valid by making a simple request
  async checkSession(): Promise<{ ok: boolean; error?: string }> {
    try {
      const url = `${webBaseURL}/api/v1/users/web_profile_info/?username=instagram`
      const resp = await fetch(url, { headers: this.webHeaders() })
      if (resp.status === 200) return { ok: true }
      const body = await resp.text()
      if (body.includes('checkpoint_required')) return { ok: false, error: 'checkpoint_required' }
      if (body.includes('login_required')) return { ok: false, error: 'login_required' }
      return { ok: false, error: `HTTP ${resp.status}` }
    } catch (e) {
      return { ok: false, error: e instanceof Error ? e.message : String(e) }
    }
  }

  private async checkResponse(resp: Response): Promise<void> {
    if (resp.status === 429) throw new Error('rate limited')
    if (resp.status === 200) {
      // Verify we got JSON, not an HTML login page
      const ct = resp.headers.get('content-type') || ''
      if (ct.includes('text/html')) {
        throw new Error('Session error: Instagram returned HTML instead of JSON (likely requires re-login)')
      }
      return
    }
    const body = await resp.text()
    if (body.includes('checkpoint_required') || body.includes('challenge_required')) {
      throw new Error('Session locked: Instagram requires verification. Please re-login and update session cookies.')
    }
    if (body.includes('login_required')) {
      throw new Error('Session expired: Please re-login and update session cookies.')
    }
    if (resp.status === 404) throw new Error('not found')
    throw new Error(`HTTP ${resp.status}: ${body.slice(0, 300)}`)
  }

  // GraphQL query by query_hash (GET)
  async graphqlQuery(queryHash: string, variables: Record<string, unknown>): Promise<Record<string, unknown>> {
    const params = new URLSearchParams()
    params.set('query_hash', queryHash)
    params.set('variables', JSON.stringify(variables))
    const url = `${webBaseURL}/graphql/query/?${params.toString()}`
    const resp = await fetch(url, { headers: this.webHeaders() })
    await this.checkResponse(resp)
    return await resp.json() as Record<string, unknown>
  }

  // GraphQL POST with doc_id
  async graphqlPost(docId: string, variables: Record<string, unknown>): Promise<Record<string, unknown>> {
    const body = new URLSearchParams()
    body.set('doc_id', docId)
    body.set('variables', JSON.stringify(variables))
    body.set('server_timestamps', 'true')
    const headers = { ...this.webHeaders(), 'content-type': 'application/x-www-form-urlencoded' }
    const resp = await fetch(`${webBaseURL}/graphql/query/`, { method: 'POST', headers, body: body.toString() })
    await this.checkResponse(resp)
    return await resp.json() as Record<string, unknown>
  }

  // iPhone API for stories
  async getStories(userId: string): Promise<Record<string, unknown>> {
    const url = `${iPhoneBaseURL}/api/v1/feed/reels_media/?reel_ids=${userId}`
    const resp = await fetch(url, { headers: this.iphoneHeaders() })
    await this.checkResponse(resp)
    return await resp.json() as Record<string, unknown>
  }

  // iPhone API for highlights
  async getHighlightItems(highlightId: string): Promise<Record<string, unknown>> {
    const url = `${iPhoneBaseURL}/api/v1/feed/reels_media/?reel_ids=${encodeURIComponent(highlightId)}`
    const resp = await fetch(url, { headers: this.iphoneHeaders() })
    await this.checkResponse(resp)
    return await resp.json() as Record<string, unknown>
  }

  // Top search
  async search(query: string): Promise<Record<string, unknown>> {
    const params = new URLSearchParams({
      context: 'blended',
      query,
      include_reel: 'false',
      search_surface: 'web_top_search',
    })
    const url = `${webBaseURL}/web/search/topsearch/?${params.toString()}`
    const resp = await fetch(url, { headers: this.webHeaders() })
    await this.checkResponse(resp)
    return await resp.json() as Record<string, unknown>
  }

  // Post detail via __a=1 endpoint (same as Go client)
  async getPostDetail(shortcode: string): Promise<Record<string, unknown>> {
    const url = `${webBaseURL}/p/${shortcode}/?__a=1&__d=dis`
    const resp = await fetch(url, { headers: this.webHeaders(), redirect: 'manual' })
    // If redirected (likely to login page), session is invalid
    if (resp.status >= 300 && resp.status < 400) {
      throw new Error('Session expired: post detail endpoint redirected to login')
    }
    await this.checkResponse(resp)
    return await resp.json() as Record<string, unknown>
  }
}
