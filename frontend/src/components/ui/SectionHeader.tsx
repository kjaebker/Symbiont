import type { ElementType, ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface SectionHeaderProps {
  title: string
  icon?: ElementType
  /** Right-aligned content, usually a Button. */
  action?: ReactNode
  className?: string
}

/** Heading for a section within a page ("Due Now", "Recent Events"). */
export function SectionHeader({ title, icon: Icon, action, className }: SectionHeaderProps) {
  return (
    <div className={cn('flex items-center justify-between gap-3', className)}>
      <div className="flex items-center gap-2 min-w-0">
        {Icon && <Icon size={16} className="text-primary shrink-0" />}
        <h2 className="text-sm font-semibold text-on-surface uppercase tracking-widest truncate">{title}</h2>
      </div>
      {action}
    </div>
  )
}
