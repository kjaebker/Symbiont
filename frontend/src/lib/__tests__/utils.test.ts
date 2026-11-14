import { describe, it, expect } from 'vitest'
import { relativeTime, overdueBy, daysSince, formatBytes, cn, splitCamelCase } from '../utils'

const daysAgo = (n: number) => new Date(Date.now() - n * 86400_000 - 60_000).toISOString()

describe('long ages', () => {
  it.each([
    [13, '13d ago'],
    [21, '3w ago'],
    [151, '5mo ago'],
    [800, '2y ago'],
  ])('relativeTime of %i days is %s', (days, want) => {
    expect(relativeTime(daysAgo(days))).toBe(want)
  })

  it.each([
    [0, ''],
    [1, '1 day'],
    [3, '3 days'],
    [21, '3 weeks'],
    [151, '5 months'],
  ])('overdueBy %i days is "%s"', (days, want) => {
    expect(overdueBy(daysAgo(days))).toBe(want)
  })

  it('daysSince is 0 for future timestamps', () => {
    expect(daysSince(new Date(Date.now() + 86400_000).toISOString())).toBe(0)
  })
})

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
