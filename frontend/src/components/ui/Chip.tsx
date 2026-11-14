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
  /** Makes the chip a button (a filter toggle). */
  onClick?: () => void
  /** For clickable chips: this filter is currently applied. */
  active?: boolean
  className?: string
  children: ReactNode
}

/** Small uppercase status pill: severity, state, category. Clickable when given onClick. */
export function Chip({ tone = 'neutral', pulse, onClick, active, className, children }: ChipProps) {
  const classes = cn(
    'inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium uppercase tracking-wider whitespace-nowrap leading-none',
    toneClass[tone],
    pulse && 'animate-bio-pulse',
    onClick && 'cursor-pointer transition-fluid hover:brightness-125',
    active && 'ring-1 ring-current',
    className,
  )
  if (onClick) {
    return (
      <button type="button" onClick={onClick} aria-pressed={!!active} className={classes}>
        {children}
      </button>
    )
  }
  return <span className={classes}>{children}</span>
}
