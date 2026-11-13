import { useState, useRef, useEffect, useCallback } from 'react'
import { ConfirmButton } from '@/components/ConfirmButton'
import { useParams, useNavigate, Link } from 'react-router-dom'
import {
  Fish, Flower2, Shell, Box,
  ArrowLeft, Pencil, Trash2, X,
  Camera, ImagePlus, Plus,
  ChevronLeft, ChevronRight, ZoomIn, Crop, History,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { LivestockStatus, LivestockType } from '@/api/types'
import {
  useLivestockItem,
  useLivestockObservations,
  useUpdateLivestockItem,
  useDeleteLivestockItem,
  useUploadLivestockImage,
  useEditLivestockImage,
  useResetLivestockImage,
  useDeleteLivestockImage,
  useCreateLivestockObservation,
  useUploadObservationImage,
  useDeleteObservationImage,
  useEditObservationImage,
  useResetObservationImage,
  useLivestockSpecies,
} from '@/hooks/useLivestock'
import { LivestockForm } from '@/components/LivestockForm'
import { ImageEditModal } from '@/components/ImageEditModal'
import type { LivestockFormData } from '@/components/LivestockForm'
import { usePageTitle } from '@/hooks/usePageTitle'
import { thumbUrl, originalUrl } from '@/api/client'
import {
  Button, IconButton, Card, Chip, SectionHeader, LoadingState, EmptyState, PageBody,
  Select, Textarea, labelClass, TimeAgo,
} from '@/components/ui'
import type { ChipTone } from '@/components/ui'

// ─── type maps ────────────────────────────────────────────────────────────────

const typeIcons: Record<LivestockType, LucideIcon> = {
  fish: Fish, coral: Flower2, invertebrate: Shell, other: Box,
}

const typePlaceholderClass: Record<LivestockType, string> = {
  fish:         'bg-primary/10',
  coral:        'bg-tertiary/10',
  invertebrate: 'bg-secondary/10',
  other:        'bg-surface-container-high',
}

const typeIconClass: Record<LivestockType, string> = {
  fish:         'text-primary/25',
  coral:        'text-tertiary/25',
  invertebrate: 'text-secondary/25',
  other:        'text-on-surface-faint/25',
}

const typeLabel: Record<LivestockType, string> = {
  fish: 'Fish', coral: 'Coral', invertebrate: 'Invertebrate', other: 'Other',
}

const statusDotClass: Record<LivestockStatus, string> = {
  healthy:    'bg-secondary animate-bio-pulse',
  sick:       'bg-warning',
  quarantine: 'bg-primary',
  deceased:   'bg-on-surface-faint',
}

const statusTone: Record<LivestockStatus, ChipTone> = {
  healthy: 'secondary', sick: 'warning', quarantine: 'primary', deceased: 'neutral',
}

const statusLabel: Record<LivestockStatus, string> = {
  healthy: 'Healthy', sick: 'Sick', quarantine: 'Quarantine', deceased: 'Deceased',
}

// ─── helpers ──────────────────────────────────────────────────────────────────

function daysInTank(dateAdded: string | null): string | null {
  if (!dateAdded) return null
  const added = new Date(dateAdded)
  const now = new Date()
  const days = Math.floor((now.getTime() - added.getTime()) / (1000 * 60 * 60 * 24))
  if (days < 0) return null
  if (days === 0) return 'Added today'
  if (days === 1) return '1 day in tank'
  if (days < 30) return `${days} days in tank`
  const months = Math.floor(days / 30)
  if (months === 1) return '1 month in tank'
  if (months < 12) return `${months} months in tank`
  const years = Math.floor(months / 12)
  const remMonths = months % 12
  if (remMonths === 0) return `${years} ${years === 1 ? 'year' : 'years'} in tank`
  return `${years}y ${remMonths}m in tank`
}

// ─── lightbox ─────────────────────────────────────────────────────────────────

interface LightboxProps {
  images: string[]
  index: number
  onClose: () => void
  onChange: (i: number) => void
}

function Lightbox({ images, index, onClose, onChange }: LightboxProps) {
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'Escape') onClose()
    if (e.key === 'ArrowLeft' && index > 0) onChange(index - 1)
    if (e.key === 'ArrowRight' && index < images.length - 1) onChange(index + 1)
  }, [images.length, index, onClose, onChange])

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      style={{ background: 'rgba(7, 13, 31, 0.92)', backdropFilter: 'blur(8px)' }}
      onClick={onClose}
    >
      {/* Image */}
      <div
        className="relative max-w-5xl w-full mx-4 flex items-center justify-center"
        onClick={(e) => e.stopPropagation()}
      >
        <img
          src={images[index]}
          alt=""
          className="max-h-[85vh] max-w-full object-contain rounded-2xl shadow-abyss"
        />

        {/* Close */}
        <button
          onClick={onClose}
          className="absolute -top-3 -right-3 p-2 rounded-full glass text-on-surface-dim hover:text-on-surface transition-fluid"
        >
          <X className="h-4 w-4" />
        </button>

        {/* Prev */}
        {index > 0 && (
          <button
            onClick={() => onChange(index - 1)}
            className="absolute left-3 p-2.5 rounded-full glass text-on-surface-dim hover:text-on-surface transition-fluid"
          >
            <ChevronLeft className="h-5 w-5" />
          </button>
        )}

        {/* Next */}
        {index < images.length - 1 && (
          <button
            onClick={() => onChange(index + 1)}
            className="absolute right-3 p-2.5 rounded-full glass text-on-surface-dim hover:text-on-surface transition-fluid"
          >
            <ChevronRight className="h-5 w-5" />
          </button>
        )}

        {/* Counter */}
        {images.length > 1 && (
          <div className="absolute bottom-3 left-1/2 -translate-x-1/2 px-3 py-1 rounded-full glass text-xs text-on-surface-dim">
            {index + 1} / {images.length}
          </div>
        )}
      </div>
    </div>
  )
}

// ─── add-observation form ─────────────────────────────────────────────────────

interface AddObsFormProps {
  livestockId: number
  onClose: () => void
}

function AddObsForm({ livestockId, onClose }: AddObsFormProps) {
  const [status, setStatus] = useState<LivestockStatus | ''>('')
  const [note, setNote] = useState('')
  const [imageFile, setImageFile] = useState<File | null>(null)
  const [imagePreview, setImagePreview] = useState<string | null>(null)
  const imageInputRef = useRef<HTMLInputElement>(null)

  const createObs = useCreateLivestockObservation()
  const uploadObsImage = useUploadObservationImage()

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
    if (!status && !note.trim() && !imageFile) return

    const obs = await createObs.mutateAsync({
      livestockId,
      data: {
        status: status || null,
        note: note.trim() || null,
      },
    })

    if (imageFile && obs?.id) {
      await uploadObsImage.mutateAsync({ livestockId, obsId: obs.id, file: imageFile })
    }

    clearImage()
    setStatus('')
    setNote('')
    onClose()
  }

  const isEmpty = !status && !note.trim() && !imageFile

  return (
    <Card className="p-4">
    <form onSubmit={handleSubmit} className="space-y-3">
      <SectionHeader
        title="Log observation"
        action={<IconButton icon={X} label="Close" size={16} onClick={onClose} />}
      />

      <Select
        value={status}
        onChange={(e) => setStatus(e.target.value as LivestockStatus | '')}
      >
        <option value="">Status (optional)</option>
        <option value="healthy">Healthy</option>
        <option value="sick">Sick</option>
        <option value="quarantine">Quarantine</option>
        <option value="deceased">Deceased</option>
      </Select>

      <Textarea
        value={note}
        onChange={(e) => setNote(e.target.value)}
        placeholder="What did you observe? Coloration, behavior, feeding response…"
        rows={3}
      />

      {/* Image */}
      {imagePreview ? (
        <div className="relative">
          <img src={imagePreview} alt="preview" className="w-full max-h-48 rounded-xl object-cover" />
          <button
            type="button"
            onClick={clearImage}
            className="absolute top-2 right-2 p-1.5 rounded-lg glass text-on-surface hover:text-tertiary transition-fluid"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      ) : (
        <Button variant="ghost" icon={ImagePlus} onClick={() => imageInputRef.current?.click()}>
          Attach photo
        </Button>
      )}
      <input
        ref={imageInputRef}
        type="file"
        accept=".jpg,.jpeg,.png,.webp"
        className="hidden"
        onChange={handleImageChange}
      />

      <div className="flex gap-2 pt-1">
        <Button type="submit" variant="primary" disabled={isEmpty} loading={createObs.isPending}>
          {createObs.isPending ? 'Saving…' : 'Save'}
        </Button>
        <Button onClick={() => { onClose(); clearImage() }}>Cancel</Button>
      </div>
    </form>
    </Card>
  )
}

// Module-level map so cache-busting timestamps survive SPA navigation.
const imageBustMap: Record<string, number> = {}

// ─── main page ────────────────────────────────────────────────────────────────

export default function LivestockDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const numericId = parseInt(id ?? '', 10)

  const { data: item, isLoading, isError } = useLivestockItem(numericId)
  const { data: obsData, isLoading: obsLoading } = useLivestockObservations(numericId)
  const { data: speciesData } = useLivestockSpecies()
  const updateItem = useUpdateLivestockItem()
  const deleteItem = useDeleteLivestockItem()
  const uploadItemImage = useUploadLivestockImage()
  const editItemImage = useEditLivestockImage()
  const resetItemImage = useResetLivestockImage()
  const deleteItemImage = useDeleteLivestockImage()
  const deleteObsImage = useDeleteObservationImage()
  const editObsImage = useEditObservationImage()
  const resetObsImage = useResetObservationImage()

  const itemImageInputRef = useRef<HTMLInputElement>(null)

  const [editing, setEditing] = useState(false)
  const [addingObs, setAddingObs] = useState(false)
  const [imageError, setImageError] = useState('')
  const [editingImage, setEditingImage] = useState(false)
  const [editingObsImage, setEditingObsImage] = useState<number | null>(null)
  const [, forceRender] = useState(0)
  const bustUrl = (path: string, base?: string) => {
    const url = base ?? `/${path}`
    return imageBustMap[path] ? `${url}?v=${imageBustMap[path]}` : url
  }
  const bumpBust = (path: string) => {
    imageBustMap[path] = Date.now()
    forceRender((n) => n + 1)
  }
  const [lightboxImages, setLightboxImages] = useState<string[]>([])
  const [lightboxIndex, setLightboxIndex] = useState<number | null>(null)

  usePageTitle(item?.name ?? 'Livestock')

  const observations = [...(obsData?.observations ?? [])].sort(
    (a, b) => new Date(b.ts).getTime() - new Date(a.ts).getTime(),
  )

  // Build the gallery from item image + observation images
  const galleryImages: string[] = []
  if (item?.image_path) galleryImages.push(bustUrl(item.image_path))
  for (const obs of observations) {
    if (obs.image_path) galleryImages.push(bustUrl(obs.image_path))
  }

  function openLightbox(index: number) {
    setLightboxImages(galleryImages)
    setLightboxIndex(index)
  }

  function handleItemImageChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    setImageError('')
    uploadItemImage.mutate(
      { id: numericId, file },
      {
        onError: (err) => setImageError(err instanceof Error ? err.message : 'Upload failed'),
        onSuccess: (result) => bumpBust(result.image_path),
      },
    )
    e.target.value = ''
  }

  function handleDelete() {
    deleteItem.mutate(numericId, {
      onSuccess: () => navigate('/livestock'),
    })
  }

  async function handleUpdate(form: LivestockFormData): Promise<number> {
    await updateItem.mutateAsync({
      id: numericId,
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
    return numericId
  }

  // ── loading/error states ──────────────────────────────────────────────────

  if (isNaN(numericId)) {
    return (
      <PageBody narrow>
        <EmptyState
          message="Invalid livestock ID."
          action={<Link to="/livestock" className="text-sm text-primary hover:underline">← Back to Livestock</Link>}
        />
      </PageBody>
    )
  }

  if (isLoading) {
    return (
      <PageBody narrow>
        <LoadingState label="Loading resident" />
      </PageBody>
    )
  }

  if (isError || !item) {
    return (
      <PageBody narrow>
        <EmptyState
          message="This resident couldn't be found."
          action={<Link to="/livestock" className="text-sm text-primary hover:underline">← Back to Livestock</Link>}
        />
      </PageBody>
    )
  }

  const TypeIcon = typeIcons[item.type] ?? Box
  const tenure = daysInTank(item.date_added)
  const speciesSuggestions = speciesData?.species ?? []

  const editInitial: LivestockFormData = {
    name:       item.name,
    species:    item.species ?? '',
    type:       item.type,
    quantity:   item.quantity,
    status:     item.status,
    date_added: item.date_added ?? new Date().toISOString().slice(0, 10),
    notes:      item.notes ?? '',
  }

  return (
    <>
      {/* Lightbox */}
      {lightboxIndex !== null && (
        <Lightbox
          images={lightboxImages}
          index={lightboxIndex}
          onClose={() => setLightboxIndex(null)}
          onChange={setLightboxIndex}
        />
      )}

      {/* Image edit modal */}
      {editingImage && item.image_path && (
        <ImageEditModal
          imageSrc={originalUrl(item.image_path)}
          fallbackSrc={`/${item.image_path}`}
          onSave={async (file) => {
            setImageError('')
            await editItemImage.mutateAsync({ id: numericId, file })
            if (item.image_path) bumpBust(item.image_path)
            setEditingImage(false)
          }}
          onClose={() => setEditingImage(false)}
        />
      )}

      {editingObsImage !== null && (() => {
        const obs = observations.find((o) => o.id === editingObsImage)
        if (!obs?.image_path) return null
        return (
          <ImageEditModal
            imageSrc={bustUrl(obs.image_path, originalUrl(obs.image_path))}
            fallbackSrc={bustUrl(obs.image_path)}
            onSave={async (file) => {
              await editObsImage.mutateAsync({ livestockId: numericId, obsId: obs.id, file })
              bumpBust(obs.image_path!)
              setEditingObsImage(null)
            }}
            onClose={() => setEditingObsImage(null)}
          />
        )
      })()}

      <PageBody narrow>
        {/* ── navigation + actions row ───────────────────────────────────── */}
        <div className="flex items-center justify-between gap-4">
          <Link
            to="/livestock"
            className="flex items-center gap-1.5 text-sm text-on-surface-dim hover:text-on-surface transition-fluid"
          >
            <ArrowLeft className="h-4 w-4" />
            Livestock
          </Link>

          <div className="flex items-center gap-2">
            <Button
              size="sm"
              variant={editing ? 'tonal' : 'secondary'}
              icon={Pencil}
              onClick={() => setEditing((v) => !v)}
            >
              Edit
            </Button>

            <ConfirmButton
              onConfirm={handleDelete}
              disabled={deleteItem.isPending}
              title="Remove livestock"
              confirmLabel="Remove"
              className="p-1.5 rounded-xl text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </ConfirmButton>
          </div>
        </div>

        {/* ── edit form ──────────────────────────────────────────────────── */}
        {editing && (
          <div>
            <LivestockForm
              initial={editInitial}
              currentImagePath={item.image_path}
              speciesSuggestions={speciesSuggestions}
              onSubmit={handleUpdate}
              onClose={() => setEditing(false)}
              title={`Edit: ${item.name}`}
            />
          </div>
        )}

        {/* ── hero image ─────────────────────────────────────────────────── */}
        <div className="group relative overflow-hidden rounded-2xl" style={{ aspectRatio: '16/9' }}>
          {item.image_path ? (
            <img
              src={bustUrl(item.image_path)}
              alt={item.name}
              className="w-full h-full object-cover cursor-zoom-in transition-transform duration-500 ease-out group-hover:scale-[1.02]"
              onClick={() => openLightbox(0)}
            />
          ) : (
            <div className={cn('w-full h-full flex items-center justify-center', typePlaceholderClass[item.type])}>
              <TypeIcon className={cn('h-24 w-24', typeIconClass[item.type])} />
            </div>
          )}

          {/* Bottom gradient + identity */}
          <div
            className="absolute bottom-0 left-0 right-0 p-5"
            style={{ background: 'linear-gradient(to top, rgba(7,13,31,0.85) 0%, rgba(7,13,31,0) 100%)' }}
          >
            <h1 className="text-2xl md:text-3xl font-bold text-on-surface tracking-tight leading-tight">
              {item.name}
            </h1>
            {item.species && (
              <p className="text-sm text-on-surface-dim italic mt-0.5">{item.species}</p>
            )}
          </div>

          {/* Image controls — revealed on hover */}
          <div className="absolute top-3 left-3 flex gap-1.5 md:opacity-0 md:group-hover:opacity-100 transition-fluid">
            <button
              onClick={() => itemImageInputRef.current?.click()}
              disabled={uploadItemImage.isPending}
              title={item.image_path ? 'Replace photo' : 'Add photo'}
              className="p-2 rounded-xl glass text-on-surface-dim hover:text-on-surface transition-fluid"
            >
              <Camera className="h-4 w-4" />
            </button>
            {item.image_path && (
              <button
                onClick={() => setEditingImage(true)}
                title="Crop / rotate photo"
                className="p-2 rounded-xl glass text-on-surface-dim hover:text-on-surface transition-fluid"
              >
                <Crop className="h-4 w-4" />
              </button>
            )}
            {item.image_path && (
              <button
                onClick={() => {
                  resetItemImage.mutate(numericId, {
                    onError: (err) => setImageError(err instanceof Error ? err.message : 'Reset failed'),
                    onSuccess: () => { if (item.image_path) bumpBust(item.image_path) },
                  })
                }}
                disabled={resetItemImage.isPending}
                title="Reset to original photo"
                className="p-2 rounded-xl glass text-on-surface-dim hover:text-on-surface transition-fluid disabled:opacity-40"
              >
                <History className="h-4 w-4" />
              </button>
            )}
            {item.image_path && (
              <ConfirmButton
                onConfirm={() => deleteItemImage.mutate(numericId)}
                disabled={deleteItemImage.isPending}
                title="Remove photo"
                confirmLabel="Remove photo"
                floating
                className="p-2 rounded-xl glass text-on-surface-dim hover:text-tertiary transition-fluid"
              >
                <X className="h-4 w-4" />
              </ConfirmButton>
            )}
          </div>
          <input
            ref={itemImageInputRef}
            type="file"
            accept=".jpg,.jpeg,.png,.webp"
            className="hidden"
            onChange={handleItemImageChange}
          />
        </div>
        {imageError && <p className="text-xs text-tertiary">{imageError}</p>}

        {/* ── identity chips ─────────────────────────────────────────────── */}
        <div className="flex flex-wrap gap-2">
          <Chip className="inline-flex items-center gap-1.5">
            <TypeIcon className="h-3.5 w-3.5" />
            {typeLabel[item.type]}
          </Chip>
          <Chip tone={statusTone[item.status]} className="inline-flex items-center gap-1.5">
            <span className={cn('h-2 w-2 rounded-full flex-shrink-0', statusDotClass[item.status])} />
            {statusLabel[item.status]}
          </Chip>
          {item.quantity > 1 && <Chip>×{item.quantity}</Chip>}
          {tenure && <Chip>{tenure}</Chip>}
        </div>

        {/* ── notes ──────────────────────────────────────────────────────── */}
        {item.notes && (
          <Card className="px-4 py-3">
            <p className={cn(labelClass, 'mb-1.5')}>Notes</p>
            <p className="text-sm text-on-surface leading-relaxed">{item.notes}</p>
          </Card>
        )}

        {/* ── photo gallery filmstrip ────────────────────────────────────── */}
        {galleryImages.length > 1 && (
          <div className="space-y-3">
            <SectionHeader title="All Photos" />
            <div className="flex gap-2 overflow-x-auto pb-1">
              {galleryImages.map((src, i) => (
                <button
                  key={i}
                  onClick={() => openLightbox(i)}
                  className="group relative flex-shrink-0 w-20 h-20 rounded-xl overflow-hidden"
                >
                  <img src={src} alt="" className="w-full h-full object-cover" />
                  <div className="absolute inset-0 bg-surface-container-lowest/0 group-hover:bg-surface-container-lowest/30 transition-fluid flex items-center justify-center">
                    <ZoomIn className="h-4 w-4 text-on-surface opacity-0 group-hover:opacity-100 transition-fluid" />
                  </div>
                </button>
              ))}
            </div>
          </div>
        )}

        {/* ── observation log ────────────────────────────────────────────── */}
        <div className="space-y-4">
          <SectionHeader
            title="Observation Log"
            action={
              <div className="flex items-center gap-3">
                {observations.length > 0 && (
                  <span className="text-xs text-on-surface-faint">
                    {observations.length} {observations.length === 1 ? 'entry' : 'entries'}
                  </span>
                )}
                {!addingObs && (
                  <Button variant="tonal" size="sm" icon={Plus} onClick={() => setAddingObs(true)}>
                    Log observation
                  </Button>
                )}
              </div>
            }
          />

          {/* Add observation form */}
          {addingObs && (
            <AddObsForm livestockId={numericId} onClose={() => setAddingObs(false)} />
          )}

          {/* Observation entries */}
          {obsLoading ? (
            <LoadingState label="Loading observations" />
          ) : observations.length === 0 ? (
            <EmptyState
              message="No observations logged yet. Log feeding responses, color changes, behavior notes — build a record of this resident's life in your tank."
              action={!addingObs && (
                <Button variant="tonal" onClick={() => setAddingObs(true)}>
                  Log first observation
                </Button>
              )}
            />
          ) : (
            <div className="space-y-3">
              {observations.map((obs) => (
                <Card key={obs.id} className="p-4 space-y-3">
                  {/* Header: time + status */}
                  <div className="flex items-center justify-between gap-3">
                    <div className="flex items-center gap-2.5 flex-wrap">
                      <span className="text-xs text-on-surface-faint">
                        <TimeAgo ts={obs.ts} />
                      </span>
                      {obs.status && (
                        <Chip
                          tone={statusTone[obs.status as LivestockStatus]}
                          className="inline-flex items-center gap-1.5"
                        >
                          <span className={cn('h-1.5 w-1.5 rounded-full', statusDotClass[obs.status as LivestockStatus])} />
                          {statusLabel[obs.status as LivestockStatus]}
                        </Chip>
                      )}
                    </div>
                    {obs.image_path && (
                      <div className="flex items-center gap-0.5 flex-shrink-0">
                        <IconButton
                          icon={Crop}
                          label="Edit photo"
                          tone="primary"
                          onClick={() => setEditingObsImage(obs.id)}
                        />
                        <IconButton
                          icon={History}
                          label="Reset to original"
                          onClick={() => resetObsImage.mutate({ livestockId: numericId, obsId: obs.id }, { onSuccess: () => { if (obs.image_path) bumpBust(obs.image_path) } })}
                          disabled={resetObsImage.isPending}
                        />
                        <ConfirmButton
                          onConfirm={() => deleteObsImage.mutate({ livestockId: numericId, obsId: obs.id })}
                          disabled={deleteObsImage.isPending}
                          className="p-1 rounded-lg text-on-surface-faint hover:text-tertiary transition-fluid"
                          title="Remove photo"
                          confirmLabel="Remove photo"
                        >
                          <X className="h-3.5 w-3.5" />
                        </ConfirmButton>
                      </div>
                    )}
                  </div>

                  {/* Note text */}
                  {obs.note && (
                    <p className="text-sm text-on-surface-dim leading-relaxed">{obs.note}</p>
                  )}

                  {/* Observation image */}
                  {obs.image_path && (
                    <button
                      onClick={() => {
                        const idx = galleryImages.indexOf(bustUrl(obs.image_path!))
                        if (idx >= 0) openLightbox(idx)
                      }}
                      className="group block w-full overflow-hidden rounded-xl"
                    >
                      <img
                        src={bustUrl(obs.image_path, thumbUrl(obs.image_path))}
                        alt="observation"
                        className="w-full max-h-56 object-cover transition-transform duration-500 ease-out group-hover:scale-105"
                      />
                    </button>
                  )}
                </Card>
              ))}
            </div>
          )}
        </div>
      </PageBody>
    </>
  )
}
