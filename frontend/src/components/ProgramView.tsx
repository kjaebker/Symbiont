import { useMemo } from 'react'
import type { OutputProgram, TDataPoint } from '@/api/client'

// ── Helpers ──────────────────────────────────────────────────────────────────

function secondsToHHMM(s: number): string {
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
}

function nowSeconds(): number {
  const d = new Date()
  return d.getHours() * 3600 + d.getMinutes() * 60 + d.getSeconds()
}

// Find which channels have any non-zero values.
function activeChannels(points: TDataPoint[]): number[] {
  const active: number[] = []
  for (let ch = 0; ch < 13; ch++) {
    if (points.some((p) => p.ch[ch] > 0)) active.push(ch)
  }
  return active
}

// Build an SVG polyline path for a channel, linearly interpolated.
function buildPath(points: TDataPoint[], ch: number, w: number, h: number): string {
  if (points.length === 0) return ''
  const maxT = 86400
  const coords = points.map((p) => {
    const x = (p.t / maxT) * w
    const y = h - (p.ch[ch] / 100) * h
    return `${x.toFixed(1)},${y.toFixed(1)}`
  })
  return `M ${coords.join(' L ')}`
}

// Build the closed SVG path for the area fill (path + baseline).
function buildAreaPath(points: TDataPoint[], ch: number, w: number, h: number): string {
  if (points.length === 0) return ''
  const maxT = 86400
  const pts = points.map((p) => ({
    x: (p.t / maxT) * w,
    y: h - (p.ch[ch] / 100) * h,
  }))
  const line = pts.map((p, i) => `${i === 0 ? 'M' : 'L'} ${p.x.toFixed(1)},${p.y.toFixed(1)}`).join(' ')
  const close = ` L ${pts[pts.length - 1].x.toFixed(1)},${h} L ${pts[0].x.toFixed(1)},${h} Z`
  return line + close
}

// ── TData chart ───────────────────────────────────────────────────────────────

const CHANNEL_COLORS = ['#3adffa', '#6dfe9c', '#ff8796', '#ffd27d', '#c8a2f8']

interface TDataChartProps {
  points: TDataPoint[]
}

function TDataChart({ points }: TDataChartProps) {
  const W = 600
  const H = 80
  const nowS = nowSeconds()
  const nowX = (nowS / 86400) * W

  const channels = useMemo(() => activeChannels(points), [points])

  // Hour tick marks every 4h
  const hourTicks = [0, 4, 8, 12, 16, 20, 24]

  return (
    <div className="w-full">
      <svg
        viewBox={`0 0 ${W} ${H + 20}`}
        className="w-full"
        style={{ maxHeight: 120 }}
        preserveAspectRatio="none"
      >
        <defs>
          {channels.map((ch, i) => (
            <linearGradient
              key={ch}
              id={`grad-ch${ch}`}
              x1="0" y1="0" x2="0" y2="1"
            >
              <stop offset="0%" stopColor={CHANNEL_COLORS[i % CHANNEL_COLORS.length]} stopOpacity="0.35" />
              <stop offset="100%" stopColor={CHANNEL_COLORS[i % CHANNEL_COLORS.length]} stopOpacity="0.02" />
            </linearGradient>
          ))}
        </defs>

        {/* Hour gridlines */}
        {hourTicks.map((h) => {
          const x = (h / 24) * W
          return (
            <line
              key={h}
              x1={x} y1={0} x2={x} y2={H}
              stroke="rgba(255,255,255,0.06)"
              strokeWidth="1"
            />
          )
        })}

        {/* Area fills */}
        {channels.map((ch) => (
          <path
            key={`area-${ch}`}
            d={buildAreaPath(points, ch, W, H)}
            fill={`url(#grad-ch${ch})`}
          />
        ))}

        {/* Lines */}
        {channels.map((ch, i) => (
          <path
            key={`line-${ch}`}
            d={buildPath(points, ch, W, H)}
            fill="none"
            stroke={CHANNEL_COLORS[i % CHANNEL_COLORS.length]}
            strokeWidth="2"
            strokeLinejoin="round"
            strokeLinecap="round"
          />
        ))}

        {/* Current time marker */}
        <line
          x1={nowX} y1={0} x2={nowX} y2={H}
          stroke="rgba(255,255,255,0.5)"
          strokeWidth="1.5"
          strokeDasharray="4 3"
        />

        {/* Hour labels */}
        {hourTicks.filter((h) => h < 24).map((h) => {
          const x = (h / 24) * W
          return (
            <text
              key={`lbl-${h}`}
              x={x}
              y={H + 14}
              textAnchor="middle"
              fontSize="9"
              fill="rgba(255,255,255,0.35)"
              fontFamily="Manrope, sans-serif"
            >
              {String(h).padStart(2, '0')}:00
            </text>
          )
        })}
      </svg>

      {/* Key points table */}
      <div className="mt-2 flex flex-wrap gap-x-4 gap-y-1">
        {points.filter((p, i) => {
          // Show points where any active channel has a value, or first/last
          return i === 0 || i === points.length - 1 || channels.some((ch) => p.ch[ch] > 0)
        }).map((p, i) => {
          const vals = channels.map((ch) => p.ch[ch])
          const allZero = vals.every((v) => v === 0)
          return (
            <div key={i} className="flex items-center gap-1.5 text-xs">
              <span className="text-on-surface-faint font-mono">{secondsToHHMM(p.t)}</span>
              {channels.map((ch, ci) => (
                <span
                  key={ch}
                  style={{ color: allZero ? 'rgba(255,255,255,0.25)' : CHANNEL_COLORS[ci % CHANNEL_COLORS.length] }}
                  className="font-semibold tabular-nums"
                >
                  {p.ch[ch]}%
                </span>
              ))}
            </div>
          )
        })}
      </div>
    </div>
  )
}

// ── Logic program code view ───────────────────────────────────────────────────

function colorizeProgLine(line: string): React.ReactNode {
  // Keywords
  if (/^(Set|Fallback)\s/i.test(line)) {
    const [kw, ...rest] = line.split(' ')
    return (
      <>
        <span className="text-primary/80">{kw}</span>
        {' '}
        <span className="text-secondary">{rest.join(' ')}</span>
      </>
    )
  }
  if (/^If\s/i.test(line)) {
    // "If <condition> Then <action>"
    const thenIdx = line.toLowerCase().indexOf(' then ')
    if (thenIdx >= 0) {
      const condition = line.slice(3, thenIdx)
      const action = line.slice(thenIdx + 6)
      return (
        <>
          <span className="text-on-surface-faint">If</span>
          <span className="text-on-surface"> {condition} </span>
          <span className="text-on-surface-faint">Then</span>
          <span className="text-secondary"> {action}</span>
        </>
      )
    }
  }
  if (/^(Defer|Min Time)\s/i.test(line)) {
    const [kw, ...rest] = line.split(' ')
    return (
      <>
        <span className="text-tertiary/80">{kw}</span>
        {' '}
        <span className="text-on-surface-dim">{rest.join(' ')}</span>
      </>
    )
  }
  return <span className="text-on-surface-dim">{line}</span>
}

interface ProgramCodeProps {
  prog: string
}

function ProgramCode({ prog }: ProgramCodeProps) {
  const lines = prog.split('\n').filter((l) => l.trim())
  return (
    <pre className="text-xs font-mono leading-5 select-text">
      {lines.map((line, i) => (
        <div key={i} className="flex gap-3">
          <span className="text-on-surface-faint/40 w-4 text-right shrink-0 select-none">
            {i + 1}
          </span>
          <span>{colorizeProgLine(line.trim())}</span>
        </div>
      ))}
    </pre>
  )
}

// ── Main component ────────────────────────────────────────────────────────────

interface ProgramViewProps {
  program: OutputProgram
}

export function ProgramView({ program }: ProgramViewProps) {
  const hasTData = program.tdata && program.tdata.length > 0

  return (
    <div className="space-y-3">
      {hasTData ? (
        <>
          <div className="flex items-center gap-2">
            <span className="text-xs text-on-surface-faint uppercase tracking-widest">
              Daily Schedule
            </span>
            <span className="text-xs text-primary/60 font-mono">
              {program.tdata!.length} points
            </span>
          </div>
          <TDataChart points={program.tdata!} />
        </>
      ) : (
        <>
          <div className="flex items-center gap-2">
            <span className="text-xs text-on-surface-faint uppercase tracking-widest">
              Program
            </span>
            <span className="text-xs text-on-surface-faint/50 font-mono">
              {program.prog.split('\n').filter((l) => l.trim()).length} lines
            </span>
          </div>
          <ProgramCode prog={program.prog} />
        </>
      )}
    </div>
  )
}
