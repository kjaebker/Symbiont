import { usePageTheme } from '@/lib/pageTheme'

export function Caustic() {
  const { accent, sub } = usePageTheme()

  return (
    <div
      className="fixed inset-0 z-0 pointer-events-none overflow-hidden"
      style={{ mixBlendMode: 'screen' }}
    >
      {/* Blob A — top left */}
      <div
        className="absolute animate-caustic-a"
        style={{
          top: -60,
          left: -40,
          width: 320,
          height: 220,
          background: `radial-gradient(ellipse at center, ${accent}2e, transparent 70%)`,
          filter: 'blur(24px)',
        }}
      />
      {/* Blob B — top right */}
      <div
        className="absolute animate-caustic-b"
        style={{
          top: -40,
          right: -60,
          width: 260,
          height: 340,
          background: `radial-gradient(ellipse at center, ${sub}26, transparent 70%)`,
          filter: 'blur(28px)',
        }}
      />
      {/* Blob C — bottom center */}
      <div
        className="absolute animate-caustic-c"
        style={{
          bottom: -40,
          left: 'calc(50% - 120px)',
          width: 240,
          height: 240,
          background: `radial-gradient(ellipse at center, ${accent}1f, transparent 70%)`,
          filter: 'blur(22px)',
        }}
      />
    </div>
  )
}
