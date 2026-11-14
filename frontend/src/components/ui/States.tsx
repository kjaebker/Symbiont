import type { ReactNode } from 'react'

/** Placeholder while a query loads. */
export function LoadingState({ label }: { label: string }) {
  return (
    <div className="p-12 flex items-center justify-center">
      <div className="flex flex-col items-center gap-3">
        <div className="h-8 w-8 rounded-full bg-primary/20 animate-bio-pulse" />
        <span className="text-xs text-on-surface-faint uppercase tracking-widest">{label}</span>
      </div>
    </div>
  )
}

/** Shown when a list has nothing in it, or could not be loaded. */
export function EmptyState({ icon, message, action }: { icon?: ReactNode; message: string; action?: ReactNode }) {
  return (
    <div className="p-12 flex flex-col items-center justify-center gap-3">
      {icon && <div className="text-on-surface-faint">{icon}</div>}
      <span className="text-on-surface-dim text-sm text-center max-w-sm">{message}</span>
      {action}
    </div>
  )
}
