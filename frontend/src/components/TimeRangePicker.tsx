import { cn } from '@/lib/utils'

interface TimeRange {
  from: Date
  to: Date
}

interface Preset {
  label: string
  duration: number // milliseconds
}

const presets: Preset[] = [
  { label: '2h', duration: 2 * 60 * 60 * 1000 },
  { label: '6h', duration: 6 * 60 * 60 * 1000 },
  { label: '24h', duration: 24 * 60 * 60 * 1000 },
  { label: '7d', duration: 7 * 24 * 60 * 60 * 1000 },
  { label: '30d', duration: 30 * 24 * 60 * 60 * 1000 },
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
  const toDiff = Math.abs(range.to.getTime() - now)
  // Consider "active" if `to` is within 60s of now
  if (toDiff > 60_000) return null

  const duration = range.to.getTime() - range.from.getTime()
  for (const p of presets) {
    if (Math.abs(duration - p.duration) < 60_000) return p.label
  }
  return null
}

export function TimeRangePicker({ value, onChange }: TimeRangePickerProps) {
  const activePreset = getActivePreset(value)

  return (
    <div className="flex flex-wrap items-center gap-2">
      {presets.map((p) => (
        <button
          key={p.label}
          onClick={() => {
            const now = new Date()
            onChange({ from: new Date(now.getTime() - p.duration), to: now })
          }}
          className={cn(
            'px-3 py-1.5 rounded-full text-xs font-medium uppercase tracking-wider transition-fluid',
            activePreset === p.label
              ? 'bg-primary/20 text-primary text-glow-primary'
              : 'bg-surface-container-high text-on-surface-dim hover:text-on-surface hover:bg-surface-container-highest',
          )}
        >
          {p.label}
        </button>
      ))}

      <div className="flex items-center gap-1.5 ml-2">
        <input
          type="datetime-local"
          value={toLocalDatetime(value.from)}
          onChange={(e) => {
            const d = new Date(e.target.value)
            if (!isNaN(d.getTime())) onChange({ ...value, from: d })
          }}
          className="bg-surface-container-high text-on-surface text-xs rounded-xl px-2.5 py-1.5 outline-none focus:ring-1 focus:ring-primary/30"
        />
        <span className="text-on-surface-faint text-xs">to</span>
        <input
          type="datetime-local"
          value={toLocalDatetime(value.to)}
          onChange={(e) => {
            const d = new Date(e.target.value)
            if (!isNaN(d.getTime())) onChange({ ...value, to: d })
          }}
          className="bg-surface-container-high text-on-surface text-xs rounded-xl px-2.5 py-1.5 outline-none focus:ring-1 focus:ring-primary/30"
        />
      </div>
    </div>
  )
}
