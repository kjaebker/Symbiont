import { useState, useRef } from 'react'
import { X, ImagePlus } from 'lucide-react'
import type { LivestockType, LivestockStatus } from '@/api/types'
import { useUploadLivestockImage } from '@/hooks/useLivestock'

export const TYPE_OPTIONS: { value: LivestockType; label: string }[] = [
  { value: 'fish', label: 'Fish' },
  { value: 'coral', label: 'Coral' },
  { value: 'invertebrate', label: 'Invertebrate' },
  { value: 'other', label: 'Other' },
]

export const STATUS_OPTIONS: { value: LivestockStatus; label: string }[] = [
  { value: 'healthy', label: 'Healthy' },
  { value: 'sick', label: 'Sick' },
  { value: 'quarantine', label: 'Quarantine' },
  { value: 'deceased', label: 'Deceased' },
]

export function todayString(): string {
  return new Date().toISOString().slice(0, 10)
}

export interface LivestockFormData {
  name: string
  species: string
  type: LivestockType
  quantity: number
  status: LivestockStatus
  date_added: string
  notes: string
}

export const defaultForm = (): LivestockFormData => ({
  name: '',
  species: '',
  type: 'fish',
  quantity: 1,
  status: 'healthy',
  date_added: todayString(),
  notes: '',
})

interface LivestockFormProps {
  initial?: LivestockFormData
  currentImagePath?: string | null
  speciesSuggestions: string[]
  onSubmit: (data: LivestockFormData) => Promise<number>
  onClose: () => void
  title: string
}

export function LivestockForm({ initial, currentImagePath, speciesSuggestions, onSubmit, onClose, title }: LivestockFormProps) {
  const [form, setForm] = useState<LivestockFormData>(initial ?? defaultForm())
  const [error, setError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [imageFile, setImageFile] = useState<File | null>(null)
  const [imagePreview, setImagePreview] = useState<string | null>(null)
  const imageInputRef = useRef<HTMLInputElement>(null)
  const uploadImage = useUploadLivestockImage()

  function set<K extends keyof LivestockFormData>(key: K, value: LivestockFormData[K]) {
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
            className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-base outline-none placeholder:text-on-surface-faint"
          />
        </div>

        {/* Species */}
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
            className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-base outline-none placeholder:text-on-surface-faint"
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
              className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-base outline-none"
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
              className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-base outline-none"
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
              className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-base outline-none"
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
              className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-base outline-none"
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
            className="w-full bg-surface-container-highest text-on-surface rounded-xl px-3 py-2 text-base outline-none resize-none placeholder:text-on-surface-faint"
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
              <img src={`/${currentImagePath}`} alt="Current" className="w-24 h-24 rounded-xl object-cover" />
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
