import { useState, useRef } from 'react'
import { Fish, Plus, X, ImagePlus } from 'lucide-react'
import {
  useLivestock,
  useLivestockSpecies,
  useCreateLivestockItem,
  useUpdateLivestockItem,
  useDeleteLivestockItem,
  useUploadLivestockImage,
} from '@/hooks/useLivestock'
import { usePageTitle } from '@/hooks/usePageTitle'
import { cn } from '@/lib/utils'
import type { LivestockItem, LivestockType, LivestockStatus } from '@/api/types'
import { LivestockCard } from '@/components/LivestockCard'

const TYPE_OPTIONS: { value: LivestockType; label: string }[] = [
  { value: 'fish', label: 'Fish' },
  { value: 'coral', label: 'Coral' },
  { value: 'invertebrate', label: 'Invertebrate' },
  { value: 'other', label: 'Other' },
]

const STATUS_OPTIONS: { value: LivestockStatus; label: string }[] = [
  { value: 'healthy', label: 'Healthy' },
  { value: 'sick', label: 'Sick' },
  { value: 'quarantine', label: 'Quarantine' },
  { value: 'deceased', label: 'Deceased' },
]

function todayString(): string {
  return new Date().toISOString().slice(0, 10)
}

interface FormData {
  name: string
  species: string
  type: LivestockType
  quantity: number
  status: LivestockStatus
  date_added: string
  notes: string
}

const defaultForm = (): FormData => ({
  name: '',
  species: '',
  type: 'fish',
  quantity: 1,
  status: 'healthy',
  date_added: todayString(),
  notes: '',
})

interface LivestockFormProps {
  initial?: FormData
  currentImagePath?: string | null
  speciesSuggestions: string[]
  onSubmit: (data: FormData) => Promise<number>
  onClose: () => void
  title: string
}

function LivestockForm({ initial, currentImagePath, speciesSuggestions, onSubmit, onClose, title }: LivestockFormProps) {
  const [form, setForm] = useState<FormData>(initial ?? defaultForm())
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [imageFile, setImageFile] = useState<File | null>(null)
  const [imagePreview, setImagePreview] = useState<string | null>(null)
  const imageInputRef = useRef<HTMLInputElement>(null)
  const uploadImage = useUploadLivestockImage()

  function set<K extends keyof FormData>(key: K, value: FormData[K]) {
    setForm((f) => ({ ...f, [key]: value }))
  }

  function handleImageChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    if (imagePreview) URL.revokeObjectURL(imagePreview)
    setImageFile(file)
    setImagePreview(URL.createObjectURL(file))
    e.target.value = ''
  }

  function clearImage() {
    if (imagePreview) URL.revokeObjectURL(imagePreview)
    setImageFile(null)
    setImagePreview(null)
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (!form.name.trim()) {
      setError('Name is required.')
      return
    }
    setSubmitting(true)
    try {
      const itemId = await onSubmit(form)
      if (imageFile) {
        await uploadImage.mutateAsync({ id: itemId, file: imageFile })
      }
      onClose()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Something went wrong.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="bg-surface-container rounded-2xl p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-on-surface">{title}</h2>
        <button
          onClick={onClose}
          className="p-1.5 rounded-lg text-on-surface-faint hover:text-on-surface hover:bg-surface-container-high transition-fluid"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      <form onSubmit={handleSubmit} className="space-y-3">
        {/* Name */}
        <div>
          <label className="block text-xs text-on-surface-dim uppercase tracking-wider mb-1">
            Name <span className="text-tertiary">*</span>
          </label>
          <input
            type="text"
            value={form.name}
            onChange={(e) => set('name', e.target.value)}
            placeholder="e.g. Ocellaris Clownfish"
            className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-sm outline-none placeholder:text-on-surface-faint"
          />
        </div>

        {/* Species (combobox via datalist) */}
        <div>
          <label className="block text-xs text-on-surface-dim uppercase tracking-wider mb-1">
            Species
          </label>
          <input
            type="text"
            list="livestock-species-list"
            value={form.species}
            onChange={(e) => set('species', e.target.value)}
            placeholder="e.g. Amphiprion ocellaris"
            className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-sm outline-none placeholder:text-on-surface-faint"
          />
          <datalist id="livestock-species-list">
            {speciesSuggestions.map((s) => (
              <option key={s} value={s} />
            ))}
          </datalist>
        </div>

        {/* Type + Status row */}
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs text-on-surface-dim uppercase tracking-wider mb-1">
              Type
            </label>
            <select
              value={form.type}
              onChange={(e) => set('type', e.target.value as LivestockType)}
              className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-sm outline-none"
            >
              {TYPE_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs text-on-surface-dim uppercase tracking-wider mb-1">
              Status
            </label>
            <select
              value={form.status}
              onChange={(e) => set('status', e.target.value as LivestockStatus)}
              className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-sm outline-none"
            >
              {STATUS_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </select>
          </div>
        </div>

        {/* Quantity + Date Added row */}
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs text-on-surface-dim uppercase tracking-wider mb-1">
              Quantity
            </label>
            <input
              type="number"
              min={1}
              value={form.quantity}
              onChange={(e) => set('quantity', Math.max(1, parseInt(e.target.value) || 1))}
              className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-sm outline-none"
            />
          </div>
          <div>
            <label className="block text-xs text-on-surface-dim uppercase tracking-wider mb-1">
              Date Added
            </label>
            <input
              type="date"
              value={form.date_added}
              onChange={(e) => set('date_added', e.target.value)}
              className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-sm outline-none"
            />
          </div>
        </div>

        {/* Notes */}
        <div>
          <label className="block text-xs text-on-surface-dim uppercase tracking-wider mb-1">
            Notes
          </label>
          <textarea
            value={form.notes}
            onChange={(e) => set('notes', e.target.value)}
            placeholder="Optional notes"
            rows={2}
            className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-sm outline-none resize-none placeholder:text-on-surface-faint"
          />
        </div>

        {/* Photo */}
        <div>
          <label className="block text-xs text-on-surface-dim uppercase tracking-wider mb-1">
            Photo
          </label>
          {imagePreview ? (
            <div className="relative w-24 h-24">
              <img src={imagePreview} alt="Preview" className="w-24 h-24 rounded-xl object-cover" />
              <button
                type="button"
                onClick={clearImage}
                className="absolute -top-1.5 -right-1.5 p-0.5 rounded-full bg-surface-container-highest text-on-surface-faint hover:text-tertiary transition-fluid"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
          ) : currentImagePath ? (
            <div className="flex items-center gap-3">
              <img src={`/data/${currentImagePath}`} alt="Current" className="w-24 h-24 rounded-xl object-cover" />
              <button
                type="button"
                onClick={() => imageInputRef.current?.click()}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium bg-surface-container-high text-on-surface-dim hover:bg-surface-container-highest transition-fluid"
              >
                <ImagePlus className="h-3.5 w-3.5" />
                Replace
              </button>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => imageInputRef.current?.click()}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium bg-surface-container-high text-on-surface-dim hover:bg-surface-container-highest transition-fluid"
            >
              <ImagePlus className="h-3.5 w-3.5" />
              Attach photo
            </button>
          )}
          <input
            ref={imageInputRef}
            type="file"
            accept="image/*"
            className="hidden"
            onChange={handleImageChange}
          />
        </div>

        {error && <p className="text-xs text-tertiary">{error}</p>}

        <div className="flex gap-2 pt-1">
          <button
            type="submit"
            disabled={submitting}
            className="px-4 py-2 rounded-xl text-sm font-semibold bg-primary/20 text-primary hover:bg-primary/30 disabled:opacity-40 transition-fluid"
          >
            {submitting ? 'Saving…' : 'Save'}
          </button>
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 rounded-xl text-sm font-semibold bg-surface-container-high text-on-surface-dim hover:bg-surface-container-highest transition-fluid"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  )
}

function itemToFormData(item: LivestockItem): FormData {
  return {
    name:       item.name,
    species:    item.species ?? '',
    type:       item.type,
    quantity:   item.quantity,
    status:     item.status,
    date_added: item.date_added ?? todayString(),
    notes:      item.notes ?? '',
  }
}

export default function Livestock() {
  usePageTitle('Livestock')

  const { data, isLoading, isError } = useLivestock()
  const { data: speciesData } = useLivestockSpecies()
  const createItem = useCreateLivestockItem()
  const updateItem = useUpdateLivestockItem()
  const deleteItem = useDeleteLivestockItem()

  const [typeFilter, setTypeFilter] = useState<LivestockType | ''>('')
  const [statusFilter, setStatusFilter] = useState<LivestockStatus | ''>('')
  const [showAddForm, setShowAddForm] = useState(false)
  const [editing, setEditing] = useState<LivestockItem | null>(null)

  const allItems = data?.livestock ?? []
  const speciesSuggestions = speciesData?.species ?? []

  // Filtered list for display; stats computed from full list.
  const filtered = allItems.filter((item) => {
    if (typeFilter && item.type !== typeFilter) return false
    if (statusFilter && item.status !== statusFilter) return false
    return true
  })

  function countType(type: LivestockType) {
    return allItems.filter((i) => i.type === type).reduce((sum, i) => sum + i.quantity, 0)
  }
  const nonHealthyCount = allItems.filter((i) => i.status !== 'healthy').length

  async function handleCreate(form: FormData): Promise<number> {
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

  async function handleUpdate(form: FormData): Promise<number> {
    if (!editing) return 0
    await updateItem.mutateAsync({
      id: editing.id,
      data: {
        name:       form.name.trim(),
        species:    form.species.trim() || null,
        type:       form.type,
        quantity:   form.quantity,
        status:     form.status,
        date_added: form.date_added || null,
        notes:      form.notes.trim() || null,
      },
    })
    setEditing(null)
    return editing.id
  }

  function handleDelete(id: number) {
    deleteItem.mutate(id)
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
          onClick={() => { setShowAddForm((v) => !v); setEditing(null) }}
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
          speciesSuggestions={speciesSuggestions}
          onSubmit={handleCreate}
          onClose={() => setShowAddForm(false)}
          title="Add Livestock"
        />
      )}

      {/* Edit form */}
      {editing && (
        <LivestockForm
          initial={itemToFormData(editing)}
          currentImagePath={editing.image_path}
          speciesSuggestions={speciesSuggestions}
          onSubmit={handleUpdate}
          onClose={() => setEditing(null)}
          title={`Edit: ${editing.name}`}
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
              {nonHealthyCount} Non-healthy
            </span>
          )}
        </div>
      )}

      {/* Filter bar */}
      <div className="space-y-2">
        {/* Type filters */}
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
        {/* Status filters */}
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
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {[...Array(3)].map((_, i) => (
            <div key={i} className="h-48 bg-surface-container rounded-2xl animate-pulse" />
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
            {allItems.length === 0 ? 'No livestock yet.' : 'No livestock match the current filters.'}
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
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {filtered.map((item) => (
            <LivestockCard
              key={item.id}
              item={item}
              onEdit={(i) => { setEditing(i); setShowAddForm(false) }}
              onDelete={handleDelete}
            />
          ))}
        </div>
      )}
    </div>
  )
}
