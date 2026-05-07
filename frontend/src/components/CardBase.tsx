import { type ReactNode } from 'react'
import { cn } from '@/lib/utils'
import type { ProbeStatus } from '@/api/types'

// Standard card gradient: radial accent at top-right + linear tint.
// Shared by DeviceCard and OutletCard; other cards use category-specific tints.
export function cardGradient(color: string, isOn = false): string {
  return `radial-gradient(ellipse 55% 50% at 100% 0%, ${color}22 0%, transparent 100%), linear-gradient(145deg, var(--color-surface-container) 20%, ${color}${isOn ? '33' : '18'})`
}

interface CardIconBlobProps {
  color: string      // hex accent, e.g. '#3adffa'
  blobShape: string  // CSS border-radius
  size?: 'sm' | 'md' // sm=w-8 h-8, md=w-10 h-10 (default)
  status?: ProbeStatus
  isOn?: boolean
  activeBg?: string     // background when isOn && no status alert (falls back to ${color}1a)
  normalBg?: string     // background at rest (falls back to ${color}0a)
  normalShadow?: string // box-shadow at rest (defaults to 'none')
  className?: string
  children: ReactNode
}

// Icon container with status-aware background, glow, and pulse animation.
// Critical and warning states are handled automatically; isOn drives the
// "active" glow; everything else uses the calm idle appearance.
export function CardIconBlob({
  color,
  blobShape,
  size = 'md',
  status,
  isOn,
  activeBg,
  normalBg,
  normalShadow = 'none',
  className,
  children,
}: CardIconBlobProps) {
  const isCritical = status === 'critical'
  const isWarning = status === 'warning'

  const background = isCritical
    ? 'rgba(255,135,150,0.22)'
    : isWarning
      ? 'rgba(251,191,36,0.18)'
      : isOn
        ? (activeBg ?? `${color}1a`)
        : (normalBg ?? `${color}0a`)

  const boxShadow = isCritical
    ? '0 0 16px rgba(255,135,150,0.55)'
    : isWarning
      ? '0 0 14px rgba(251,191,36,0.45)'
      : isOn
        ? `0 0 14px ${color}33`
        : normalShadow

  const animation = isCritical
    ? 'bioluminescent-pulse 1.8s ease-in-out infinite'
    : isWarning
      ? 'bioluminescent-pulse 2.8s ease-in-out infinite'
      : undefined

  return (
    <div
      className={cn(
        size === 'sm' ? 'w-8 h-8' : 'w-10 h-10',
        'flex items-center justify-center shrink-0 transition-fluid',
        className,
      )}
      style={{ borderRadius: blobShape, background, boxShadow, animation }}
    >
      {children}
    </div>
  )
}

interface OutletControlButtonsProps {
  outletState: string
  onControl: (state: 'ON' | 'OFF' | 'AUTO') => void
  isPending: boolean
  isError: boolean
  revealed: boolean    // true = controls visible; false = collapsed
  innerPadding?: string // Tailwind pt-* class for the button row (default 'pt-2')
  className?: string   // extra classes on the outer grid wrapper
}

// Slide-reveal OFF/ON/AUTO control row shared by DeviceCard and OutletCard.
// Uses a grid-template-rows animation to expand/collapse without layout shift.
export function OutletControlButtons({
  outletState,
  onControl,
  isPending,
  isError,
  revealed,
  innerPadding = 'pt-2',
  className,
}: OutletControlButtonsProps) {
  const isAuto = outletState === 'AON' || outletState === 'AOF' || outletState === 'TBL'

  return (
    <div
      className={cn('grid overflow-hidden', className)}
      style={{
        gridTemplateRows: revealed ? '1fr' : '0fr',
        transition: 'grid-template-rows 250ms cubic-bezier(0.65, 0, 0.35, 1)',
      }}
    >
      <div className="min-h-0">
        <div className={cn('flex gap-1.5', innerPadding)}>
          {(['OFF', 'ON', 'AUTO'] as const).map((s) => {
            const active =
              (s === 'OFF' && (outletState === 'OFF' || outletState === 'AOF')) ||
              (s === 'ON' && outletState === 'ON') ||
              (s === 'AUTO' && isAuto)
            return (
              <button
                key={s}
                onClick={() => onControl(s)}
                disabled={isPending}
                className={cn(
                  'flex-1 py-2 rounded-xl text-xs font-semibold uppercase tracking-wider transition-fluid',
                  active
                    ? s === 'AUTO'
                      ? 'bg-primary text-on-primary'
                      : s === 'ON'
                        ? 'bg-secondary text-on-secondary'
                        : 'bg-surface-container-highest text-on-surface'
                    : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim hover:bg-surface-container-highest',
                )}
              >
                {s}
              </button>
            )
          })}
        </div>
        {isError && (
          <p className="text-xs text-tertiary mt-2">Failed to set state. Try again.</p>
        )}
      </div>
    </div>
  )
}
