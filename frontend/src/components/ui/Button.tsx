import type { ButtonHTMLAttributes, ElementType } from 'react'
import { cn } from '@/lib/utils'

type Variant = 'primary' | 'tonal' | 'secondary' | 'ghost' | 'danger'
type Size = 'sm' | 'md'

const variantClass: Record<Variant, string> = {
  /** The one main action of a form or dialog. */
  primary: 'text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary',
  /** Main action inside a list row or card ("Mark Done", "Log Dose"). */
  tonal: 'text-primary bg-primary/15 hover:bg-primary/25',
  /** Cancel and other neutral actions. */
  secondary: 'text-on-surface-dim bg-surface-container-high hover:text-on-surface hover:bg-surface-container-highest',
  /** Low-emphasis action with no resting background. */
  ghost: 'text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high',
  /** Destructive action, after confirmation. */
  danger: 'text-tertiary bg-tertiary/10 hover:bg-tertiary/20',
}

const sizeClass: Record<Size, string> = {
  sm: 'px-3 py-1.5 text-xs gap-1.5',
  md: 'px-4 py-2 text-sm gap-2',
}

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  size?: Size
  icon?: ElementType
  /** Shows the button as busy and disables it. */
  loading?: boolean
}

export function Button({
  variant = 'secondary',
  size = 'md',
  icon: Icon,
  loading,
  disabled,
  className,
  children,
  type = 'button',
  ...rest
}: ButtonProps) {
  return (
    <button
      type={type}
      disabled={disabled || loading}
      className={cn(
        'inline-flex items-center justify-center rounded-xl font-semibold whitespace-nowrap cursor-pointer',
        'transition-fluid disabled:opacity-50 disabled:cursor-not-allowed',
        variantClass[variant],
        sizeClass[size],
        className,
      )}
      {...rest}
    >
      {Icon && <Icon size={size === 'sm' ? 14 : 16} />}
      {children}
    </button>
  )
}

interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon: ElementType
  /** Accessible name; also shown as the tooltip. */
  label: string
  tone?: 'default' | 'primary' | 'danger'
  size?: number
}

const toneClass = {
  default: 'hover:text-on-surface hover:bg-surface-container-high',
  primary: 'hover:text-primary hover:bg-primary/10',
  danger: 'hover:text-tertiary hover:bg-tertiary/10',
}

/** Icon-only button for row actions (edit, delete, close). */
export function IconButton({
  icon: Icon,
  label,
  tone = 'default',
  size = 14,
  className,
  type = 'button',
  ...rest
}: IconButtonProps) {
  return (
    <button
      type={type}
      title={label}
      aria-label={label}
      className={cn(
        'p-1.5 rounded-lg text-on-surface-faint cursor-pointer transition-fluid disabled:opacity-50',
        toneClass[tone],
        className,
      )}
      {...rest}
    >
      <Icon size={size} />
    </button>
  )
}
