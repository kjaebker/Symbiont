import type { HTMLAttributes } from 'react'
import { cn } from '@/lib/utils'

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  /** Remove the default padding, e.g. for a table that runs edge to edge. */
  flush?: boolean
}

/** The standard content module: one surface step above the page. */
export function Card({ flush, className, ...rest }: CardProps) {
  return (
    <div
      className={cn('bg-surface-container rounded-2xl', flush ? 'overflow-hidden' : 'p-5', className)}
      {...rest}
    />
  )
}
