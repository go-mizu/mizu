const MONTHS = [
  "Jan", "Feb", "Mar", "Apr", "May", "Jun",
  "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
]

export function fmtNum(n: number): string {
  if (n >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(1)}B`
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`
  return String(n)
}

export function fmtBytes(n: number): string {
  if (n === 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB", "PB"]
  const i = Math.floor(Math.log(n) / Math.log(1024))
  const val = n / Math.pow(1024, i)
  return `${val.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

export function fmtDate(ts: string): string {
  if (!ts || ts.length < 8) return ts || ""
  const y = ts.substring(0, 4)
  const m = parseInt(ts.substring(4, 6)) - 1
  const d = parseInt(ts.substring(6, 8))
  return `${MONTHS[m]} ${d}, ${y}`
}

export function fmtTimestamp(ts: string): string {
  if (!ts || ts.length < 14) return ts || ""
  const y = ts.substring(0, 4)
  const m = ts.substring(4, 6)
  const d = ts.substring(6, 8)
  const h = ts.substring(8, 10)
  const min = ts.substring(10, 12)
  const s = ts.substring(12, 14)
  return `${y}-${m}-${d} ${h}:${min}:${s}`
}

export function truncURL(url: string, max = 80): string {
  if (url.length <= max) return url
  return url.substring(0, max - 1) + "\u2026"
}

export function crawlToDate(crawlID: string): string {
  const match = crawlID.match(/CC-MAIN-(\d{4})-(\d{2})/)
  if (!match) return crawlID
  const year = match[1]
  const week = parseInt(match[2])
  const monthIdx = Math.min(Math.floor((week - 1) / 4.33), 11)
  return `${MONTHS[monthIdx]} ${year}`
}
