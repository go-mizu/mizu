import type { Profile, Tweet, TimelineResult } from './types'

const DEFAULT_NITTER_INSTANCES = [
  'https://nitter.net',
  'https://nitter.privacydev.net',
  'https://nitter.poast.org',
]

const NITTER_CURSOR_PREFIX = 'nitter:'

export type TweetConversation = {
  mainTweet: Tweet | null
  replies: Tweet[]
  cursor: string
}

function decodeHTML(s: string): string {
  return s
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&#x2F;/g, '/')
    .replace(/&#x27;/g, "'")
    .replace(/&#(\d+);/g, (_m, code) => {
      const n = parseInt(code, 10)
      return Number.isFinite(n) ? String.fromCharCode(n) : ''
    })
}

function stripTags(s: string): string {
  return decodeHTML(s.replace(/<br\s*\/?>/gi, '\n').replace(/<[^>]+>/g, '')).trim()
}

function parseCount(s: string): number {
  const clean = s.trim().toLowerCase().replace(/,/g, '')
  if (!clean) return 0
  const m = clean.match(/^(\d+(?:\.\d+)?)([km])?$/)
  if (!m) return parseInt(clean, 10) || 0
  const base = parseFloat(m[1])
  const unit = m[2]
  if (unit === 'k') return Math.round(base * 1_000)
  if (unit === 'm') return Math.round(base * 1_000_000)
  return Math.round(base)
}

function absoluteURL(base: string, value: string): string {
  if (!value) return ''
  try {
    return new URL(value, base).toString()
  } catch {
    return value
  }
}

function unique(items: string[]): string[] {
  return Array.from(new Set(items.filter(Boolean)))
}

function extractFirst(s: string, re: RegExp): string {
  const m = s.match(re)
  return m ? (m[1] || '') : ''
}

function extractBalancedDiv(html: string, start: number): string {
  const tagRe = /<\/?div\b[^>]*>/gi
  tagRe.lastIndex = start
  let depth = 0
  let seenRoot = false

  for (;;) {
    const m = tagRe.exec(html)
    if (!m) break
    if (!seenRoot) {
      seenRoot = true
      depth = 1
      continue
    }
    if (m[0].startsWith('</div')) depth--
    else depth++
    if (depth === 0) return html.slice(start, tagRe.lastIndex)
  }

  return ''
}

function extractDivBlocksByClass(html: string, className: string): string[] {
  const blocks: string[] = []
  const re = new RegExp(`<div[^>]+class="[^"]*\\b${className}\\b[^"]*"[^>]*>`, 'gi')
  for (;;) {
    const m = re.exec(html)
    if (!m) break
    const block = extractBalancedDiv(html, m.index)
    if (block) blocks.push(block)
  }
  return blocks
}

function parseTweetDate(block: string): string {
  const raw = extractFirst(block, /<span[^>]+class="[^"]*\btweet-date\b[^"]*"[\s\S]*?<a[^>]+title="([^"]+)"/i)
  if (!raw) return ''
  const d = new Date(decodeHTML(raw))
  return Number.isNaN(d.getTime()) ? '' : d.toISOString()
}

function parseTweetText(block: string): string {
  const m = block.match(/<(?:div|p)[^>]+class="[^"]*\btweet-content\b[^"]*"[^>]*>([\s\S]*?)<\/(?:div|p)>/i)
  return m ? stripTags(m[1]) : ''
}

function parseTweetUsername(block: string): string {
  let u = extractFirst(block, /href="\/([^/"'?]+)\/status\/\d+/i)
  if (!u) u = stripTags(extractFirst(block, /<a[^>]+class="[^"]*\busername\b[^"]*"[^>]*>([\s\S]*?)<\/a>/i))
  return u.replace(/^@/, '').trim()
}

function parseTweetName(block: string): string {
  const name = stripTags(extractFirst(block, /<a[^>]+class="[^"]*\bfullname\b[^"]*"[^>]*>([\s\S]*?)<\/a>/i))
  return name
}

function parseTweetID(block: string): string {
  return extractFirst(block, /href="\/[^/"'?]+\/status\/(\d+)/i)
}

function parseTweetAvatar(block: string, baseURL: string): string {
  return absoluteURL(baseURL, extractFirst(block, /<img[^>]+class="[^"]*\bavatar\b[^"]*"[^>]+src="([^"]+)"/i))
}

function parseTweetStats(block: string): { replies: number; retweets: number; likes: number } {
  const iconCount = (iconClass: string): number => {
    const raw = extractFirst(
      block,
      new RegExp(`${iconClass}[\\s\\S]{0,140}?(?:</span>|</i>|</div>)\\s*([0-9.,kKmM]+)`, 'i')
    )
    return parseCount(raw)
  }

  const repliesByIcon = iconCount('icon-comment')
  const retweetsByIcon = Math.max(iconCount('icon-retweet'), iconCount('icon-retweet-2'))
  const likesByIcon = iconCount('icon-heart')
  if (repliesByIcon || retweetsByIcon || likesByIcon) {
    return { replies: repliesByIcon, retweets: retweetsByIcon, likes: likesByIcon }
  }

  const text = stripTags(block)
  return {
    replies: parseCount((text.match(/\b(\d+(?:\.\d+)?[km]?)\s*repl(?:y|ies)\b/i) || [])[1] || ''),
    retweets: parseCount((text.match(/\b(\d+(?:\.\d+)?[km]?)\s*(?:retweet|repost)s?\b/i) || [])[1] || ''),
    likes: parseCount((text.match(/\b(\d+(?:\.\d+)?[km]?)\s*likes?\b/i) || [])[1] || ''),
  }
}

function parseTweetMedia(block: string, baseURL: string): { photos: string[]; videos: string[]; thumbs: string[]; gifs: string[] } {
  const photos: string[] = []
  const videos: string[] = []
  const thumbs: string[] = []
  const gifs: string[] = []

  const stillImageRe = /<a[^>]+class="[^"]*\bstill-image\b[^"]*"[^>]+href="([^"]+)"/gi
  for (;;) {
    const m = stillImageRe.exec(block)
    if (!m) break
    photos.push(absoluteURL(baseURL, m[1]))
  }

  const sourceRe = /<source[^>]+src="([^"]+)"/gi
  for (;;) {
    const m = sourceRe.exec(block)
    if (!m) break
    const url = absoluteURL(baseURL, m[1])
    if (/\.(mp4|m3u8)(\?|$)/i.test(url)) videos.push(url)
  }

  const posterRe = /<video[^>]+poster="([^"]+)"/gi
  for (;;) {
    const m = posterRe.exec(block)
    if (!m) break
    thumbs.push(absoluteURL(baseURL, m[1]))
  }

  const gifRe = /<video[^>]+class="[^"]*\bgif\b[^"]*"[^>]*>[\s\S]*?<source[^>]+src="([^"]+)"/gi
  for (;;) {
    const m = gifRe.exec(block)
    if (!m) break
    gifs.push(absoluteURL(baseURL, m[1]))
  }

  return { photos: unique(photos), videos: unique(videos), thumbs: unique(thumbs), gifs: unique(gifs) }
}

function buildTweetFromBlock(block: string, baseURL: string): Tweet | null {
  const id = parseTweetID(block)
  if (!id) return null
  const username = parseTweetUsername(block)
  const name = parseTweetName(block) || username
  const text = parseTweetText(block)
  const media = parseTweetMedia(block, baseURL)
  const stats = parseTweetStats(block)

  return {
    id,
    conversationID: id,
    text,
    username,
    userID: '',
    name,
    avatar: parseTweetAvatar(block, baseURL),
    permanentURL: username ? `https://x.com/${username}/status/${id}` : '',
    isRetweet: /retweet|repost/i.test(stripTags(extractFirst(block, /<div[^>]+class="[^"]*\bretweet-header\b[^"]*"[^>]*>([\s\S]*?)<\/div>/i))),
    isReply: /replying to/i.test(stripTags(block)),
    isQuote: false,
    isPin: /pinned/i.test(stripTags(extractFirst(block, /<div[^>]+class="[^"]*\bpinned\b[^"]*"[^>]*>([\s\S]*?)<\/div>/i))),
    replyToID: '',
    replyToUser: '',
    quotedID: '',
    retweetedID: '',
    retweetedTweet: undefined,
    quotedTweet: undefined,
    likes: stats.likes,
    retweets: stats.retweets,
    replies: stats.replies,
    views: 0,
    bookmarks: 0,
    quotes: 0,
    photos: media.photos,
    videos: media.videos,
    videoThumbnails: media.thumbs,
    gifs: media.gifs,
    hashtags: unique((text.match(/(^|\s)#([A-Za-z0-9_]+)/g) || []).map((v) => v.trim().slice(1))),
    mentions: unique((text.match(/(^|\s)@([A-Za-z0-9_]+)/g) || []).map((v) => v.trim())),
    urls: unique(Array.from(text.matchAll(/https?:\/\/\S+/g)).map((m) => m[0])),
    sensitive: false,
    language: '',
    source: 'Nitter',
    place: '',
    isEdited: false,
    isBlueVerified: false,
    verifiedType: '',
    postedAt: parseTweetDate(block),
  }
}

export function isNitterCursor(cursor: string): boolean {
  return cursor.startsWith(NITTER_CURSOR_PREFIX)
}

export function encodeNitterCursor(pathAndQuery: string): string {
  return `${NITTER_CURSOR_PREFIX}${encodeURIComponent(pathAndQuery)}`
}

export function decodeNitterCursor(cursor: string): string {
  if (!isNitterCursor(cursor)) return ''
  const v = cursor.slice(NITTER_CURSOR_PREFIX.length)
  try {
    return decodeURIComponent(v)
  } catch {
    return ''
  }
}

function showMoreCursor(html: string, pageURL: string): string {
  const href =
    extractFirst(html, /<a[^>]+class="[^"]*\bshow-more\b[^"]*"[^>]+href="([^"]+)"/i) ||
    extractFirst(html, /<div[^>]+class="[^"]*\bshow-more\b[^"]*"[\s\S]*?<a[^>]+href="([^"]+)"/i)
  if (!href) return ''
  try {
    const u = new URL(href, pageURL)
    return encodeNitterCursor(u.pathname + u.search)
  } catch {
    return ''
  }
}

function parseCanonicalUsername(html: string): string {
  return extractFirst(html, /<link[^>]+rel="canonical"[^>]+href="https?:\/\/[^/]+\/([^/"'?]+)\/status\/\d+[^"]*"[^>]*>/i)
}

function parseProfileJoined(text: string): string {
  if (!text) return ''
  const asDate = new Date(text)
  if (!Number.isNaN(asDate.getTime())) return asDate.toISOString()
  const withDay = new Date(`1 ${text}`)
  if (!Number.isNaN(withDay.getTime())) return withDay.toISOString()
  return ''
}

function parseProfileCardBlock(html: string): string {
  const blocks = extractDivBlocksByClass(html, 'profile-card')
  return blocks[0] || ''
}

export function parseNitterProfileHTML(html: string, pageURL: string, fallbackUsername = ''): Profile | null {
  const u = new URL(pageURL)
  const baseURL = `${u.protocol}//${u.host}`
  const card = parseProfileCardBlock(html) || html

  let username =
    extractFirst(card, /<a[^>]+class="[^"]*\bprofile-card-username\b[^"]*"[^>]*>@?([^<]+)<\/a>/i) ||
    extractFirst(card, /<a[^>]+class="[^"]*\busername\b[^"]*"[^>]*>@?([^<]+)<\/a>/i) ||
    extractFirst(card, /<link[^>]+rel="canonical"[^>]+href="https?:\/\/[^/]+\/([^/"'?]+)\/?/i) ||
    fallbackUsername
  username = stripTags(username).replace(/^@/, '').trim()

  const name = stripTags(
    extractFirst(card, /<a[^>]+class="[^"]*\bprofile-card-fullname\b[^"]*"[^>]*>([\s\S]*?)<\/a>/i) ||
    extractFirst(card, /<a[^>]+class="[^"]*\bfullname\b[^"]*"[^>]*>([\s\S]*?)<\/a>/i)
  )
  const biography = stripTags(
    extractFirst(card, /<div[^>]+class="[^"]*\bprofile-bio\b[^"]*"[^>]*>([\s\S]*?)<\/div>/i) ||
    extractFirst(card, /<p[^>]+class="[^"]*\bprofile-bio\b[^"]*"[^>]*>([\s\S]*?)<\/p>/i)
  )
  const avatar = absoluteURL(
    baseURL,
    extractFirst(card, /<img[^>]+class="[^"]*\bprofile-card-avatar\b[^"]*"[^>]+src="([^"]+)"/i) ||
    extractFirst(card, /<img[^>]+class="[^"]*\bavatar\b[^"]*"[^>]+src="([^"]+)"/i)
  )
  const banner = absoluteURL(
    baseURL,
    extractFirst(html, /<img[^>]+class="[^"]*\bprofile-banner\b[^"]*"[^>]+src="([^"]+)"/i)
  )
  const location = stripTags(extractFirst(card, /<div[^>]+class="[^"]*\bprofile-location\b[^"]*"[^>]*>([\s\S]*?)<\/div>/i))
  const website = absoluteURL(
    baseURL,
    extractFirst(card, /<div[^>]+class="[^"]*\bprofile-website\b[^"]*"[\s\S]*?<a[^>]+href="([^"]+)"/i)
  )
  const joinedRaw = stripTags(
    extractFirst(card, /<div[^>]+class="[^"]*\bprofile-joindate\b[^"]*"[^>]*>([\s\S]*?)<\/div>/i) ||
    extractFirst(card, /Joined\s+([^<\n]+)/i)
  ).replace(/^Joined\s+/i, '')
  const joined = parseProfileJoined(joinedRaw)

  const followingCount = parseCount(
    stripTags(extractFirst(html, /<a[^>]+href="\/[^/"'?]+\/following"[^>]*>([\s\S]*?)<\/a>/i)).match(/[0-9][0-9.,kKmM]*/)?.[0] || ''
  )
  const followersCount = parseCount(
    stripTags(extractFirst(html, /<a[^>]+href="\/[^/"'?]+\/followers"[^>]*>([\s\S]*?)<\/a>/i)).match(/[0-9][0-9.,kKmM]*/)?.[0] || ''
  )
  const tweetsCount = parseCount(
    (stripTags(extractFirst(html, /<ul[^>]+class="[^"]*\bprofile-statlist\b[^"]*"[^>]*>([\s\S]*?)<\/ul>/i)).match(/([0-9][0-9.,kKmM]*)\s*(?:tweets|posts)/i) || [])[1] || ''
  )

  if (!username && !name) return null

  return {
    id: '',
    username,
    name: name || username,
    biography,
    avatar,
    banner,
    location,
    website,
    url: website,
    joined,
    birthday: '',
    followersCount,
    followingCount,
    tweetsCount,
    likesCount: 0,
    mediaCount: 0,
    listedCount: 0,
    isPrivate: false,
    isVerified: false,
    isBlueVerified: false,
    verifiedType: '',
    pinnedTweetIDs: [],
    professionalType: '',
    professionalCategory: '',
  }
}

export function parseNitterTimelineHTML(html: string, pageURL: string): TimelineResult {
  const u = new URL(pageURL)
  const baseURL = `${u.protocol}//${u.host}`
  const blocks = extractDivBlocksByClass(html, 'timeline-item')
  const tweets: Tweet[] = []
  const seen = new Set<string>()
  for (const block of blocks) {
    const t = buildTweetFromBlock(block, baseURL)
    if (!t || !t.id || seen.has(t.id)) continue
    seen.add(t.id)
    tweets.push(t)
  }
  return { tweets, cursor: showMoreCursor(html, pageURL) }
}

function parseProfileFromUserBlock(block: string, baseURL: string): Profile | null {
  let username =
    stripTags(extractFirst(block, /<a[^>]+class="[^"]*\busername\b[^"]*"[^>]*>([\s\S]*?)<\/a>/i)).replace(/^@/, '') ||
    extractFirst(block, /href="\/([^/"'?]+)"(?![^>]*\/status\/)/i)
  username = username.trim()
  const name = stripTags(extractFirst(block, /<a[^>]+class="[^"]*\bfullname\b[^"]*"[^>]*>([\s\S]*?)<\/a>/i))
  if (!username && !name) return null

  const biography = stripTags(
    extractFirst(block, /<div[^>]+class="[^"]*\bprofile-bio\b[^"]*"[^>]*>([\s\S]*?)<\/div>/i) ||
    extractFirst(block, /<(?:div|p)[^>]+class="[^"]*\btweet-content\b[^"]*"[^>]*>([\s\S]*?)<\/(?:div|p)>/i)
  )
  const avatar = absoluteURL(baseURL, extractFirst(block, /<img[^>]+class="[^"]*\bavatar\b[^"]*"[^>]+src="([^"]+)"/i))

  return {
    id: '',
    username,
    name: name || username,
    biography,
    avatar,
    banner: '',
    location: '',
    website: '',
    url: '',
    joined: '',
    birthday: '',
    followersCount: 0,
    followingCount: 0,
    tweetsCount: 0,
    likesCount: 0,
    mediaCount: 0,
    listedCount: 0,
    isPrivate: false,
    isVerified: false,
    isBlueVerified: false,
    verifiedType: '',
    pinnedTweetIDs: [],
    professionalType: '',
    professionalCategory: '',
  }
}

export function parseNitterSearchUsersHTML(html: string, pageURL: string): { users: Profile[]; cursor: string } {
  const u = new URL(pageURL)
  const baseURL = `${u.protocol}//${u.host}`
  const blocks = [
    ...extractDivBlocksByClass(html, 'timeline-item'),
    ...extractDivBlocksByClass(html, 'user-card'),
  ]
  const users: Profile[] = []
  const seen = new Set<string>()
  for (const block of blocks) {
    const profile = parseProfileFromUserBlock(block, baseURL)
    if (!profile) continue
    if (profile.username && seen.has(profile.username.toLowerCase())) continue
    if (profile.username) seen.add(profile.username.toLowerCase())
    users.push(profile)
  }
  return { users, cursor: showMoreCursor(html, pageURL) }
}

export function parseNitterTweetHTML(html: string, tweetID: string, pageURL: string): TweetConversation {
  const u = new URL(pageURL)
  const baseURL = `${u.protocol}//${u.host}`
  const mainBlocks = extractDivBlocksByClass(html, 'main-tweet')
  const mainBlock = mainBlocks[0] || ''
  const fallbackUsername = parseCanonicalUsername(html)
  const parsedMain = mainBlock ? buildTweetFromBlock(mainBlock, baseURL) : null
  const mainTweet = parsedMain
    ? {
        ...parsedMain,
        id: parsedMain.id || tweetID,
        username: parsedMain.username || fallbackUsername,
        name: parsedMain.name || fallbackUsername,
      }
    : null
  return { mainTweet, replies: [], cursor: showMoreCursor(html, pageURL) }
}

function parseInstances(instancesCSV?: string): string[] {
  const fromEnv = (instancesCSV || '')
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
  return fromEnv.length > 0 ? fromEnv : DEFAULT_NITTER_INSTANCES
}

function normalizePath(pathOrURL: string): string {
  if (!pathOrURL) return '/'
  if (pathOrURL.startsWith('/')) return pathOrURL
  try {
    const u = new URL(pathOrURL)
    return u.pathname + u.search
  } catch {
    return '/' + pathOrURL.replace(/^\/+/, '')
  }
}

function cursorPath(cursor: string): string {
  const decoded = decodeNitterCursor(cursor)
  return decoded ? normalizePath(decoded) : ''
}

function userTimelinePath(username: string, tab: string): string {
  if (tab === 'replies') return `/${encodeURIComponent(username)}/with_replies`
  if (tab === 'media') return `/${encodeURIComponent(username)}/media`
  return `/${encodeURIComponent(username)}`
}

function searchTweetsPath(query: string, mode: string): string {
  const f = (mode === 'Media' || mode === 'Photos') ? 'media' : 'tweets'
  return `/search?f=${encodeURIComponent(f)}&q=${encodeURIComponent(query)}`
}

function searchUsersPath(query: string): string {
  return `/search?f=users&q=${encodeURIComponent(query)}`
}

async function fetchFromInstances<T>(
  instancesCSV: string | undefined,
  defaultPath: string,
  cursor: string,
  parse: (html: string, pageURL: string) => T,
  isValid: (parsed: T) => boolean,
  errorPrefix: string
): Promise<T> {
  const instances = parseInstances(instancesCSV)
  const forcedPath = cursorPath(cursor)
  const path = forcedPath || normalizePath(defaultPath)
  let lastError = ''

  for (const base of instances) {
    const root = base.replace(/\/+$/, '')
    const url = `${root}${path.startsWith('/') ? path : '/' + path}`
    try {
      const resp = await fetch(url, {
        headers: {
          accept: 'text/html,application/xhtml+xml',
          'user-agent': 'Mozilla/5.0 (compatible; x-viewer/1.0; +https://workers.dev)',
        },
        redirect: 'follow',
      })

      if (resp.status === 404) continue
      if (resp.status === 429) {
        lastError = `${errorPrefix}: nitter rate limited on ${base}`
        continue
      }
      if (!resp.ok) {
        lastError = `${errorPrefix}: nitter status ${resp.status} on ${base}`
        continue
      }

      const html = await resp.text()
      const parsed = parse(html, resp.url || url)
      if (isValid(parsed)) return parsed
      lastError = `${errorPrefix}: parsed result missing required data on ${base}`
    } catch (e) {
      lastError = `${errorPrefix}: ${e instanceof Error ? e.message : String(e)}`
    }
  }

  throw new Error(lastError || `${errorPrefix}: nitter fallback failed`)
}

export async function fetchTweetFromNitter(tweetID: string, instancesCSV?: string, cursor = ''): Promise<TweetConversation> {
  return fetchFromInstances(
    instancesCSV,
    `/i/status/${encodeURIComponent(tweetID)}`,
    cursor,
    (html, pageURL) => parseNitterTweetHTML(html, tweetID, pageURL),
    (parsed) => Boolean(parsed.mainTweet),
    'tweet'
  )
}

export async function fetchProfileFromNitter(username: string, instancesCSV?: string): Promise<Profile | null> {
  return fetchFromInstances(
    instancesCSV,
    `/${encodeURIComponent(username)}`,
    '',
    (html, pageURL) => parseNitterProfileHTML(html, pageURL, username),
    (parsed) => Boolean(parsed),
    'profile'
  )
}

export async function fetchUserTweetsFromNitter(
  username: string,
  tab = 'tweets',
  cursor = '',
  instancesCSV?: string
): Promise<TimelineResult> {
  return fetchFromInstances(
    instancesCSV,
    userTimelinePath(username, tab),
    cursor,
    (html, pageURL) => parseNitterTimelineHTML(html, pageURL),
    () => true,
    'user tweets'
  )
}

export async function fetchSearchTweetsFromNitter(
  query: string,
  mode: string,
  cursor = '',
  instancesCSV?: string
): Promise<TimelineResult> {
  return fetchFromInstances(
    instancesCSV,
    searchTweetsPath(query, mode),
    cursor,
    (html, pageURL) => parseNitterTimelineHTML(html, pageURL),
    () => true,
    'search tweets'
  )
}

export async function fetchSearchUsersFromNitter(
  query: string,
  cursor = '',
  instancesCSV?: string
): Promise<{ users: Profile[]; cursor: string }> {
  return fetchFromInstances(
    instancesCSV,
    searchUsersPath(query),
    cursor,
    (html, pageURL) => parseNitterSearchUsersHTML(html, pageURL),
    () => true,
    'search users'
  )
}
