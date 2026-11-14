import { absoluteTime, relativeTime } from '@/lib/utils'

interface TimeAgoProps {
  ts: string
  className?: string
}

/** A relative time ("5mo ago") that shows the exact date and time on hover. */
export function TimeAgo({ ts, className }: TimeAgoProps) {
  return (
    <time dateTime={ts} title={absoluteTime(ts)} className={className}>
      {relativeTime(ts)}
    </time>
  )
}
