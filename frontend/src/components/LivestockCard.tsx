import { useState } from 'react'
import { Fish, Flower2, Shell, Box, ChevronDown, ChevronUp, Pencil, Trash2 } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { cn, relativeTime } from '@/lib/utils'
import type { LivestockItem, LivestockType, LivestockStatus } from '@/api/types'
import { useLivestockObservations, useCreateLivestockObservation } from '@/hooks/useLivestock'

const typeIcons: Record<LivestockType, LucideIcon> = {
  fish:         Fish,
  coral:        Flower2,
  invertebrate: Shell,
  other:        Box,
}

const typeLabels: Record<LivestockType, string> = {
  fish:         'Fish',
  coral:        'Coral',
  invertebrate: 'Invertebrate',
  other:        'Other',
}

const statusDotClass: Record<LivestockStatus, string> = {
  healthy:    'bg-secondary animate-bio-pulse',
  sick:       'bg-amber-400',
  quarantine: 'bg-primary',
  deceased:   'bg-on-surface-faint',
}

const statusTextClass: Record<LivestockStatus, string> = {
  healthy:    'text-secondary',
  sick:       'text-amber-400',
  quarantine: 'text-primary',
  deceased:   'text-on-surface-faint',
}

interface LivestockCardProps {
  item: LivestockItem
  onEdit: (item: LivestockItem) => void
  onDelete: (id: number) => void
}

export function LivestockCard({ item, onEdit, onDelete }: LivestockCardProps) {
  const [obsOpen, setObsOpen] = useState(false)
  const [addingObs, setAddingObs] = useState(false)
  const [obsStatus, setObsStatus] = useState<LivestockStatus | ''>('')
  const [obsNote, setObsNote] = useState('')
  const [confirmDelete, setConfirmDelete] = useState(false)

  const { data: obsData, isLoading: obsLoading } = useLivestockObservations(item.id, obsOpen)
  const createObs = useCreateLivestockObservation()

  const TypeIcon = typeIcons[item.type] ?? Box
  const observations = obsData?.observations ?? []

  function handleSubmitObs(e: React.FormEvent) {
    e.preventDefault()
    if (!obsStatus && !obsNote.trim()) return
    createObs.mutate(
      {
        livestockId: item.id,
        data: {
          status: obsStatus || null,
          note: obsNote.trim() || null,
        },
      },
      {
        onSuccess: () => {
          setObsStatus('')
          setObsNote('')
          setAddingObs(false)
        },
      },
    )
  }

  return (
    <div className="bg-surface-container rounded-2xl p-5 flex flex-col gap-3">
      {/* Header row: image or icon + name + species */}
      <div className="flex items-start gap-3">
        {item.image_path ? (
          <img
            src={`/data/${item.image_path}`}
            alt={item.name}
            className="h-12 w-12 rounded-xl object-cover flex-shrink-0"
          />
        ) : (
          <div className="h-12 w-12 rounded-xl bg-primary/10 flex items-center justify-center flex-shrink-0">
            <TypeIcon className="h-5 w-5 text-primary" />
          </div>
        )}
        <div className="flex-1 min-w-0">
          <div className="font-semibold text-on-surface truncate">{item.name}</div>
          {item.species && (
            <div className="text-xs text-on-surface-dim italic truncate">{item.species}</div>
          )}
        </div>
        {/* Edit / Delete */}
        <div className="flex items-center gap-1 flex-shrink-0">
          <button
            onClick={() => onEdit(item)}
            className="p-1.5 rounded-lg text-on-surface-faint hover:text-primary hover:bg-primary/10 transition-fluid"
          >
            <Pencil className="h-3.5 w-3.5" />
          </button>
          {confirmDelete ? (
            <div className="flex items-center gap-1">
              <button
                onClick={() => onDelete(item.id)}
                className="text-xs px-2 py-1 rounded-lg bg-tertiary/20 text-tertiary hover:bg-tertiary/30 transition-fluid"
              >
                Confirm
              </button>
              <button
                onClick={() => setConfirmDelete(false)}
                className="text-xs px-2 py-1 rounded-lg bg-surface-container-high text-on-surface-dim hover:bg-surface-container-highest transition-fluid"
              >
                Cancel
              </button>
            </div>
          ) : (
            <button
              onClick={() => setConfirmDelete(true)}
              className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Type badge + Status badge */}
      <div className="flex items-center gap-2 flex-wrap">
        <span className="px-2 py-0.5 rounded-full bg-surface-container-high text-on-surface-dim text-xs font-medium">
          {typeLabels[item.type]}
        </span>
        <span className={cn('flex items-center gap-1.5 text-xs font-medium', statusTextClass[item.status])}>
          <span className={cn('h-2 w-2 rounded-full', statusDotClass[item.status])} />
          {item.status.charAt(0).toUpperCase() + item.status.slice(1)}
        </span>
        {item.quantity > 1 && (
          <span className="px-2 py-0.5 rounded-full bg-surface-container-high text-on-surface-dim text-xs font-medium">
            ×{item.quantity}
          </span>
        )}
      </div>

      {/* Meta: date added */}
      {item.date_added && (
        <div className="text-xs text-on-surface-faint">
          Added {item.date_added}
        </div>
      )}

      {/* Notes */}
      {item.notes && (
        <p className="text-xs text-on-surface-dim truncate">{item.notes}</p>
      )}

      {/* Observation drawer trigger */}
      <button
        onClick={() => setObsOpen((v) => !v)}
        className="flex items-center gap-1 text-xs text-on-surface-faint hover:text-on-surface-dim transition-fluid self-start"
      >
        {obsOpen ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
        {observations.length > 0
          ? `${observations.length} observation${observations.length === 1 ? '' : 's'}`
          : 'View log'}
      </button>

      {/* Observation timeline */}
      {obsOpen && (
        <div className="space-y-2">
          {obsLoading ? (
            <div className="h-8 bg-surface-container-high rounded-lg animate-pulse" />
          ) : observations.length === 0 ? (
            <p className="text-xs text-on-surface-faint">No observations yet.</p>
          ) : (
            <div className="space-y-1.5">
              {observations.map((obs) => (
                <div key={obs.id} className="flex items-start gap-2 text-xs">
                  <span className="text-on-surface-faint flex-shrink-0 pt-0.5">{relativeTime(obs.ts)}</span>
                  {obs.status && (
                    <span
                      className={cn(
                        'px-1.5 py-0.5 rounded-full font-medium flex-shrink-0',
                        statusTextClass[obs.status as LivestockStatus],
                        'bg-surface-container-high',
                      )}
                    >
                      {obs.status}
                    </span>
                  )}
                  {obs.note && <span className="text-on-surface-dim">{obs.note}</span>}
                </div>
              ))}
            </div>
          )}

          {/* Add observation form */}
          {addingObs ? (
            <form onSubmit={handleSubmitObs} className="space-y-2 pt-1">
              <select
                value={obsStatus}
                onChange={(e) => setObsStatus(e.target.value as LivestockStatus | '')}
                className="w-full bg-surface-container-highest text-on-surface text-xs rounded-lg px-2 py-1.5 outline-none"
              >
                <option value="">Status (optional)</option>
                <option value="healthy">Healthy</option>
                <option value="sick">Sick</option>
                <option value="quarantine">Quarantine</option>
                <option value="deceased">Deceased</option>
              </select>
              <textarea
                value={obsNote}
                onChange={(e) => setObsNote(e.target.value)}
                placeholder="Note (optional)"
                rows={2}
                className="w-full bg-surface-container-highest text-on-surface text-xs rounded-lg px-2 py-1.5 outline-none resize-none placeholder:text-on-surface-faint"
              />
              <div className="flex gap-2">
                <button
                  type="submit"
                  disabled={(!obsStatus && !obsNote.trim()) || createObs.isPending}
                  className="px-3 py-1 rounded-lg text-xs font-semibold bg-primary/20 text-primary hover:bg-primary/30 disabled:opacity-40 transition-fluid"
                >
                  Save
                </button>
                <button
                  type="button"
                  onClick={() => setAddingObs(false)}
                  className="px-3 py-1 rounded-lg text-xs font-semibold bg-surface-container-high text-on-surface-dim hover:bg-surface-container-highest transition-fluid"
                >
                  Cancel
                </button>
              </div>
            </form>
          ) : (
            <button
              onClick={() => setAddingObs(true)}
              className="text-xs text-primary/70 hover:text-primary transition-fluid"
            >
              + Add observation
            </button>
          )}
        </div>
      )}
    </div>
  )
}
