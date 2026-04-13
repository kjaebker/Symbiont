import { Link } from 'react-router-dom'
import { Fish, Flower2, Shell, Box } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { LivestockItem, LivestockType, LivestockStatus } from '@/api/types'

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

const statusDotClass: Record<LivestockStatus, string> = {
  healthy:    'bg-secondary animate-bio-pulse',
  sick:       'bg-amber-400',
  quarantine: 'bg-primary',
  deceased:   'bg-on-surface-faint',
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

export function LivestockCard({ item }: LivestockCardProps) {
  const TypeIcon = typeIcons[item.type] ?? Box

  return (
    <Link
      to={`/livestock/${item.id}`}
      className="group relative block aspect-[4/3] rounded-2xl overflow-hidden shadow-abyss focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/50"
    >
      {/* Background: photo or tinted placeholder */}
      {item.image_path ? (
        <img
          src={`/data/${item.image_path}`}
          alt={item.name}
          className="absolute inset-0 w-full h-full object-cover transition-transform duration-500 ease-out group-hover:scale-105"
        />
      ) : (
        <div className={cn('absolute inset-0 flex items-center justify-center', typePlaceholderClass[item.type])}>
          <TypeIcon className={cn('h-16 w-16', typeIconClass[item.type])} />
        </div>
      )}

      {/* Subtle darkening vignette on hover */}
      <div className="absolute inset-0 bg-surface-container-lowest/0 group-hover:bg-surface-container-lowest/20 transition-fluid" />

      {/* Bottom glass overlay: identity */}
      <div className="absolute bottom-0 left-0 right-0 p-4 glass">
        <div className="font-semibold text-on-surface leading-tight truncate">{item.name}</div>
        {item.species && (
          <div className="text-xs text-on-surface-dim italic truncate mt-0.5">{item.species}</div>
        )}
        <div className="flex items-center gap-2 mt-2">
          <span className={cn('h-1.5 w-1.5 rounded-full flex-shrink-0', statusDotClass[item.status])} />
          <span className="text-xs text-on-surface-dim">{statusLabel[item.status]}</span>
          {item.quantity > 1 && (
            <span className="ml-auto text-xs text-on-surface-faint">×{item.quantity}</span>
          )}
        </div>
      </div>
    </Link>
  )
}
