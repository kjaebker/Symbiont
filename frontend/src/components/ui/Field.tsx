import type { InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react'
import { cn } from '@/lib/utils'
import { controlClass, labelClass } from './styles'

interface FieldProps {
  label: string
  /** Helper text under the control. */
  hint?: string
  /** Error text under the control; replaces the hint. */
  error?: string
  className?: string
  children: ReactNode
}

/** A labelled form control. Wraps the control in a <label> so clicks focus it. */
export function Field({ label, hint, error, className, children }: FieldProps) {
  return (
    <label className={cn('block', className)}>
      <span className={cn(labelClass, 'block mb-1.5')}>{label}</span>
      {children}
      {error
        ? <span className="block mt-1 text-xs text-tertiary">{error}</span>
        : hint && <span className="block mt-1 text-xs text-on-surface-faint">{hint}</span>}
    </label>
  )
}

export function Input({ className, ...rest }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={cn(controlClass, className)} {...rest} />
}

export function Select({ className, children, ...rest }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select className={cn(controlClass, 'cursor-pointer', className)} {...rest}>
      {children}
    </select>
  )
}

export function Textarea({ className, ...rest }: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return <textarea className={cn(controlClass, 'resize-y', className)} {...rest} />
}
