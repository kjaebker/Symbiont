import { describe, it, expect } from 'vitest'
import { relativeTime, formatBytes, cn, splitCamelCase } from '../utils'

describe('relativeTime', () => {
  it('returns "just now" for timestamps within 5 seconds', () => {
    const now = new Date().toISOString()
    expect(relativeTime(now)).toBe('just now')
  })

  it('returns seconds ago for recent timestamps', () => {
    const ts = new Date(Date.now() - 30_000).toISOString()
    expect(relativeTime(ts)).toBe('30s ago')
  })

  it('returns minutes ago', () => {
    const ts = new Date(Date.now() - 5 * 60_000).toISOString()
    expect(relativeTime(ts)).toBe('5m ago')
  })

  it('returns hours ago', () => {
    const ts = new Date(Date.now() - 3 * 3600_000).toISOString()
    expect(relativeTime(ts)).toBe('3h ago')
  })

  it('returns days ago', () => {
    const ts = new Date(Date.now() - 2 * 86400_000).toISOString()
    expect(relativeTime(ts)).toBe('2d ago')
  })
})

describe('formatBytes', () => {
  it('handles 0 bytes', () => {
    expect(formatBytes(0)).toBe('0 B')
  })

  it('formats bytes', () => {
    expect(formatBytes(512)).toBe('512 B')
  })

  it('formats kilobytes', () => {
    expect(formatBytes(1024)).toBe('1 KB')
  })

  it('formats megabytes', () => {
    expect(formatBytes(1048576)).toBe('1 MB')
  })

  it('formats gigabytes', () => {
    expect(formatBytes(1073741824)).toBe('1 GB')
  })

  it('formats with decimal', () => {
    expect(formatBytes(1536)).toBe('1.5 KB')
  })
})

describe('cn', () => {
  it('merges class names', () => {
    expect(cn('foo', 'bar')).toBe('foo bar')
  })

  it('handles conditional classes', () => {
    expect(cn('base', false && 'hidden', 'visible')).toBe('base visible')
  })

  it('deduplicates tailwind classes', () => {
    expect(cn('text-red-500', 'text-blue-500')).toBe('text-blue-500')
  })
})

describe('splitCamelCase', () => {
  it('splits camelCase words', () => {
    expect(splitCamelCase('SumpFlow')).toBe('Sump Flow')
  })

  it('splits multiple words', () => {
    expect(splitCamelCase('ReturnPump')).toBe('Return Pump')
  })

  it('leaves single words unchanged', () => {
    expect(splitCamelCase('Circ')).toBe('Circ')
  })

  it('handles empty string', () => {
    expect(splitCamelCase('')).toBe('')
  })
})
