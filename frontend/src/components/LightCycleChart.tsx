interface LightCycleSeries {
  data: number[]
  color: string
  label: string
}

interface LightCycleChartProps {
  series: LightCycleSeries[]
}

const PLACEHOLDER_COLORS = ['#3adffa', '#6dfe9c', '#ff8796']

export function LightCycleChart({ series }: LightCycleChartProps) {
  const hasData = series.some((s) => s.data.length >= 2)

  if (!hasData) {
    return (
      <div className="flex flex-col gap-1.5">
        <div className="h-[34px] flex items-end gap-px opacity-20">
          {Array.from({ length: 48 }).map((_, i) => {
            const x = i / 47
            const h = Math.round(4 + 26 * Math.exp(-Math.pow((x - 0.5) * 4, 2)))
            return (
              <div
                key={i}
                className="flex-1 rounded-full"
                style={{ height: `${h}px`, background: PLACEHOLDER_COLORS[0] }}
              />
            )
          })}
        </div>
        <div className="flex gap-3">
          {series.map((s, i) => (
            <div key={i} className="flex items-center gap-1">
              <div className="w-1.5 h-1.5 rounded-full shrink-0" style={{ background: s.color }} />
              <span className="text-[9px] uppercase tracking-widest font-semibold" style={{ color: s.color, opacity: 0.7 }}>
                {s.label}
              </span>
            </div>
          ))}
        </div>
      </div>
    )
  }

  const w = 200
  const h = 34
  const pad = 2
  // Fixed 0-100 scale so intensity % is always in context. A drop from 85→80%
  // looks like a small step, not a cliff.
  const SCALE_MAX = 100

  function toPoints(data: number[]): string {
    if (data.length < 2) return ''
    return data
      .map((v, i) => {
        const x = (i / (data.length - 1)) * w
        const y = h - pad - (Math.min(v, SCALE_MAX) / SCALE_MAX) * (h - pad * 2)
        return `${x},${y}`
      })
      .join(' ')
  }

  return (
    <div className="flex flex-col gap-1.5">
      <svg viewBox={`0 0 ${w} ${h}`} className="w-full" style={{ height: h }} preserveAspectRatio="none">
        <defs>
          {series.map((s, i) => (
            <linearGradient key={i} id={`lcc-fill-${i}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={s.color} stopOpacity="0.25" />
              <stop offset="100%" stopColor={s.color} stopOpacity="0" />
            </linearGradient>
          ))}
        </defs>
        {series.map((s, i) => {
          const pts = toPoints(s.data)
          if (!pts) return null
          return (
            <g key={i}>
              <polygon
                points={`0,${h} ${pts} ${w},${h}`}
                fill={`url(#lcc-fill-${i})`}
              />
              <polyline
                points={pts}
                fill="none"
                stroke={s.color}
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
                style={{ filter: `drop-shadow(0 0 3px ${s.color}60)` }}
              />
            </g>
          )
        })}
      </svg>
      <div className="flex gap-3">
        {series.map((s, i) => (
          <div key={i} className="flex items-center gap-1">
            <div
              className="w-1.5 h-1.5 rounded-full shrink-0"
              style={{ background: s.color, boxShadow: `0 0 4px ${s.color}` }}
            />
            <span
              className="text-[9px] uppercase tracking-widest font-semibold"
              style={{ color: s.color, opacity: 0.8 }}
            >
              {s.label}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
