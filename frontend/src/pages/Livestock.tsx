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
import { cn } from '@/lib/utils'
import { Button, Chip, PageBody, LoadingState, EmptyState } from '@/components/ui'
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

  // Deceased entries are kept as a record but are not part of the tank: they
  // are left out of the counts and only listed when asked for by status.
  const living = allItems.filter((i) => i.status !== 'deceased')
  const deceasedCount = allItems.length - living.length

  const filtered = (statusFilter === 'deceased' ? allItems : living).filter((item) => {
    if (typeFilter && item.type !== typeFilter) return false
    if (statusFilter && item.status !== statusFilter) return false
    return true
  })

  function countType(type: LivestockType) {
    return living.filter((i) => i.type === type).reduce((sum, i) => sum + i.quantity, 0)
  }
  const needAttentionCount = living.filter((i) => i.status !== 'healthy').length

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
        options={[{ value: '', label: 'All living' }, ...STATUS_OPTIONS]}
        onChange={(v) => setStatusFilter(v as LivestockStatus | '')}
      />
    </>
  )

  const statusPills = allItems.length > 0 ? (
    <>
      {countType('fish') > 0 && (
        <Chip tone="primary">{countType('fish')} Fish</Chip>
      )}
      {countType('coral') > 0 && (
        <Chip tone="critical">{countType('coral')} Corals</Chip>
      )}
      {countType('invertebrate') > 0 && (
        <Chip tone="secondary">{countType('invertebrate')} Invertebrates</Chip>
      )}
      {countType('other') > 0 && (
        <Chip>{countType('other')} Other</Chip>
      )}
      {needAttentionCount > 0 && (
        <Chip tone="warning">{needAttentionCount} Need attention</Chip>
      )}
      {deceasedCount > 0 && (
        <button
          onClick={() => setStatusFilter(statusFilter === 'deceased' ? '' : 'deceased')}
          className={cn(
            'px-3 py-1.5 rounded-full text-xs font-medium transition-fluid cursor-pointer',
            statusFilter === 'deceased'
              ? 'bg-surface-container-highest text-on-surface'
              : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim',
          )}
        >
          {deceasedCount} Deceased
        </button>
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
      />

      <PageBody>
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
        <LoadingState label="Loading livestock" />
      ) : isError ? (
        <EmptyState message="Failed to load livestock." />
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={<Fish className="h-12 w-12" />}
          message={allItems.length === 0 ? 'Your tank is waiting for its first resident.' : 'No livestock match the current filters.'}
          action={allItems.length === 0 && (
            <Button variant="tonal" onClick={() => setShowAddForm(true)}>
              Add your first resident
            </Button>
          )}
        />
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4">
          {filtered.map((item) => (
            <LivestockCard key={item.id} item={item} />
          ))}
        </div>
      )}
      </PageBody>
    </div>
  )
}
