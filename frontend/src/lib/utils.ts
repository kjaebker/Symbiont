import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

const DAY = 86400

/** "3 days", "2 weeks", "5 months", "2 years": a span of whole days in words. */
function daysInWords(days: number): string {
  if (days < 14) return `${days} day${days === 1 ? '' : 's'}`
  if (days < 60) return `${Math.floor(days / 7)} weeks`
  if (days < 730) return `${Math.floor(days / 30)} months`
  return `${Math.floor(days / 365)} years`
}

/** Compact age of a timestamp: "just now", "5m ago", "3h ago", "6d ago", "3w ago", "5mo ago", "2y ago". */
export function relativeTime(ts: string): string {
  const now = Date.now()
  const then = new Date(ts).getTime()
  const diff = Math.floor((now - then) / 1000)

  if (diff < 5) return 'just now'
  if (diff < 60) return `${diff}s ago`
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
  if (diff < DAY) return `${Math.floor(diff / 3600)}h ago`
  const days = Math.floor(diff / DAY)
  if (days < 14) return `${days}d ago`
  if (days < 60) return `${Math.floor(days / 7)}w ago`
  if (days < 730) return `${Math.floor(days / 30)}mo ago`
  return `${Math.floor(days / 365)}y ago`
}

/** Full local date and time, for tooltips next to a relative time. */
export function absoluteTime(ts: string): string {
  return new Date(ts).toLocaleString(undefined, {
    year: 'numeric', month: 'short', day: 'numeric', hour: 'numeric', minute: '2-digit',
  })
}

/** Whole days since a timestamp (0 for today or the future). */
export function daysSince(ts: string): number {
  return Math.max(0, Math.floor((Date.now() - new Date(ts).getTime()) / (DAY * 1000)))
}

/** How long ago a due date passed, in words ("5 months"); '' if under a day. */
export function overdueBy(ts: string): string {
  const days = daysSince(ts)
  return days < 1 ? '' : daysInWords(days)
}

export function splitCamelCase(s: string): string {
  return s.replace(/([a-z])([A-Z])/g, '$1 $2')
}

export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}
