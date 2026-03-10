/**
 * Detect X/Twitter URLs and return redirect paths.
 * Supports: profile, tweet/status, article URLs.
 */
export function detectXURL(input: string): string | null {
  const s = input.trim()

  // Match x.com or twitter.com URLs
  const m = s.match(/^https?:\/\/(?:www\.)?(?:x\.com|twitter\.com)\/(.+)/i)
  if (!m) return null

  const path = m[1].replace(/\?.*$/, '').replace(/#.*$/, '') // strip query/hash

  // Article: /:username/article/:id
  const article = path.match(/^([^/]+)\/article\/(\d+)/)
  if (article) return `/${article[1]}/article/${article[2]}`

  // Tweet: /:username/status/:id
  const tweet = path.match(/^([^/]+)\/status\/(\d+)/)
  if (tweet) return `/${tweet[1]}/status/${tweet[2]}`

  // Profile: /:username (single path segment, not a reserved path)
  const reserved = new Set(['i', 'search', 'explore', 'notifications', 'messages', 'settings', 'home', 'compose', 'hashtag'])
  const profile = path.match(/^([A-Za-z0-9_]+)\/?$/)
  if (profile && !reserved.has(profile[1].toLowerCase())) return `/${profile[1]}`

  return null
}
