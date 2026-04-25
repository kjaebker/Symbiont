import { Utensils, X } from 'lucide-react'
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
    <div
      className="rounded-2xl p-5 transition-fluid"
      style={{ background: `linear-gradient(145deg, var(--color-surface-container) 20%, rgba(58,223,250,${isActive ? '0.18' : '0.10'}))` }}
    >
      {/* Header: icon + name + status */}
      <div className="flex items-start justify-between mb-1">
        <div className="flex items-center gap-3">
          <div
            className="w-10 h-10 flex items-center justify-center shrink-0 transition-fluid"
            style={{
              borderRadius: '50% 60% 40% 60% / 60% 50% 60% 40%',
              background: isActive ? 'rgba(58,223,250,0.12)' : 'rgba(58,223,250,0.06)',
              boxShadow: isActive ? '0 0 14px rgba(58,223,250,0.20)' : 'none',
            }}
          >
            <Utensils
              size={18}
              className={isActive ? 'text-primary' : 'text-on-surface-faint'}
            />
          </div>
          <span className="text-sm font-semibold text-on-surface">Feed Mode</span>
        </div>

        <div className="flex items-center gap-2 mt-0.5">
          {isActive && !isError && (
            <span className="flex h-2 w-2 relative shrink-0">
              <span className="animate-bio-pulse absolute inline-flex h-full w-full rounded-full bg-primary opacity-75" />
              <span className="relative inline-flex rounded-full h-2 w-2 bg-primary" />
            </span>
          )}
          <span
            className={cn(
              'text-xs font-semibold uppercase tracking-wider shrink-0',
              isError
                ? 'text-tertiary'
                : isActive
                  ? 'text-primary'
                  : 'text-on-surface-faint',
            )}
          >
            {isError ? 'Unavailable' : isActive ? activeFeedLabel : 'Inactive'}
          </span>
        </div>
      </div>

      {/* Controls — revealed when unlocked */}
      <div
        className="grid overflow-hidden"
        style={{
          gridTemplateRows: controlsLocked ? '0fr' : '1fr',
          transition: 'grid-template-rows 250ms cubic-bezier(0.65, 0, 0.35, 1)',
        }}
      >
        <div className="min-h-0">
          <div className="pt-4 space-y-1.5">
            <div className="flex gap-1.5">
              {FEED_CYCLES.map(({ name, label }) => (
                <button
                  key={name}
                  onClick={() => handleStart(name)}
                  disabled={mutation.isPending}
                  className={cn(
                    'flex-1 py-2 rounded-xl text-xs font-semibold uppercase tracking-wider transition-fluid',
                    isActive && activeFeed === name
                      ? 'bg-primary text-on-primary'
                      : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim hover:bg-surface-container-highest',
                  )}
                >
                  {label}
                </button>
              ))}
            </div>

            {isActive && (
              <button
                onClick={handleCancel}
                disabled={mutation.isPending}
                className="w-full py-2 rounded-xl text-xs font-semibold uppercase tracking-wider transition-fluid flex items-center justify-center gap-1.5 bg-tertiary/15 text-tertiary hover:bg-tertiary/25"
              >
                <X size={12} />
                Cancel Feed
              </button>
            )}

            {mutation.isError && (
              <p className="text-xs text-tertiary">Failed to set feed mode. Try again.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
