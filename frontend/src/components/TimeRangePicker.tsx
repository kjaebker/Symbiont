import { useState, useRef, useEffect } from 'react'
import { ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'

interface TimeRange {
  from: Date
  to: Date
}

interface Preset {
  label: string
  duration: number
}

const presets: Preset[] = [
  { label: 'Last 2h', duration: 2 * 60 * 60 * 1000 },
  { label: 'Last 6h', duration: 6 * 60 * 60 * 1000 },
  { label: 'Last 24h', duration: 24 * 60 * 60 * 1000 },
  { label: 'Last 7d', duration: 7 * 24 * 60 * 60 * 1000 },
  { label: 'Last 30d', duration: 30 * 24 * 60 * 60 * 1000 },
]

interface TimeRangePickerProps {
  value: TimeRange
  onChange: (range: TimeRange) => void
}

function toLocalDatetime(d: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function getActivePreset(range: TimeRange): string | null {
  const now = Date.now()
  if (Math.abs(range.to.getTime() - now) > 60_000) return null
  const duration = range.to.getTime() - range.from.getTime()
  for (const p of presets) {
    if (Math.abs(duration - p.duration) < 60_000) return p.label
  }
  return null
}

function formatRangeLabel(range: TimeRange): string {
  const { from, to } = range
  const sameYear = from.getFullYear() === to.getFullYear()
  const sameDay =
    sameYear &&
    from.getMonth() === to.getMonth() &&
    from.getDate() === to.getDate()

  const monthDay = (d: Date) =>
    d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
  const time = (d: Date) =>
    d.toLocaleTimeString(undefined, { hour: 'numeric', minute: '2-digit' })

  if (sameDay) {
    return `${monthDay(from)}, ${time(from)} – ${time(to)}`
  }
  if (sameYear) {
    return `${monthDay(from)} – ${monthDay(to)}`
  }
  return `${monthDay(from)}, ${from.getFullYear()} – ${monthDay(to)}, ${to.getFullYear()}`
}

export function TimeRangePicker({ value, onChange }: TimeRangePickerProps) {
  const activePreset = getActivePreset(value)
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [])

  const label = activePreset ?? formatRangeLabel(value)

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className={cn(
          'flex items-center gap-2 rounded-xl px-3 py-2 text-sm transition-fluid',
          activePreset
            ? 'bg-primary/20 text-primary'
            : 'bg-surface-container-high text-on-surface hover:bg-surface-container-highest',
        )}
      >
        <span>{label}</span>
        <ChevronDown
          className={cn(
            'h-3.5 w-3.5 transition-fluid',
            open && 'rotate-180',
            activePreset ? 'text-primary/70' : 'text-on-surface-faint',
          )}
        />
      </button>

      {open && (
        <div className="absolute z-20 top-full mt-1.5 left-0 bg-surface-container-high rounded-2xl p-1.5 shadow-abyss flex flex-col sm:flex-row">
          {/* Presets */}
          <div className="flex flex-col min-w-[120px]">
            {presets.map((p) => (
              <button
                key={p.label}
                onClick={() => {
                  const now = new Date()
                  onChange({ from: new Date(now.getTime() - p.duration), to: now })
                  setOpen(false)
                }}
                className={cn(
                  'w-full text-left px-3 py-2 text-sm rounded-xl transition-fluid',
                  activePreset === p.label
                    ? 'text-primary bg-primary/10'
                    : 'text-on-surface hover:bg-surface-container-highest',
                )}
              >
                {p.label}
              </button>
            ))}
          </div>

          {/* Divider — vertical on desktop, horizontal on mobile */}
          <div className="sm:hidden my-1 mx-2 h-px bg-outline-variant/20" />
          <div className="hidden sm:block mx-1 w-px bg-outline-variant/20 self-stretch" />

          {/* Custom date inputs */}
          <div className="flex flex-col justify-center gap-1.5 px-1 py-1 min-w-[220px]">
            <label className="flex items-center gap-2">
              <span className="text-xs text-on-surface-faint w-7 shrink-0">From</span>
              <input
                type="datetime-local"
                value={toLocalDatetime(value.from)}
                max={toLocalDatetime(value.to)}
                onClick={(e) => e.stopPropagation()}
                onChange={(e) => {
                  const d = new Date(e.target.value)
                  if (!isNaN(d.getTime())) {
                    if (d >= value.to) {
                      onChange({ from: d, to: new Date(d.getTime() + 60 * 60 * 1000) })
                    } else {
                      onChange({ ...value, from: d })
                    }
                  }
                }}
                className="flex-1 bg-surface-container text-on-surface text-sm rounded-xl px-2.5 py-1.5 outline-none focus:ring-1 focus:ring-primary/30"
              />
            </label>
            <label className="flex items-center gap-2">
              <span className="text-xs text-on-surface-faint w-7 shrink-0">To</span>
              <input
                type="datetime-local"
                value={toLocalDatetime(value.to)}
                min={toLocalDatetime(value.from)}
                onClick={(e) => e.stopPropagation()}
                onChange={(e) => {
                  const d = new Date(e.target.value)
                  if (!isNaN(d.getTime())) {
                    if (d <= value.from) {
                      onChange({ from: new Date(d.getTime() - 60 * 60 * 1000), to: d })
                    } else {
                      onChange({ ...value, to: d })
                    }
                  }
                }}
                className="flex-1 bg-surface-container text-on-surface text-sm rounded-xl px-2.5 py-1.5 outline-none focus:ring-1 focus:ring-primary/30"
              />
            </label>
          </div>
        </div>
      )}
    </div>
  )
}
