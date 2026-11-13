import { cn } from '@/lib/utils'

interface ToggleProps {
  checked: boolean
  onChange: (checked: boolean) => void
  /** Accessible name, e.g. "Enable rule". */
  label: string
  disabled?: boolean
}

/** On/off switch. */
export function Toggle({ checked, onChange, label, disabled }: ToggleProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        'relative w-10 h-5 rounded-full shrink-0 transition-fluid cursor-pointer disabled:opacity-50',
        checked ? 'bg-primary' : 'bg-surface-container-highest',
      )}
    >
      <span
        className={cn(
          'absolute top-0.5 h-4 w-4 rounded-full bg-on-surface transition-fluid',
          checked ? 'left-5.5' : 'left-0.5',
        )}
      />
    </button>
  )
}
