// Format large numbers: 1000 -> 1K, 1500000 -> 1.5M
export function fmtNum(n: number): string {
  if (n >= 1_000_000) {
    const m = n / 1_000_000
    return m >= 10 ? Math.floor(m) + 'M' : m.toFixed(1).replace(/\.0$/, '') + 'M'
  }
  if (n >= 1_000) {
    const k = n / 1_000
    return k >= 10 ? Math.floor(k) + 'K' : k.toFixed(1).replace(/\.0$/, '') + 'K'
  }
  return String(n)
}

// Relative time: "5s", "3m", "2h", "Jan 5", "Mar 12, 2024"
export function relTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  const now = new Date()
  const diff = Math.floor((now.getTime() - d.getTime()) / 1000)
  if (diff < 60) return diff + 's'
  if (diff < 3600) return Math.floor(diff / 60) + 'm'
  if (diff < 86400) return Math.floor(diff / 3600) + 'h'
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
  if (d.getFullYear() === now.getFullYear()) {
    return months[d.getMonth()] + ' ' + d.getDate()
  }
  return months[d.getMonth()] + ' ' + d.getDate() + ', ' + d.getFullYear()
}

// Full date: "3:45 PM · Jan 5, 2025"
export function fullDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
  let hours = d.getHours()
  const ampm = hours >= 12 ? 'PM' : 'AM'
  hours = hours % 12 || 12
  const mins = d.getMinutes().toString().padStart(2, '0')
  return `${hours}:${mins} ${ampm} · ${months[d.getMonth()]} ${d.getDate()}, ${d.getFullYear()}`
}

// Join date: "Joined March 2020"
export function joinDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  const months = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December']
  return `Joined ${months[d.getMonth()]} ${d.getFullYear()}`
}

// Parse tweet text into segments for rich rendering
export interface TextSegment {
  type: 'text' | 'mention' | 'hashtag' | 'url'
  text: string
  href?: string
}

export function parseText(text: string, tweetUrls: string[]): TextSegment[] {
  const segments: TextSegment[] = []
  // Regex for @mentions, #hashtags, and URLs
  const regex = /(@\w+)|(#\w+)|(https?:\/\/\S+)/g
  let lastIndex = 0
  let match: RegExpExecArray | null

  while ((match = regex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      segments.push({ type: 'text', text: text.slice(lastIndex, match.index) })
    }
    if (match[1]) {
      // @mention
      segments.push({ type: 'mention', text: match[1], href: match[1].slice(1) })
    } else if (match[2]) {
      // #hashtag
      segments.push({ type: 'hashtag', text: match[2], href: match[2].slice(1) })
    } else if (match[3]) {
      // URL - try to find expanded URL from tweet's urls array
      let displayUrl = match[3]
      for (const expanded of tweetUrls) {
        if (expanded && !expanded.includes('t.co')) {
          displayUrl = expanded
          break
        }
      }
      // Truncate display URL
      const display = displayUrl.replace(/^https?:\/\//, '').slice(0, 40)
      segments.push({ type: 'url', text: display + (displayUrl.length > 50 ? '...' : ''), href: displayUrl })
    }
    lastIndex = regex.lastIndex
  }

  if (lastIndex < text.length) {
    segments.push({ type: 'text', text: text.slice(lastIndex) })
  }

  return segments
}
