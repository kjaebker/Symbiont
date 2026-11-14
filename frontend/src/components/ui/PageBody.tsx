import type { ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface PageBodyProps {
  /** Reading-width layout for a single record (e.g. a livestock detail page). */
  narrow?: boolean
  className?: string
  children: ReactNode
}

/**
 * The content area under a PageHeader. Every page uses the same width and
 * gutters so content lines up with the header as you move between pages.
 */
export function PageBody({ narrow, className, children }: PageBodyProps) {
  return (
    <div className={cn('px-6 md:px-8 pt-5 pb-8 mx-auto space-y-6', narrow ? 'max-w-3xl' : 'max-w-7xl', className)}>
      {children}
    </div>
  )
}
