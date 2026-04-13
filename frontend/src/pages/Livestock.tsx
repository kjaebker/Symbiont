import { useState } from 'react'
import { Fish, Plus } from 'lucide-react'
import {
  useLivestock,
  useLivestockSpecies,
  useCreateLivestockItem,
} from '@/hooks/useLivestock'
import { usePageTitle } from '@/hooks/usePageTitle'
import { cn } from '@/lib/utils'
import type { LivestockType, LivestockStatus } from '@/api/types'
import { LivestockCard } from '@/components/LivestockCard'
import { LivestockForm, defaultForm, TYPE_OPTIONS, STATUS_OPTIONS } from '@/components/LivestockForm'
import type { LivestockFormData } from '@/components/LivestockForm'

export default function Livestock() {
  usePageTitle('Livestock')

  const { data, isLoading, isError } = useLivestock()
  const { data: speciesData } = useLivestockSpecies()
  const createItem = useCreateLivestockItem()

  const [typeFilter, setTypeFilter] = useState<LivestockType | ''>('')
  const [statusFilter, setStatusFilter] = useState<LivestockStatus | ''>('')
  const [showAddForm, setShowAddForm] = useState(false)

  const allItems = data?.livestock ?? []
  const speciesSuggestions = speciesData?.species ?? []

  const filtered = allItems.filter((item) => {
    if (typeFilter && item.type !== typeFilter) return false
    if (statusFilter && item.status !== statusFilter) return false
    return true
  })

  function countType(type: LivestockType) {
    return allItems.filter((i) => i.type === type).reduce((sum, i) => sum + i.quantity, 0)
  }
  const nonHealthyCount = allItems.filter((i) => i.status !== 'healthy').length

  async function handleCreate(form: LivestockFormData): Promise<number> {
    const item = await createItem.mutateAsync({
      name:       form.name.trim(),
      species:    form.species.trim() || null,
      type:       form.type,
      quantity:   form.quantity,
      status:     form.status,
      date_added: form.date_added || null,
      notes:      form.notes.trim() || null,
    })
    return item.id
  }

  return (
    <div className="p-6 md:p-8 max-w-6xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-xs text-primary uppercase tracking-widest mb-2">Tank Biology</p>
          <h1 className="text-3xl md:text-4xl font-bold text-on-surface tracking-tight">
            Livestock
          </h1>
        </div>
        <button
          onClick={() => setShowAddForm((v) => !v)}
          className={cn(
            'flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-semibold transition-fluid',
            showAddForm
              ? 'bg-primary/20 text-primary'
              : 'bg-surface-container text-on-surface-dim hover:bg-surface-container-high hover:text-on-surface',
          )}
        >
          <Plus className="h-4 w-4" />
          Add Livestock
        </button>
      </div>

      {/* Add form */}
      {showAddForm && (
        <LivestockForm
          initial={defaultForm()}
          speciesSuggestions={speciesSuggestions}
          onSubmit={handleCreate}
          onClose={() => setShowAddForm(false)}
          title="Add Livestock"
        />
      )}

      {/* Summary chips */}
      {allItems.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {countType('fish') > 0 && (
            <span className="px-3 py-1.5 rounded-full bg-primary/10 text-primary text-xs font-medium">
              {countType('fish')} Fish
            </span>
          )}
          {countType('coral') > 0 && (
            <span className="px-3 py-1.5 rounded-full bg-tertiary/10 text-tertiary text-xs font-medium">
              {countType('coral')} Corals
            </span>
          )}
          {countType('invertebrate') > 0 && (
            <span className="px-3 py-1.5 rounded-full bg-secondary/10 text-secondary text-xs font-medium">
              {countType('invertebrate')} Invertebrates
            </span>
          )}
          {countType('other') > 0 && (
            <span className="px-3 py-1.5 rounded-full bg-surface-container-high text-on-surface-dim text-xs font-medium">
              {countType('other')} Other
            </span>
          )}
          {nonHealthyCount > 0 && (
            <span className="px-3 py-1.5 rounded-full bg-amber-400/10 text-amber-400 text-xs font-medium">
              {nonHealthyCount} Need attention
            </span>
          )}
        </div>
      )}

      {/* Filter bar */}
      <div className="space-y-2">
        <div className="flex flex-wrap gap-2">
          <button
            onClick={() => setTypeFilter('')}
            className={cn(
              'px-3 py-1.5 rounded-full text-xs font-medium transition-fluid',
              typeFilter === ''
                ? 'bg-primary/20 text-primary'
                : 'bg-surface-container text-on-surface-dim hover:bg-surface-container-high',
            )}
          >
            All Types
          </button>
          {TYPE_OPTIONS.map((o) => (
            <button
              key={o.value}
              onClick={() => setTypeFilter(typeFilter === o.value ? '' : o.value)}
              className={cn(
                'px-3 py-1.5 rounded-full text-xs font-medium transition-fluid',
                typeFilter === o.value
                  ? 'bg-primary/20 text-primary'
                  : 'bg-surface-container text-on-surface-dim hover:bg-surface-container-high',
              )}
            >
              {o.label}
            </button>
          ))}
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            onClick={() => setStatusFilter('')}
            className={cn(
              'px-3 py-1.5 rounded-full text-xs font-medium transition-fluid',
              statusFilter === ''
                ? 'bg-secondary/20 text-secondary'
                : 'bg-surface-container text-on-surface-dim hover:bg-surface-container-high',
            )}
          >
            All Status
          </button>
          {STATUS_OPTIONS.map((o) => (
            <button
              key={o.value}
              onClick={() => setStatusFilter(statusFilter === o.value ? '' : o.value)}
              className={cn(
                'px-3 py-1.5 rounded-full text-xs font-medium transition-fluid',
                statusFilter === o.value
                  ? 'bg-secondary/20 text-secondary'
                  : 'bg-surface-container text-on-surface-dim hover:bg-surface-container-high',
              )}
            >
              {o.label}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      {isLoading ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          {[...Array(4)].map((_, i) => (
            <div key={i} className="aspect-[4/3] bg-surface-container rounded-2xl animate-pulse" />
          ))}
        </div>
      ) : isError ? (
        <div className="text-center py-16 text-on-surface-faint">
          Failed to load livestock.
        </div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-16 space-y-4">
          <Fish className="h-12 w-12 text-on-surface-faint mx-auto" />
          <p className="text-on-surface-dim">
            {allItems.length === 0 ? 'Your tank is waiting for its first resident.' : 'No livestock match the current filters.'}
          </p>
          {allItems.length === 0 && (
            <button
              onClick={() => setShowAddForm(true)}
              className="px-4 py-2 rounded-xl text-sm font-semibold bg-primary/20 text-primary hover:bg-primary/30 transition-fluid"
            >
              Add your first resident
            </button>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          {filtered.map((item) => (
            <LivestockCard key={item.id} item={item} />
          ))}
        </div>
      )}
    </div>
  )
}
