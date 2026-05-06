import { useState } from 'react'
import { Fish, Plus } from 'lucide-react'
import {
  useLivestock,
  useLivestockSpecies,
  useCreateLivestockItem,
} from '@/hooks/useLivestock'
import { usePageTitle } from '@/hooks/usePageTitle'
import type { LivestockType, LivestockStatus } from '@/api/types'
import { LivestockCard } from '@/components/LivestockCard'
import { LivestockForm, defaultForm, TYPE_OPTIONS, STATUS_OPTIONS } from '@/components/LivestockForm'
import { PageHeader } from '@/components/PageHeader'
import { FilterDropdown } from '@/components/FilterDropdown'
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

  const filterBar = (
    <>
      <FilterDropdown
        label="Type"
        value={typeFilter}
        options={[{ value: '', label: 'All types' }, ...TYPE_OPTIONS]}
        onChange={(v) => setTypeFilter(v as LivestockType | '')}
      />
      <FilterDropdown
        label="Status"
        value={statusFilter}
        options={[{ value: '', label: 'All status' }, ...STATUS_OPTIONS]}
        onChange={(v) => setStatusFilter(v as LivestockStatus | '')}
      />
    </>
  )

  const statusPills = allItems.length > 0 ? (
    <>
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
    </>
  ) : null

  return (
    <div>
      <PageHeader
        subtitle="Tank Biology"
        title="Livestock"
        action={{ label: 'Add Livestock', icon: Plus, onClick: () => setShowAddForm((v) => !v), active: showAddForm }}
        filterBar={filterBar}
        filterCount={2}
        filterActive={!!(typeFilter || statusFilter)}
        onClearFilters={() => { setTypeFilter(''); setStatusFilter('') }}
        statusPills={statusPills ?? undefined}
        maxWidth="max-w-6xl"
      />

      <div className="px-6 md:px-8 pt-5 pb-8 max-w-6xl mx-auto space-y-6">
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
    </div>
  )
}
