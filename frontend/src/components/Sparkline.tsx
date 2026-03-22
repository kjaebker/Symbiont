interface SparklineProps {
  data: number[]
  color?: string
}

export function Sparkline({ data, color = '#3adffa' }: SparklineProps) {
  if (data.length < 2) {
    return (
      <div className="w-full h-full flex items-center justify-center">
        <div className="flex gap-1">
          {Array.from({ length: 8 }).map((_, i) => (
            <div
              key={i}
              className="w-1 bg-surface-container-highest rounded-full"
              style={{ height: `${12 + Math.random() * 20}px` }}
            />
          ))}
        </div>
      </div>
    )
  }

  const min = Math.min(...data)
  const max = Math.max(...data)
  const range = max - min || 1
  const w = 200
  const h = 40
  const pad = 2

  const points = data.map((v, i) => {
    const x = (i / (data.length - 1)) * w
    const y = h - pad - ((v - min) / range) * (h - pad * 2)
    return `${x},${y}`
  })

  return (
    <svg viewBox={`0 0 ${w} ${h}`} className="w-full h-full" preserveAspectRatio="none">
      <defs>
        <linearGradient id={`spark-fill-${color}`} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={color} stopOpacity="0.2" />
          <stop offset="100%" stopColor={color} stopOpacity="0" />
        </linearGradient>
      </defs>
      <polyline
        points={points.join(' ')}
        fill="none"
        stroke={color}
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        style={{ filter: `drop-shadow(0 0 4px ${color}50)` }}
      />
      <polygon
        points={`0,${h} ${points.join(' ')} ${w},${h}`}
        fill={`url(#spark-fill-${color})`}
      />
    </svg>
  )
}
