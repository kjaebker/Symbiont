import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

export type ChipTone = 'neutral' | 'primary' | 'secondary' | 'warning' | 'critical'

const toneClass: Record<ChipTone, string> = {
  neutral: 'bg-surface-container-high text-on-surface-dim',
  primary: 'bg-primary/15 text-primary',
  secondary: 'bg-secondary/15 text-secondary',
  warning: 'bg-warning/15 text-warning',
  critical: 'bg-tertiary/15 text-tertiary',
}

interface ChipProps {
  tone?: ChipTone
  /** Slow pulse for live states ("Active"). */
  pulse?: boolean
  className?: string
  children: ReactNode
}

/** Small uppercase status pill: severity, state, category. */
export function Chip({ tone = 'neutral', pulse, className, children }: ChipProps) {
  return (
    <span
      className={cn(
        'inline-block px-2 py-0.5 rounded-full text-xs font-medium uppercase tracking-wider whitespace-nowrap',
        toneClass[tone],
        pulse && 'animate-bio-pulse',
        className,
      )}
    >
      {children}
    </span>
  )
}
