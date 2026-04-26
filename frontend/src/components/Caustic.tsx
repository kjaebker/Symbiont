import { usePageTheme } from '@/lib/pageTheme'

export function Caustic() {
  const { accent, sub } = usePageTheme()

  const blobStyle = { transition: 'background 600ms ease-in-out' }

  return (
    <div
      className="fixed inset-0 z-0 pointer-events-none"
      style={{ mixBlendMode: 'screen', clipPath: 'inset(0)' }}
    >
      {/* Blob A — top left */}
      <div
        className="absolute animate-caustic-a"
        style={{
          ...blobStyle,
          top: -60,
          left: -40,
          width: 380,
          height: 260,
          background: `radial-gradient(ellipse at center, ${accent}55, transparent 70%)`,
          filter: 'blur(24px)',
        }}
      />
      {/* Blob B — top right, kept inside viewport so Safari can't bleed it */}
      <div
        className="absolute animate-caustic-b"
        style={{
          ...blobStyle,
          top: -40,
          right: 0,
          width: 240,
          height: 400,
          background: `radial-gradient(ellipse at center, ${sub}44, transparent 70%)`,
          filter: 'blur(28px)',
        }}
      />
      {/* Blob C — bottom center */}
      <div
        className="absolute animate-caustic-c"
        style={{
          ...blobStyle,
          bottom: -40,
          left: 'calc(50% - 140px)',
          width: 280,
          height: 280,
          background: `radial-gradient(ellipse at center, ${accent}33, transparent 70%)`,
          filter: 'blur(22px)',
        }}
      />
    </div>
  )
}
