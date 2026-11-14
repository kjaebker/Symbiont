import { useState } from 'react'
import { ConfirmButton } from '@/components/ConfirmButton'
import { Plus, Pencil, Trash2, ChevronDown, ChevronUp } from 'lucide-react'
import { DOSING_FREQUENCY_LABELS as FREQUENCY_LABELS } from '@/lib/dosing'
import { Button, Card, EmptyState, Field, IconButton, Input, SectionHeader, Select } from '@/components/ui'
import {
  useDosingProducts,
  useDosingSchedules,
  useCreateDosingProduct,
  useUpdateDosingProduct,
  useDeleteDosingProduct,
  useCreateDosingSchedule,
  useDeleteDosingSchedule,
} from '@/hooks/useDosing'
import type { DosingProduct, DosingProductType, DosingFrequency } from '@/api/client'

const PRODUCT_TYPE_LABELS: Record<DosingProductType, string> = {
  two_part_a: '2-Part A',
  two_part_b: '2-Part B',
  calcium: 'Calcium',
  alkalinity: 'Alkalinity',
  magnesium: 'Magnesium',
  trace: 'Trace Elements',
  amino: 'Amino / Carbon',
  bacteria: 'Bacteria',
  carbon_source: 'Carbon Source',
  filter_media: 'Filter Media',
  other: 'Other',
}

// ─── Product Form ─────────────────────────────────────────────────────────────

interface ProductFormProps {
  initial?: DosingProduct
  onSubmit: (data: {
    brand: string; name: string; type: DosingProductType; unit: string; notes?: string
  }) => void
  onCancel: () => void
  loading: boolean
}

function ProductForm({ initial, onSubmit, onCancel, loading }: ProductFormProps) {
  const [brand, setBrand] = useState(initial?.brand ?? '')
  const [name, setName] = useState(initial?.name ?? '')
  const [type, setType] = useState<DosingProductType>(initial?.type ?? 'other')
  const [unit, setUnit] = useState(initial?.unit ?? 'mL')
  const [notes, setNotes] = useState(initial?.notes ?? '')

  return (
    <Card className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <Field label="Brand">
          <Input
            value={brand}
            onChange={e => setBrand(e.target.value)}
            placeholder="e.g. BRS"
          />
        </Field>
        <Field label="Product name">
          <Input
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="e.g. 2-Part Alkalinity"
          />
        </Field>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Type">
          <Select
            value={type}
            onChange={e => setType(e.target.value as DosingProductType)}
          >
            {(Object.entries(PRODUCT_TYPE_LABELS) as [DosingProductType, string][]).map(([v, l]) => (
              <option key={v} value={v}>{l}</option>
            ))}
          </Select>
        </Field>
        <Field label="Unit">
          <Select
            value={unit}
            onChange={e => setUnit(e.target.value)}
          >
            <option value="mL">mL</option>
            <option value="g">g</option>
            <option value="drops">drops</option>
            <option value="tsp">tsp</option>
            <option value="tbsp">tbsp</option>
            <option value="fl oz">fl oz</option>
          </Select>
        </Field>
      </div>
      <Field label="Notes (optional)">
        <Input
          value={notes}
          onChange={e => setNotes(e.target.value)}
          placeholder="Optional notes"
        />
      </Field>
      <div className="flex gap-2 justify-end">
        <Button variant="secondary" onClick={onCancel}>Cancel</Button>
        <Button
          variant="primary"
          disabled={loading || !brand || !name}
          onClick={() => onSubmit({ brand, name, type, unit, notes: notes || undefined })}
        >
          {loading ? 'Saving…' : initial ? 'Update' : 'Add Product'}
        </Button>
      </div>
    </Card>
  )
}

// ─── Schedule Form ────────────────────────────────────────────────────────────

function ScheduleForm({
  products,
  initial,
  onSubmit,
  onCancel,
  loading,
}: {
  products: DosingProduct[]
  initial?: { productId: number; amount: number; frequency: DosingFrequency; intervalDays?: number; dayOfWeek?: number; notes?: string }
  onSubmit: (data: {
    product_id: number; amount: number; frequency: DosingFrequency;
    interval_days?: number; day_of_week?: number; notes?: string; enabled: boolean
  }) => void
  onCancel: () => void
  loading: boolean
}) {
  const [productId, setProductId] = useState(initial?.productId ?? (products[0]?.id ?? 0))
  const [amount, setAmount] = useState(String(initial?.amount ?? ''))
  const [frequency, setFrequency] = useState<DosingFrequency>(initial?.frequency ?? 'daily')
  const [intervalDays, setIntervalDays] = useState(String(initial?.intervalDays ?? ''))
  const [dayOfWeek, setDayOfWeek] = useState(initial?.dayOfWeek ?? 0)
  const [notes, setNotes] = useState(initial?.notes ?? '')

  const product = products.find(p => p.id === productId)

  return (
    <Card className="space-y-4">
      <Field label="Product">
        <Select
          value={productId}
          onChange={e => setProductId(Number(e.target.value))}
        >
          {products.map(p => (
            <option key={p.id} value={p.id}>{p.brand} {p.name}</option>
          ))}
        </Select>
      </Field>
      <div className="grid grid-cols-2 gap-3">
        <Field label={`Amount (${product?.unit ?? 'mL'})`}>
          <Input
            type="number"
            min="0"
            step="0.5"
            value={amount}
            onChange={e => setAmount(e.target.value)}
            placeholder="5"
          />
        </Field>
        <Field label="Frequency">
          <Select
            value={frequency}
            onChange={e => setFrequency(e.target.value as DosingFrequency)}
          >
            {(Object.entries(FREQUENCY_LABELS) as [DosingFrequency, string][]).map(([v, l]) => (
              <option key={v} value={v}>{l}</option>
            ))}
          </Select>
        </Field>
      </div>
      {frequency === 'every_n_days' && (
        <Field label="Every how many days">
          <Input
            type="number"
            min="1"
            value={intervalDays}
            onChange={e => setIntervalDays(e.target.value)}
            placeholder="3"
          />
        </Field>
      )}
      {frequency === 'weekly' && (
        <Field label="Day of week">
          <Select
            value={dayOfWeek}
            onChange={e => setDayOfWeek(Number(e.target.value))}
          >
            {['Sunday','Monday','Tuesday','Wednesday','Thursday','Friday','Saturday'].map((d, i) => (
              <option key={i} value={i}>{d}</option>
            ))}
          </Select>
        </Field>
      )}
      <Field label="Notes (optional)">
        <Input
          value={notes}
          onChange={e => setNotes(e.target.value)}
          placeholder="Optional notes"
        />
      </Field>
      <div className="flex gap-2 justify-end">
        <Button variant="secondary" onClick={onCancel}>Cancel</Button>
        <Button
          variant="primary"
          disabled={loading || !productId || !amount}
          onClick={() => onSubmit({
            product_id: productId,
            amount: Number(amount),
            frequency,
            interval_days: frequency === 'every_n_days' && intervalDays ? Number(intervalDays) : undefined,
            day_of_week: frequency === 'weekly' ? dayOfWeek : undefined,
            notes: notes || undefined,
            enabled: true,
          })}
        >
          {loading ? 'Saving…' : 'Add Schedule'}
        </Button>
      </div>
    </Card>
  )
}

// ─── Dosing Tab ───────────────────────────────────────────────────────────────
// Setup only: which products exist and how often each is dosed. Logging a dose
// happens on the Tasks page.

export default function DosingTab() {
  const { data: productsData } = useDosingProducts()
  const { data: schedulesData } = useDosingSchedules()
  const products = productsData?.products ?? []
  const schedules = schedulesData?.schedules ?? []

  const createProduct = useCreateDosingProduct()
  const updateProduct = useUpdateDosingProduct()
  const deleteProduct = useDeleteDosingProduct()
  const createSchedule = useCreateDosingSchedule()
  const deleteSchedule = useDeleteDosingSchedule()

  const [showProductForm, setShowProductForm] = useState(false)
  const [editingProduct, setEditingProduct] = useState<DosingProduct | null>(null)
  const [showScheduleForm, setShowScheduleForm] = useState(false)
  // Collapsed by default, but open when there is nothing in it yet.
  const [catalogToggled, setCatalogOpen] = useState<boolean | null>(null)

  const noProducts = productsData !== undefined && products.length === 0

  const catalogOpen = catalogToggled ?? noProducts

  return (
    <div className="space-y-6">
      <Card className="space-y-4">
        <SectionHeader
          title="Dosing Schedules"
          action={
            <Button
              variant="tonal"
              size="sm"
              icon={Plus}
              onClick={() => setShowScheduleForm(v => !v)}
              disabled={noProducts}
              title={noProducts ? 'Add products to your catalog first' : undefined}
            >
              Add Schedule
            </Button>
          }
        />

        {showScheduleForm && (
          <ScheduleForm
            products={products}
            onSubmit={async data => {
              await createSchedule.mutateAsync(data)
              setShowScheduleForm(false)
            }}
            onCancel={() => setShowScheduleForm(false)}
            loading={createSchedule.isPending}
          />
        )}

        {schedules.length === 0 ? (
          <EmptyState
            message={noProducts
              ? 'No products in your catalog yet. Add products below, then come back to create a schedule.'
              : 'No dosing schedules. Add one above.'}
          />
        ) : (
          <div className="space-y-2">
            {schedules.map(s => (
              <div key={s.id} className="flex items-center gap-3 p-3 rounded-xl bg-surface-container-high/50 hover:bg-surface-container-high transition-fluid">
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-on-surface">
                    {s.product_brand} {s.product_name}
                  </div>
                  <div className="text-xs text-on-surface-dim mt-0.5">
                    {s.amount} {s.product_unit} · {FREQUENCY_LABELS[s.frequency]}
                    {s.interval_days && s.frequency === 'every_n_days' && ` (${s.interval_days}d)`}
                  </div>
                </div>
                <ConfirmButton
                  onConfirm={() => deleteSchedule.mutate(s.id)}
                  className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid shrink-0"
                  title="Remove schedule"
                  confirmLabel="Remove"
                >
                  <Trash2 size={15} />
                </ConfirmButton>
              </div>
            ))}
          </div>
        )}
      </Card>

      {/* Product catalog */}
      <Card flush>
        <button
          onClick={() => setCatalogOpen(!catalogOpen)}
          className="w-full flex items-center justify-between p-5 text-left hover:bg-surface-container-high/30 transition-fluid"
        >
          <span className="text-sm uppercase tracking-widest text-on-surface font-semibold">Product Catalog</span>
          {catalogOpen ? <ChevronUp size={16} className="text-on-surface-dim" /> : <ChevronDown size={16} className="text-on-surface-dim" />}
        </button>

        {catalogOpen && (
          <div className="px-5 pb-5 space-y-4">
            <div className="flex justify-end">
              <Button variant="tonal" size="sm" icon={Plus} onClick={() => setShowProductForm(v => !v)}>
                Add Product
              </Button>
            </div>

            {showProductForm && !editingProduct && (
              <ProductForm
                onSubmit={async data => {
                  await createProduct.mutateAsync(data)
                  setShowProductForm(false)
                }}
                onCancel={() => setShowProductForm(false)}
                loading={createProduct.isPending}
              />
            )}

            {editingProduct && (
              <ProductForm
                initial={editingProduct}
                onSubmit={async data => {
                  await updateProduct.mutateAsync({ id: editingProduct.id, data })
                  setEditingProduct(null)
                }}
                onCancel={() => setEditingProduct(null)}
                loading={updateProduct.isPending}
              />
            )}

            {products.length === 0 ? (
              <EmptyState message="No products yet." />
            ) : (
              <div className="space-y-2">
                {products.map(p => (
                  <div key={p.id} className="flex items-center justify-between p-3 rounded-xl bg-surface-container-high/50">
                    <div>
                      <span className="text-sm font-medium text-on-surface">{p.brand} {p.name}</span>
                      <span className="ml-2 text-xs text-on-surface-dim">{PRODUCT_TYPE_LABELS[p.type]} · {p.unit}</span>
                    </div>
                    <div className="flex gap-1">
                      <IconButton icon={Pencil} label="Edit product" tone="primary" onClick={() => setEditingProduct(p)} />
                      <ConfirmButton
                        onConfirm={() => deleteProduct.mutate(p.id)}
                        className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid"
                        title="Delete product"
                      >
                        <Trash2 size={14} />
                      </ConfirmButton>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </Card>
    </div>
  )
}

