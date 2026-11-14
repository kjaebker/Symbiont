import { Link } from 'react-router-dom'
import { Fish, Flower2, Shell, Box } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { cn, relativeTime, absoluteTime } from '@/lib/utils'
import { isStale } from '@/lib/staleness'
import type { LivestockItem, LivestockType, LivestockStatus } from '@/api/types'
import { thumbUrl } from '@/api/client'
import { Chip } from '@/components/ui'
import type { ChipTone } from '@/components/ui'

const typeIcons: Record<LivestockType, LucideIcon> = {
  fish:         Fish,
  coral:        Flower2,
  invertebrate: Shell,
  other:        Box,
}

const typePlaceholderClass: Record<LivestockType, string> = {
  fish:         'bg-primary/10',
  coral:        'bg-tertiary/10',
  invertebrate: 'bg-secondary/10',
  other:        'bg-surface-container-high',
}

const typeIconClass: Record<LivestockType, string> = {
  fish:         'text-primary/30',
  coral:        'text-tertiary/30',
  invertebrate: 'text-secondary/30',
  other:        'text-on-surface-faint/30',
}

const statusTone: Record<LivestockStatus, ChipTone> = {
  healthy:    'secondary',
  sick:       'warning',
  quarantine: 'primary',
  deceased:   'neutral',
}

const statusLabel: Record<LivestockStatus, string> = {
  healthy:    'Healthy',
  sick:       'Sick',
  quarantine: 'Quarantine',
  deceased:   'Deceased',
}

interface LivestockCardProps {
  item: LivestockItem
}

/**
 * A square photo tile with the name over the bottom of the image. Healthy is
 * the normal case and says nothing; any other status gets a chip.
 */
export function LivestockCard({ item }: LivestockCardProps) {
  const TypeIcon = typeIcons[item.type] ?? Box

  // A sick or quarantined animal should be getting looked at. Say when it was
  // last observed, and flag it once that is long ago (or never).
  const unwell = item.status === 'sick' || item.status === 'quarantine'
  const neglected = unwell && (!item.last_observed_at || isStale(item.last_observed_at, 'livestockObservation'))

  return (
    <Link
      to={`/livestock/${item.id}`}
      className="group relative block aspect-square rounded-2xl overflow-hidden bg-surface-container focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
    >
      {item.image_path ? (
        <img
          src={thumbUrl(item.image_path)}
          alt=""
          loading="lazy"
          className={cn(
            'absolute inset-0 w-full h-full object-cover transition-transform duration-500 ease-out group-hover:scale-105',
            item.status === 'deceased' && 'grayscale opacity-60',
          )}
        />
      ) : (
        <div className={cn('absolute inset-0 flex items-center justify-center pb-8', typePlaceholderClass[item.type])}>
          <TypeIcon className={cn('h-10 w-10', typeIconClass[item.type])} />
        </div>
      )}

      {/* Status (only when not healthy) and quantity */}
      <div className="absolute top-2 inset-x-2 flex items-start justify-between gap-2">
        {item.status !== 'healthy' ? (
          <Chip tone={statusTone[item.status]} className="glass">{statusLabel[item.status]}</Chip>
        ) : <span />}
        {item.quantity > 1 && (
          <span className="glass px-2 py-1 rounded-full text-xs font-semibold text-on-surface leading-none">
            ×{item.quantity}
          </span>
        )}
      </div>

      {/* Identity, over a scrim so it reads on any photo */}
      <div className="absolute inset-x-0 bottom-0 px-3 pb-2.5 pt-8 bg-gradient-to-t from-surface-container-lowest/95 via-surface-container-lowest/60 to-transparent">
        <div className="font-semibold text-sm text-on-surface leading-tight line-clamp-2">{item.name}</div>
        {unwell ? (
          <div
            className={cn('text-xs truncate mt-0.5', neglected ? 'text-warning' : 'text-on-surface-dim')}
            title={item.last_observed_at ? `Last observed ${absoluteTime(item.last_observed_at)}` : undefined}
          >
            {item.last_observed_at ? `Seen ${relativeTime(item.last_observed_at)}` : 'No observations'}
          </div>
        ) : item.species && (
          <div className="text-xs text-on-surface-dim italic truncate mt-0.5">{item.species}</div>
        )}
      </div>
    </Link>
  )
}
