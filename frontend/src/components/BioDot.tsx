interface BioDotProps {
  color: string
  size?: number
}

export function BioDot({ color, size = 10 }: BioDotProps) {
  const outerSize = size * 2.5
  const ringSize = size * 2.2

  return (
    <div
      className="relative flex-shrink-0 flex items-center justify-center"
      style={{ width: outerSize, height: outerSize }}
    >
      <div
        className="absolute rounded-full animate-bio-ring"
        style={{
          width: ringSize,
          height: ringSize,
          border: `1.5px solid ${color}`,
        }}
      />
      <div
        className="rounded-full"
        style={{
          width: size,
          height: size,
          background: color,
          boxShadow: `0 0 5px ${color}88`,
        }}
      />
    </div>
  )
}
