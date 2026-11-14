import { useState, useRef } from 'react'
import { X, ImagePlus } from 'lucide-react'
import type { LivestockType, LivestockStatus } from '@/api/types'
import { useUploadLivestockImage } from '@/hooks/useLivestock'
import { Button, IconButton, Card, Field, Input, Select, Textarea, SectionHeader, labelClass } from '@/components/ui'
import { cn } from '@/lib/utils'

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
    <Card className="p-6 space-y-4">
      <SectionHeader
        title={title}
        action={<IconButton icon={X} label="Close" size={16} onClick={onClose} />}
      />

      <form onSubmit={handleSubmit} className="space-y-3">
        <Field label="Name *">
          <Input
            type="text"
            value={form.name}
            onChange={(e) => set('name', e.target.value)}
            placeholder="e.g. Ocellaris Clownfish"
          />
        </Field>

        <Field label="Species">
          <Input
            type="text"
            list="livestock-species-list"
            value={form.species}
            onChange={(e) => set('species', e.target.value)}
            placeholder="e.g. Amphiprion ocellaris"
          />
          <datalist id="livestock-species-list">
            {speciesSuggestions.map((s) => (
              <option key={s} value={s} />
            ))}
          </datalist>
        </Field>

        <div className="grid grid-cols-2 gap-3">
          <Field label="Type">
            <Select
              value={form.type}
              onChange={(e) => set('type', e.target.value as LivestockType)}
            >
              {TYPE_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </Select>
          </Field>
          <Field label="Status">
            <Select
              value={form.status}
              onChange={(e) => set('status', e.target.value as LivestockStatus)}
            >
              {STATUS_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>{o.label}</option>
              ))}
            </Select>
          </Field>
        </div>

        <div className="grid grid-cols-2 gap-3">
          <Field label="Quantity">
            <Input
              type="number"
              min={1}
              value={form.quantity}
              onChange={(e) => set('quantity', Math.max(1, parseInt(e.target.value) || 1))}
            />
          </Field>
          <Field label="Date Added">
            <Input
              type="date"
              value={form.date_added}
              onChange={(e) => set('date_added', e.target.value)}
            />
          </Field>
        </div>

        <Field label="Notes">
          <Textarea
            value={form.notes}
            onChange={(e) => set('notes', e.target.value)}
            placeholder="Optional notes"
            rows={2}
          />
        </Field>

        {/* Photo */}
        <div>
          <span className={cn(labelClass, 'block mb-1.5')}>Photo</span>
          {imagePreview ? (
            <div className="relative w-24 h-24">
              <img src={imagePreview} alt="Preview" className="w-24 h-24 rounded-xl object-cover" />
              <button
                type="button"
                onClick={clearImage}
                aria-label="Remove photo"
                className="absolute -top-1.5 -right-1.5 p-0.5 rounded-full bg-surface-container-highest text-on-surface-faint hover:text-tertiary transition-fluid cursor-pointer"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
          ) : currentImagePath ? (
            <div className="flex items-center gap-3">
              <img src={`/${currentImagePath}`} alt="Current" className="w-24 h-24 rounded-xl object-cover" />
              <Button size="sm" icon={ImagePlus} onClick={() => imageInputRef.current?.click()}>
                Replace
              </Button>
            </div>
          ) : (
            <Button size="sm" icon={ImagePlus} onClick={() => imageInputRef.current?.click()}>
              Attach photo
            </Button>
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
          <Button type="submit" variant="primary" loading={submitting}>
            {submitting ? 'Saving…' : 'Save'}
          </Button>
          <Button onClick={onClose}>Cancel</Button>
        </div>
      </form>
    </Card>
  )
}
