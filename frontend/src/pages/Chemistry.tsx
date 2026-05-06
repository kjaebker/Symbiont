import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import { FlaskConical, Droplets, Plus, Pencil, Trash2, CheckCircle2, ChevronDown, ChevronUp } from 'lucide-react'
import { usePageTitle } from '@/hooks/usePageTitle'
import { cn, relativeTime } from '@/lib/utils'
import Measurements from '@/pages/Measurements'
import { PageHeader } from '@/components/PageHeader'
import { FilterDropdown } from '@/components/FilterDropdown'
import { useMeasurementParameters } from '@/hooks/useMeasurements'
import {
  useDosingProducts,
  useDosingSchedules,
  useDosingLogs,
  useCreateDosingProduct,
  useUpdateDosingProduct,
  useDeleteDosingProduct,
  useCreateDosingSchedule,
  useDeleteDosingSchedule,
  useLogDose,
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

const FREQUENCY_LABELS: Record<DosingFrequency, string> = {
  daily: 'Daily',
  twice_daily: 'Twice daily',
  every_n_days: 'Every N days',
  weekly: 'Weekly',
  as_needed: 'As needed',
}

function formatNextDue(iso: string | null): string {
  if (!iso) return 'Not scheduled'
  const d = new Date(iso)
  const now = new Date()
  const diff = d.getTime() - now.getTime()
  if (diff < 0) return 'Overdue'
  if (diff < 3600_000) return 'Due soon'
  if (diff < 86400_000) return 'Due today'
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
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
    <div className="bg-surface-container rounded-2xl p-5 space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <label className="text-xs uppercase tracking-wider text-on-surface-dim">Brand</label>
          <input
            className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
            value={brand}
            onChange={e => setBrand(e.target.value)}
            placeholder="e.g. BRS"
          />
        </div>
        <div className="space-y-1">
          <label className="text-xs uppercase tracking-wider text-on-surface-dim">Product name</label>
          <input
            className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
            value={name}
            onChange={e => setName(e.target.value)}
            placeholder="e.g. 2-Part Alkalinity"
          />
        </div>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <label className="text-xs uppercase tracking-wider text-on-surface-dim">Type</label>
          <select
            className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
            value={type}
            onChange={e => setType(e.target.value as DosingProductType)}
          >
            {(Object.entries(PRODUCT_TYPE_LABELS) as [DosingProductType, string][]).map(([v, l]) => (
              <option key={v} value={v}>{l}</option>
            ))}
          </select>
        </div>
        <div className="space-y-1">
          <label className="text-xs uppercase tracking-wider text-on-surface-dim">Unit</label>
          <select
            className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
            value={unit}
            onChange={e => setUnit(e.target.value)}
          >
            <option value="mL">mL</option>
            <option value="g">g</option>
            <option value="drops">drops</option>
            <option value="tsp">tsp</option>
            <option value="tbsp">tbsp</option>
            <option value="fl oz">fl oz</option>
          </select>
        </div>
      </div>
      <div className="space-y-1">
        <label className="text-xs uppercase tracking-wider text-on-surface-dim">Notes (optional)</label>
        <input
          className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
          value={notes}
          onChange={e => setNotes(e.target.value)}
          placeholder="Optional notes"
        />
      </div>
      <div className="flex gap-2 justify-end">
        <button
          onClick={onCancel}
          className="px-4 py-2 rounded-xl text-sm text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high transition-fluid"
        >
          Cancel
        </button>
        <button
          disabled={loading || !brand || !name}
          onClick={() => onSubmit({ brand, name, type, unit, notes: notes || undefined })}
          className="px-4 py-2 rounded-xl text-sm font-medium bg-primary/15 text-primary hover:bg-primary/25 disabled:opacity-50 transition-fluid"
        >
          {loading ? 'Saving…' : initial ? 'Update' : 'Add Product'}
        </button>
      </div>
    </div>
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
    <div className="bg-surface-container rounded-2xl p-5 space-y-4">
      <div className="space-y-1">
        <label className="text-xs uppercase tracking-wider text-on-surface-dim">Product</label>
        <select
          className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
          value={productId}
          onChange={e => setProductId(Number(e.target.value))}
        >
          {products.map(p => (
            <option key={p.id} value={p.id}>{p.brand} {p.name}</option>
          ))}
        </select>
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div className="space-y-1">
          <label className="text-xs uppercase tracking-wider text-on-surface-dim">
            Amount ({product?.unit ?? 'mL'})
          </label>
          <input
            type="number"
            min="0"
            step="0.5"
            className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
            value={amount}
            onChange={e => setAmount(e.target.value)}
            placeholder="5"
          />
        </div>
        <div className="space-y-1">
          <label className="text-xs uppercase tracking-wider text-on-surface-dim">Frequency</label>
          <select
            className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
            value={frequency}
            onChange={e => setFrequency(e.target.value as DosingFrequency)}
          >
            {(Object.entries(FREQUENCY_LABELS) as [DosingFrequency, string][]).map(([v, l]) => (
              <option key={v} value={v}>{l}</option>
            ))}
          </select>
        </div>
      </div>
      {frequency === 'every_n_days' && (
        <div className="space-y-1">
          <label className="text-xs uppercase tracking-wider text-on-surface-dim">Every how many days</label>
          <input
            type="number"
            min="1"
            className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
            value={intervalDays}
            onChange={e => setIntervalDays(e.target.value)}
            placeholder="3"
          />
        </div>
      )}
      {frequency === 'weekly' && (
        <div className="space-y-1">
          <label className="text-xs uppercase tracking-wider text-on-surface-dim">Day of week</label>
          <select
            className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
            value={dayOfWeek}
            onChange={e => setDayOfWeek(Number(e.target.value))}
          >
            {['Sunday','Monday','Tuesday','Wednesday','Thursday','Friday','Saturday'].map((d, i) => (
              <option key={i} value={i}>{d}</option>
            ))}
          </select>
        </div>
      )}
      <div className="space-y-1">
        <label className="text-xs uppercase tracking-wider text-on-surface-dim">Notes (optional)</label>
        <input
          className="w-full bg-surface-container-high rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/50"
          value={notes}
          onChange={e => setNotes(e.target.value)}
          placeholder="Optional notes"
        />
      </div>
      <div className="flex gap-2 justify-end">
        <button
          onClick={onCancel}
          className="px-4 py-2 rounded-xl text-sm text-on-surface-dim hover:text-on-surface hover:bg-surface-container-high transition-fluid"
        >
          Cancel
        </button>
        <button
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
          className="px-4 py-2 rounded-xl text-sm font-medium bg-primary/15 text-primary hover:bg-primary/25 disabled:opacity-50 transition-fluid"
        >
          {loading ? 'Saving…' : 'Add Schedule'}
        </button>
      </div>
    </div>
  )
}

// ─── Dosing Tab ───────────────────────────────────────────────────────────────

function DosingTab() {
  const { data: productsData } = useDosingProducts()
  const { data: schedulesData } = useDosingSchedules()
  const { data: logsData } = useDosingLogs({ limit: 30 })
  const products = productsData?.products ?? []
  const schedules = schedulesData?.schedules ?? []
  const logs = logsData?.logs ?? []

  const createProduct = useCreateDosingProduct()
  const updateProduct = useUpdateDosingProduct()
  const deleteProduct = useDeleteDosingProduct()
  const createSchedule = useCreateDosingSchedule()
  const deleteSchedule = useDeleteDosingSchedule()
  const logDose = useLogDose()

  const [showProductForm, setShowProductForm] = useState(false)
  const [editingProduct, setEditingProduct] = useState<DosingProduct | null>(null)
  const [showScheduleForm, setShowScheduleForm] = useState(false)
  const [catalogOpen, setCatalogOpen] = useState(false)
  const [loggingId, setLoggingId] = useState<number | null>(null)

  const noProducts = productsData !== undefined && products.length === 0

  useEffect(() => {
    if (noProducts) setCatalogOpen(true)
  }, [noProducts])

  const overdue = schedules.filter(s => {
    if (!s.enabled || s.frequency === 'as_needed' || !s.next_due_at) return false
    return new Date(s.next_due_at) <= new Date()
  })
  const upcoming = schedules.filter(s => {
    if (!s.enabled || s.frequency === 'as_needed') return false
    if (!s.next_due_at) return true
    const diff = new Date(s.next_due_at).getTime() - Date.now()
    return diff > 0 && diff <= 86400_000 * 2
  })

  async function handleLogDose(scheduleId: number) {
    setLoggingId(scheduleId)
    try {
      await logDose.mutateAsync({ scheduleId, data: {} })
    } finally {
      setLoggingId(null)
    }
  }

  return (
    <div className="space-y-6">
      {/* Due / Overdue */}
      {(overdue.length > 0 || upcoming.length > 0) && (
        <div className="bg-surface-container rounded-2xl p-5">
          <h2 className="text-sm uppercase tracking-wider text-on-surface-dim font-semibold mb-4">Due Now</h2>
          <div className="space-y-2">
            {[...overdue, ...upcoming].map(s => {
              const isOverdue = s.next_due_at ? new Date(s.next_due_at) <= new Date() : false
              return (
                <div key={s.id} className={cn(
                  'flex items-center justify-between p-3 rounded-xl',
                  isOverdue ? 'bg-tertiary/10' : 'bg-surface-container-high',
                )}>
                  <div>
                    <span className={cn('text-sm font-medium', isOverdue ? 'text-tertiary' : 'text-on-surface')}>
                      {s.product_brand} {s.product_name}
                    </span>
                    <span className="text-xs text-on-surface-dim ml-2">
                      {s.amount} {s.product_unit}
                    </span>
                    {isOverdue && (
                      <span className="ml-2 text-xs uppercase tracking-wider text-tertiary font-semibold">Overdue</span>
                    )}
                  </div>
                  <button
                    onClick={() => handleLogDose(s.id)}
                    disabled={loggingId === s.id}
                    className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium bg-primary/15 text-primary hover:bg-primary/25 disabled:opacity-50 transition-fluid"
                  >
                    <CheckCircle2 size={14} />
                    {loggingId === s.id ? 'Logging…' : 'Log Dose'}
                  </button>
                </div>
              )
            })}
          </div>
        </div>
      )}

      {/* All schedules */}
      <div className="bg-surface-container rounded-2xl p-5">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm uppercase tracking-wider text-on-surface-dim font-semibold">Dosing Schedules</h2>
          <button
            onClick={() => setShowScheduleForm(v => !v)}
            disabled={noProducts}
            title={noProducts ? 'Add products to your catalog first' : undefined}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium bg-primary/10 text-primary hover:bg-primary/20 disabled:opacity-40 disabled:cursor-not-allowed transition-fluid"
          >
            <Plus size={14} />
            Add Schedule
          </button>
        </div>

        {showScheduleForm && (
          <div className="mb-4">
            <ScheduleForm
              products={products}
              onSubmit={async data => {
                await createSchedule.mutateAsync(data)
                setShowScheduleForm(false)
              }}
              onCancel={() => setShowScheduleForm(false)}
              loading={createSchedule.isPending}
            />
          </div>
        )}

        {schedules.length === 0 ? (
          <div className="text-center py-6 space-y-1">
            {noProducts ? (
              <>
                <p className="text-sm text-on-surface-dim">No products in your catalog yet.</p>
                <p className="text-xs text-on-surface-faint">Add products below, then come back to create a schedule.</p>
              </>
            ) : (
              <p className="text-sm text-on-surface-dim">No dosing schedules. Add one above.</p>
            )}
          </div>
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
                <div className="text-right shrink-0">
                  <div className={cn('text-xs font-medium', s.next_due_at && new Date(s.next_due_at) <= new Date() ? 'text-tertiary' : 'text-on-surface-dim')}>
                    {formatNextDue(s.next_due_at)}
                  </div>
                  {s.last_completed_at && (
                    <div className="text-xs text-on-surface-faint">
                      Last: {relativeTime(s.last_completed_at)}
                    </div>
                  )}
                </div>
                <div className="flex gap-1 shrink-0">
                  <button
                    onClick={() => handleLogDose(s.id)}
                    disabled={loggingId === s.id}
                    className="p-1.5 rounded-lg text-primary hover:bg-primary/10 disabled:opacity-50 transition-fluid"
                    title="Log dose now"
                  >
                    <CheckCircle2 size={15} />
                  </button>
                  <button
                    onClick={() => deleteSchedule.mutate(s.id)}
                    className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid"
                    title="Remove schedule"
                  >
                    <Trash2 size={15} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Recent dose history */}
      {logs.length > 0 && (
        <div className="bg-surface-container rounded-2xl p-5">
          <h2 className="text-sm uppercase tracking-wider text-on-surface-dim font-semibold mb-4">Recent Doses</h2>
          <div className="space-y-1.5">
            {logs.slice(0, 20).map(l => (
              <div key={l.id} className="flex items-center justify-between py-2 px-3 rounded-xl hover:bg-surface-container-high/50 transition-fluid">
                <div>
                  <span className="text-sm text-on-surface">{l.product_brand} {l.product_name}</span>
                  <span className="text-xs text-on-surface-dim ml-2">{l.amount} {l.product_unit}</span>
                </div>
                <span className="text-xs text-on-surface-dim">{relativeTime(l.dosed_at)}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Product catalog */}
      <div className="bg-surface-container rounded-2xl overflow-hidden">
        <button
          onClick={() => setCatalogOpen(v => !v)}
          className="w-full flex items-center justify-between p-5 text-left hover:bg-surface-container-high/30 transition-fluid"
        >
          <span className="text-sm uppercase tracking-wider text-on-surface-dim font-semibold">Product Catalog</span>
          {catalogOpen ? <ChevronUp size={16} className="text-on-surface-dim" /> : <ChevronDown size={16} className="text-on-surface-dim" />}
        </button>

        {catalogOpen && (
          <div className="px-5 pb-5 space-y-4 border-t border-surface-container-high/30">
            <div className="flex justify-end pt-4">
              <button
                onClick={() => setShowProductForm(v => !v)}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-xl text-xs font-medium bg-primary/10 text-primary hover:bg-primary/20 transition-fluid"
              >
                <Plus size={14} />
                Add Product
              </button>
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
              <p className="text-sm text-on-surface-dim text-center py-4">No products yet.</p>
            ) : (
              <div className="space-y-2">
                {products.map(p => (
                  <div key={p.id} className="flex items-center justify-between p-3 rounded-xl bg-surface-container-high/50">
                    <div>
                      <span className="text-sm font-medium text-on-surface">{p.brand} {p.name}</span>
                      <span className="ml-2 text-xs text-on-surface-dim">{PRODUCT_TYPE_LABELS[p.type]} · {p.unit}</span>
                    </div>
                    <div className="flex gap-1">
                      <button
                        onClick={() => setEditingProduct(p)}
                        className="p-1.5 rounded-lg text-on-surface-faint hover:text-primary hover:bg-primary/10 transition-fluid"
                      >
                        <Pencil size={14} />
                      </button>
                      <button
                        onClick={() => deleteProduct.mutate(p.id)}
                        className="p-1.5 rounded-lg text-on-surface-faint hover:text-tertiary hover:bg-tertiary/10 transition-fluid"
                      >
                        <Trash2 size={14} />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

// ─── Chemistry Page ───────────────────────────────────────────────────────────

export default function Chemistry() {
  const [searchParams, setSearchParams] = useSearchParams()
  const tab = searchParams.get('tab') ?? 'water-tests'
  usePageTitle('Chemistry')

  const { data: paramsData } = useMeasurementParameters()
  const parameters = paramsData?.parameters ?? []

  const [selectedParam, setSelectedParam] = useState<string | null>(null)
  const [showMeasurementForm, setShowMeasurementForm] = useState(false)

  function setTab(t: string) {
    setSearchParams({ tab: t }, { replace: true })
  }

  const measurementFilterBar = (
    <FilterDropdown
      label="All parameters"
      value={selectedParam ?? ''}
      options={[
        { value: '', label: 'All parameters' },
        ...parameters.map((p) => ({ value: p.name, label: p.name })),
      ]}
      onChange={(v) => setSelectedParam(v === '' ? null : v)}
    />
  )

  return (
    <div>
      <PageHeader
        subtitle="Chemical Balance"
        title="Water Chemistry"
        tabs={[
          { key: 'water-tests', label: 'Measurements', icon: FlaskConical },
          { key: 'dosing', label: 'Dosing', icon: Droplets },
        ]}
        activeTab={tab}
        onTabChange={setTab}
        action={tab === 'water-tests'
          ? { label: 'Add Measurement', icon: Plus, onClick: () => setShowMeasurementForm((v) => !v), active: showMeasurementForm }
          : null
        }
        filterBar={tab === 'water-tests' ? measurementFilterBar : undefined}
        filterActive={!!selectedParam}
        maxWidth="max-w-6xl"
      />

      {tab === 'water-tests' ? (
        <Measurements
          hideHeader
          selectedParam={selectedParam}
          onSelectedParamChange={setSelectedParam}
          showForm={showMeasurementForm}
          onShowFormChange={setShowMeasurementForm}
        />
      ) : (
        <div className="p-6 md:p-8 max-w-6xl mx-auto">
          <DosingTab />
        </div>
      )}
    </div>
  )
}
