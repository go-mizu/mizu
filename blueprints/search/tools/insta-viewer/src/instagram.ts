import { webBaseURL, iPhoneBaseURL, appID, webUserAgent, iPhoneUserAgent } from './config'

export class InstagramClient {
  private sessionId: string
  private csrfToken: string
  private dsUserId: string
  private mid: string
  private igDid: string

  static isSessionError(e: unknown): boolean {
    const msg = e instanceof Error ? e.message : String(e)
    return msg.includes('checkpoint') || msg.includes('login_required') ||
      msg.includes('Session locked') || msg.includes('Session expired') ||
      msg.includes('Session error') || msg.includes('requires re-login') ||
      msg.includes('requires verification')
  }

  constructor(sessionId: string, csrfToken: string, dsUserId: string, mid: string, igDid: string) {
    this.sessionId = sessionId
    this.csrfToken = csrfToken
    this.dsUserId = dsUserId
    this.mid = mid || ''
    this.igDid = igDid || ''
  }

  private hasSession(): boolean {
    return !!this.sessionId && !!this.csrfToken
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

  // Headers without session cookies — for public endpoints
  private publicHeaders(): Record<string, string> {
    return {
      accept: '*/*',
      'accept-language': 'en-US,en;q=0.8',
      'user-agent': webUserAgent,
      'x-ig-app-id': appID,
      'x-requested-with': 'XMLHttpRequest',
      origin: 'https://www.instagram.com',
      referer: 'https://www.instagram.com/',
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

  // Fetch profile via web_profile_info — works with or without session
  async getProfileInfo(username: string): Promise<Record<string, unknown>> {
    const url = `${webBaseURL}/api/v1/users/web_profile_info/?username=${encodeURIComponent(username)}`
    // Try with session first
    if (this.hasSession()) {
      try {
        const resp = await fetch(url, { headers: this.webHeaders() })
        if (resp.status === 200) {
          const ct = resp.headers.get('content-type') || ''
          if (!ct.includes('text/html')) return await resp.json() as Record<string, unknown>
        }
      } catch { /* fall through to public */ }
    }
    // Fallback: public request without session (works for public profiles)
    const resp = await fetch(url, { headers: this.publicHeaders() })
    await this.checkResponse(resp)
    return await resp.json() as Record<string, unknown>
  }

  // Check if the session is valid
  async checkSession(): Promise<{ ok: boolean; error?: string }> {
    try {
      const url = `${webBaseURL}/api/v1/users/web_profile_info/?username=instagram`
      const resp = await fetch(url, { headers: this.hasSession() ? this.webHeaders() : this.publicHeaders() })
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

  // GraphQL query by query_hash (GET) — tries with session, falls back to public
  async graphqlQuery(queryHash: string, variables: Record<string, unknown>): Promise<Record<string, unknown>> {
    const params = new URLSearchParams()
    params.set('query_hash', queryHash)
    params.set('variables', JSON.stringify(variables))
    const url = `${webBaseURL}/graphql/query/?${params.toString()}`
    if (this.hasSession()) {
      try {
        const resp = await fetch(url, { headers: this.webHeaders() })
        if (resp.status === 200) {
          const ct = resp.headers.get('content-type') || ''
          if (!ct.includes('text/html')) return await resp.json() as Record<string, unknown>
        }
      } catch { /* fall through */ }
    }
    // Public fallback
    const resp = await fetch(url, { headers: this.publicHeaders() })
    await this.checkResponse(resp)
    return await resp.json() as Record<string, unknown>
  }

  // GraphQL POST with doc_id — tries with session, falls back to public
  async graphqlPost(docId: string, variables: Record<string, unknown>): Promise<Record<string, unknown>> {
    const body = new URLSearchParams()
    body.set('doc_id', docId)
    body.set('variables', JSON.stringify(variables))
    body.set('server_timestamps', 'true')
    if (this.hasSession()) {
      try {
        const headers = { ...this.webHeaders(), 'content-type': 'application/x-www-form-urlencoded' }
        const resp = await fetch(`${webBaseURL}/graphql/query/`, { method: 'POST', headers, body: body.toString() })
        if (resp.status === 200) {
          const ct = resp.headers.get('content-type') || ''
          if (!ct.includes('text/html')) return await resp.json() as Record<string, unknown>
        }
      } catch { /* fall through */ }
    }
    // Public fallback
    const headers = { ...this.publicHeaders(), 'content-type': 'application/x-www-form-urlencoded' }
    const resp = await fetch(`${webBaseURL}/graphql/query/`, { method: 'POST', headers, body: body.toString() })
    await this.checkResponse(resp)
    return await resp.json() as Record<string, unknown>
  }

  // iPhone API for stories — requires session
  async getStories(userId: string): Promise<Record<string, unknown>> {
    const url = `${iPhoneBaseURL}/api/v1/feed/reels_media/?reel_ids=${userId}`
    const resp = await fetch(url, { headers: this.iphoneHeaders() })
    await this.checkResponse(resp)
    return await resp.json() as Record<string, unknown>
  }

  // iPhone API for highlights — requires session
  async getHighlightItems(highlightId: string): Promise<Record<string, unknown>> {
    const url = `${iPhoneBaseURL}/api/v1/feed/reels_media/?reel_ids=${encodeURIComponent(highlightId)}`
    const resp = await fetch(url, { headers: this.iphoneHeaders() })
    await this.checkResponse(resp)
    return await resp.json() as Record<string, unknown>
  }

  // Top search — works without session
  async search(query: string): Promise<Record<string, unknown>> {
    const params = new URLSearchParams({
      context: 'blended',
      query,
      include_reel: 'false',
      search_surface: 'web_top_search',
    })
    const url = `${webBaseURL}/web/search/topsearch/?${params.toString()}`
    if (this.hasSession()) {
      try {
        const resp = await fetch(url, { headers: this.webHeaders() })
        if (resp.status === 200) {
          const ct = resp.headers.get('content-type') || ''
          if (!ct.includes('text/html')) return await resp.json() as Record<string, unknown>
        }
      } catch { /* fall through */ }
    }
    const resp = await fetch(url, { headers: this.publicHeaders() })
    await this.checkResponse(resp)
    return await resp.json() as Record<string, unknown>
  }

  // Post detail — tries: 1) doc_id GraphQL  2) __a=1  3) embed page parsing
  async getPostDetail(shortcode: string): Promise<Record<string, unknown>> {
    // Strategy 1: GraphQL doc_id (most reliable with session)
    if (this.hasSession()) {
      try {
        const { docIdPostDetail } = await import('./config')
        const data = await this.graphqlPost(docIdPostDetail, { shortcode })
        if (data?.data) return data
      } catch { /* fall through */ }
    }
    // Strategy 2: __a=1 endpoint
    try {
      const url = `${webBaseURL}/p/${shortcode}/?__a=1&__d=dis`
      const resp = await fetch(url, { headers: this.hasSession() ? this.webHeaders() : this.publicHeaders(), redirect: 'manual' })
      if (resp.status === 200) {
        const ct = resp.headers.get('content-type') || ''
        if (!ct.includes('text/html')) {
          const data = await resp.json() as Record<string, unknown>
          if (data) return data
        }
      }
    } catch { /* fall through */ }
    // Strategy 3: Embed page (no auth, different rate limit pool)
    return this.getPostFromEmbed(shortcode)
  }

  // Parse post data from Instagram's embed page — completely public, no auth
  // Uses a simplified User-Agent — Instagram blocks detailed UAs from datacenter IPs
  private async getPostFromEmbed(shortcode: string): Promise<Record<string, unknown>> {
    const url = `${webBaseURL}/p/${shortcode}/embed/captioned/`
    const resp = await fetch(url, {
      headers: {
        'user-agent': 'Mozilla/5.0 (compatible)',
        accept: 'text/html',
        'accept-language': 'en-US,en;q=0.8',
      },
    })
    if (resp.status !== 200) throw new Error(`Embed page returned HTTP ${resp.status}`)
    const html = await resp.text()

    // Strategy A: Extract shortcode_media JSON from embed script (carousel/sidecar posts)
    const marker = '"shortcode_media":'
    const idx = html.indexOf(marker)
    if (idx !== -1) {
      const start = idx + marker.length
      let depth = 0
      let end = start
      for (let i = start; i < html.length; i++) {
        const ch = html[i]
        if (ch === '{') depth++
        else if (ch === '}') {
          depth--
          if (depth === 0) { end = i + 1; break }
        }
      }
      if (depth === 0) {
        let jsonStr = html.slice(start, end)
        jsonStr = jsonStr.replace(/\\"/g, '"').replace(/\\\//g, '/').replace(/\\\\/g, '\\')
        try {
          const media = JSON.parse(jsonStr)
          return { graphql: { shortcode_media: media } }
        } catch { /* fall through to HTML parsing */ }
      }
    }

    // Strategy B: Parse HTML elements for single-image/video posts
    return this.parseEmbedHTML(html, shortcode)
  }

  // Extract post data from embed HTML elements (for posts without shortcode_media JSON)
  private parseEmbedHTML(html: string, shortcode: string): Record<string, unknown> {
    const unescape = (s: string) => s.replace(/&amp;/g, '&').replace(/&lt;/g, '<').replace(/&gt;/g, '>').replace(/&quot;/g, '"').replace(/&#0?39;/g, "'").replace(/&#064;/g, '@')

    // Image URL from EmbeddedMediaImage <img> tag
    const imgMatch = html.match(/<img[^>]*class="EmbeddedMediaImage"[^>]*src="([^"]+)"/)
      || html.match(/class="EmbeddedMediaImage"[^>]*src="([^"]+)"/)
    const displayUrl = imgMatch ? unescape(imgMatch[1]) : ''

    // Video URL
    const videoMatch = html.match(/<video[^>]*src="([^"]+)"/)
    const videoUrl = videoMatch ? unescape(videoMatch[1]) : ''
    const isVideo = !!videoUrl || html.includes('EmbeddedMediaVideo')

    // Username from CaptionUsername <a> tag
    const userMatch = html.match(/class="CaptionUsername"[^>]*>([^<]+)</)
    const username = userMatch ? userMatch[1].trim() : ''

    // Profile pic: <div class="HoverCardProfile"><img src="..." />
    const picMatch = html.match(/class="HoverCardProfile"><img[^>]*src="([^"]+)"/)
    const profilePic = picMatch ? unescape(picMatch[1]) : ''

    // Like count: <div class="SocialProof"><a ...>207,804 likes</a>
    const likeMatch = html.match(/class="SocialProof">.*?([\d,]+)\s*likes?/i)
    let likeCount = 0
    if (likeMatch) likeCount = parseInt(likeMatch[1].replace(/,/g, ''), 10) || 0

    // Comment count: "View all 717 comments"
    const commentMatch = html.match(/View all ([\d,]+) comments/i)
    let commentCount = 0
    if (commentMatch) commentCount = parseInt(commentMatch[1].replace(/,/g, ''), 10) || 0

    // Caption: extract text after CaptionUsername up to CaptionComments
    let caption = ''
    const captionBlock = html.match(/class="CaptionUsername"[^>]*>[^<]*<\/a>([\s\S]*?)(?:<div class="CaptionComments"|<\/div>\s*<div class="SocialProof")/)
    if (captionBlock) {
      caption = captionBlock[1]
        .replace(/<br\s*\/?>/gi, '\n')
        .replace(/<[^>]+>/g, '')
        .trim()
      caption = unescape(caption)
    }

    // Media ID from data-ios-link="media?id=..."
    const idMatch = html.match(/data-ios-link="[^"]*media\?id=(\d+)"/)
    const mediaId = idMatch ? idMatch[1] : ''

    if (!displayUrl && !videoUrl) throw new Error('not found')

    const media: Record<string, unknown> = {
      id: mediaId,
      shortcode,
      __typename: isVideo ? 'GraphVideo' : 'GraphImage',
      display_url: displayUrl,
      is_video: isVideo,
      edge_media_to_caption: { edges: caption ? [{ node: { text: caption } }] : [] },
      edge_media_preview_like: { count: likeCount },
      edge_media_to_comment: { count: commentCount },
      owner: { username, profile_pic_url: profilePic },
      dimensions: { width: 1080, height: 1080 },
    }
    if (videoUrl) media.video_url = videoUrl

    return { graphql: { shortcode_media: media } }
  }
}
