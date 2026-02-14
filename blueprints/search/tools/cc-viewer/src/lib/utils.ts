import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B"
  const units = ["B", "KB", "MB", "GB", "TB", "PB"]
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const val = bytes / Math.pow(1024, i)
  return `${val.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

export function formatNumber(num: number): string {
  if (num >= 1_000_000_000) return `${(num / 1_000_000_000).toFixed(1)}B`
  if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(1)}M`
  if (num >= 1_000) return `${(num / 1_000).toFixed(1)}K`
  return num.toLocaleString()
}

export function formatTimestamp(ts: string): string {
  if (!ts || ts.length < 14) return ts
  const y = ts.substring(0, 4)
  const m = ts.substring(4, 6)
  const d = ts.substring(6, 8)
  const h = ts.substring(8, 10)
  const min = ts.substring(10, 12)
  const s = ts.substring(12, 14)
  return `${y}-${m}-${d} ${h}:${min}:${s}`
}

export function crawlToDate(crawlID: string): string {
  const match = crawlID.match(/CC-MAIN-(\d{4})-(\d{2})/)
  if (!match) return crawlID
  const year = match[1]
  const week = parseInt(match[2])
  const months = [
    "Jan", "Feb", "Mar", "Apr", "May", "Jun",
    "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
  ]
  const monthIdx = Math.min(Math.floor((week - 1) / 4.33), 11)
  return `${months[monthIdx]} ${year}`
}

export function statusColor(status: string): "success" | "warning" | "error" {
  const code = parseInt(status)
  if (code >= 200 && code < 300) return "success"
  if (code >= 300 && code < 400) return "warning"
  return "error"
}
