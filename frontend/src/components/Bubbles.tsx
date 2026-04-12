import { useMemo } from 'react'
import { createPortal } from 'react-dom'

interface BubbleConfig {
  id: number
  left: number
  size: number
  duration: number
  delay: number
  drift: number
  opacity: number
}

// Seeded random-ish generation so bubbles are stable across re-renders via useMemo
function generateBubbles(count: number): BubbleConfig[] {
  return Array.from({ length: count }, (_, i) => {
    const seed = i * 137.508 // golden angle spread
    const r = (n: number) => Math.abs(Math.sin(seed + n) * 10000) % 1

    return {
      id: i,
      left: r(1) * 94,           // 0–94% from left
      size: 6 + r(2) * 16,       // 6–22px
      duration: 14 + r(3) * 18,  // 14–32s
      delay: r(4) * 20,          // 0–20s initial delay
      drift: (r(5) - 0.5) * 100, // ±50px horizontal drift
      opacity: 0.12 + r(6) * 0.2, // 0.12–0.32 max opacity
    }
  })
}

export function Bubbles() {
  const bubbles = useMemo(() => generateBubbles(18), [])

  return createPortal(
    <div
      className="fixed inset-0 pointer-events-none overflow-hidden"
      style={{ zIndex: 0 }}
      aria-hidden="true"
    >
      {bubbles.map((b) => (
        <div
          key={b.id}
          className="absolute bottom-0 rounded-full"
          style={{
            left: `${b.left}%`,
            width: b.size,
            height: b.size,
            border: `1px solid rgba(58, 223, 250, ${b.opacity})`,
            background: `rgba(58, 223, 250, ${b.opacity * 0.15})`,
            animation: `bubble-float ${b.duration}s ease-in ${b.delay}s infinite`,
            '--bubble-drift': `${b.drift}px`,
          } as React.CSSProperties}
        />
      ))}
    </div>,
    document.body,
  )
}
