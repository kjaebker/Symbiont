import { Lock, Utensils, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { useFeedStatus, useSetFeedMode } from '@/hooks/useFeed'

const FEED_CYCLES = [
  { name: 1, label: 'A' },
  { name: 2, label: 'B' },
  { name: 3, label: 'C' },
  { name: 4, label: 'D' },
] as const

interface FeedCardProps {
  controlsLocked?: boolean
}

export function FeedCard({ controlsLocked = false }: FeedCardProps) {
  const { data, isError } = useFeedStatus()
  const mutation = useSetFeedMode()

  const isActive = (data?.active ?? 0) === 1
  const activeFeed = data?.name ?? 0

  function handleStart(name: number) {
    if (controlsLocked) return
    mutation.mutate({ name, active: true })
  }

  function handleCancel() {
    if (controlsLocked) return
    mutation.mutate({ name: 0, active: false })
  }

  const activeFeedLabel = activeFeed >= 1 && activeFeed <= 4
    ? `Feed ${['A', 'B', 'C', 'D'][activeFeed - 1]} Active`
    : 'Active'

  return (
    <div className="bg-surface-container rounded-2xl p-5 transition-fluid">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div
            className={cn(
              'w-10 h-10 rounded-xl flex items-center justify-center',
              isActive ? 'bg-primary/10' : 'bg-surface-container-highest',
            )}
          >
            <Utensils
              size={18}
              className={isActive ? 'text-primary' : 'text-on-surface-faint'}
            />
          </div>
          <div>
            <div className="text-sm font-semibold text-on-surface">Feed Mode</div>
            <div
              className={cn(
                'text-xs font-medium uppercase tracking-wider',
                isError
                  ? 'text-tertiary'
                  : isActive
                    ? 'text-primary'
                    : 'text-on-surface-dim',
              )}
            >
              {isError ? 'Unavailable' : isActive ? activeFeedLabel : 'Inactive'}
            </div>
          </div>
        </div>

        {isActive && !isError && (
          <span className="flex h-2 w-2 relative mt-2 shrink-0">
            <span className="animate-bio-pulse absolute inline-flex h-full w-full rounded-full bg-primary opacity-75" />
            <span className="relative inline-flex rounded-full h-2 w-2 bg-primary" />
          </span>
        )}
      </div>

      <div className={cn('relative space-y-1.5', controlsLocked && 'opacity-40')}>
        <div className="flex gap-1.5">
          {FEED_CYCLES.map(({ name, label }) => (
            <button
              key={name}
              onClick={() => handleStart(name)}
              disabled={mutation.isPending || controlsLocked}
              className={cn(
                'flex-1 py-1.5 rounded-lg text-xs font-semibold uppercase tracking-wider transition-fluid',
                controlsLocked
                  ? 'bg-surface-container-high text-on-surface-faint cursor-not-allowed'
                  : isActive && activeFeed === name
                    ? 'bg-primary text-on-primary'
                    : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim',
              )}
            >
              {label}
            </button>
          ))}
        </div>

        {isActive && (
          <button
            onClick={handleCancel}
            disabled={mutation.isPending || controlsLocked}
            className={cn(
              'w-full py-1.5 rounded-lg text-xs font-semibold uppercase tracking-wider transition-fluid flex items-center justify-center gap-1.5',
              controlsLocked
                ? 'bg-surface-container-high text-on-surface-faint cursor-not-allowed'
                : 'bg-tertiary/15 text-tertiary hover:bg-tertiary/25',
            )}
          >
            <X size={12} />
            Cancel Feed
          </button>
        )}

        {controlsLocked && (
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
            <Lock size={14} className="text-on-surface-dim" />
          </div>
        )}
      </div>

      {mutation.isError && !controlsLocked && (
        <p className="text-xs text-tertiary mt-2">Failed to set feed mode. Try again.</p>
      )}
    </div>
  )
}
