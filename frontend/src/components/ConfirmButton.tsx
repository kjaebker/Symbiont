import { useState, type ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface ConfirmButtonProps {
  /** Runs once the action has been confirmed. */
  onConfirm: () => void
  /** Content of the resting button (usually an icon, optionally with text). */
  children: ReactNode
  /** Classes for the resting button. */
  className?: string
  /** Text of the confirming button, e.g. "Delete" or "Remove photo". */
  confirmLabel?: string
  /** Accessible name and tooltip for the resting button. */
  title?: string
  disabled?: boolean
  /** Use over photos, where the pair needs its own backdrop to stay legible. */
  floating?: boolean
}

/**
 * The app's one pattern for destructive actions: the first click swaps the
 * button for an inline Confirm / Cancel pair, the second click acts.
 */
export function ConfirmButton({
  onConfirm,
  children,
  className,
  confirmLabel = 'Delete',
  title,
  disabled,
  floating,
}: ConfirmButtonProps) {
  const [confirming, setConfirming] = useState(false)

  if (!confirming) {
    return (
      <button
        type="button"
        onClick={() => setConfirming(true)}
        disabled={disabled}
        title={title}
        aria-label={title}
        className={cn('cursor-pointer', className)}
      >
        {children}
      </button>
    )
  }

  return (
    <span className={cn('inline-flex items-center gap-1', floating && 'glass rounded-xl p-1')}>
      <button
        type="button"
        autoFocus
        onClick={() => { setConfirming(false); onConfirm() }}
        disabled={disabled}
        className="px-2.5 py-1 rounded-lg text-xs font-medium text-tertiary bg-tertiary/10 hover:bg-tertiary/20 transition-fluid cursor-pointer whitespace-nowrap"
      >
        {confirmLabel}
      </button>
      <button
        type="button"
        onClick={() => setConfirming(false)}
        className="px-2.5 py-1 rounded-lg text-xs font-medium text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
      >
        Cancel
      </button>
    </span>
  )
}
