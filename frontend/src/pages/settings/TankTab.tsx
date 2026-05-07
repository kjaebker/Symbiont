import { useState,useEffect } from 'react'
import { cn } from '@/lib/utils'
import { Check,Waves } from 'lucide-react'
import { useTankProfile,useUpsertTankProfile } from '@/hooks/useTankProfile'
import type { TankSection,TankType,TankProfileInput } from '@/api/client'

// =============================================================================
// Tank Tab
// =============================================================================

const TANK_TYPE_LABELS: Record<TankType, string> = {
  reef: 'Reef',
  fowlr: 'Fish Only with Live Rock (FOWLR)',
  mixed: 'Mixed Reef',
  nano: 'Nano Reef',
  freshwater: 'Freshwater',
  other: 'Other',
}

function TankProfileForm({
  section,
  title,
}: {
  section: TankSection
  title: string
}) {
  const { data, isLoading } = useTankProfile()
  const upsert = useUpsertTankProfile()
  const profile = data?.[section]

  const [expanded, setExpanded] = useState(false)
  const [form, setForm] = useState<Partial<TankProfileInput>>({})
  const [saved, setSaved] = useState(false)

  // Populate form when profile loads
  useEffect(() => {
    if (profile) {
      setForm({
        display_name: profile.display_name ?? undefined,
        volume_gallons: profile.volume_gallons ?? undefined,
        length_in: profile.length_in ?? undefined,
        width_in: profile.width_in ?? undefined,
        height_in: profile.height_in ?? undefined,
        tank_type: profile.tank_type ?? undefined,
        manufacturer: profile.manufacturer ?? undefined,
        model: profile.model ?? undefined,
        setup_date: profile.setup_date ?? undefined,
        notes: profile.notes ?? undefined,
      })
    }
  }, [profile])

  function field(key: keyof TankProfileInput) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
      const val = e.target.value
      setForm(prev => ({ ...prev, [key]: val === '' ? undefined : val }))
    }
  }

  function numericField(key: keyof TankProfileInput) {
    return (e: React.ChangeEvent<HTMLInputElement>) => {
      const val = parseFloat(e.target.value)
      setForm(prev => ({ ...prev, [key]: isNaN(val) ? undefined : val }))
    }
  }

  async function handleSave() {
    await upsert.mutateAsync({ section, data: form })
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  if (isLoading) return <div className="py-3 text-xs text-on-surface-faint animate-pulse px-4">Loading...</div>

  const hasData = profile && (profile.display_name || profile.volume_gallons || profile.tank_type)

  return (
    <div className="rounded-2xl bg-surface-container-high/40 overflow-hidden">
      <button
        onClick={() => setExpanded(v => !v)}
        className="w-full flex items-center justify-between px-4 py-3 hover:bg-surface-container-high/60 transition-fluid"
      >
        <div className="flex items-center gap-3">
          <Waves size={16} className="text-primary/60" />
          <div className="text-left">
            <p className="text-sm font-semibold text-on-surface">{title}</p>
            {hasData && (
              <p className="text-xs text-on-surface-dim mt-0.5">
                {[profile.volume_gallons && `${profile.volume_gallons} gal`, profile.tank_type && TANK_TYPE_LABELS[profile.tank_type as TankType]].filter(Boolean).join(' · ')}
              </p>
            )}
          </div>
        </div>
        <span className={cn('text-on-surface-faint transition-fluid', expanded && 'rotate-180')}>▾</span>
      </button>

      {expanded && (
        <div className="px-4 pb-4 space-y-3 border-t border-surface-container-highest/30">
          <div className="pt-3 grid grid-cols-1 sm:grid-cols-2 gap-3">
            {/* Display name */}
            <div className="sm:col-span-2">
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Display Name</label>
              <input
                type="text"
                value={form.display_name ?? ''}
                onChange={field('display_name')}
                placeholder={title}
                className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid"
              />
            </div>

            {/* Tank type */}
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Type</label>
              <select
                value={form.tank_type ?? ''}
                onChange={field('tank_type')}
                className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/40 transition-fluid"
              >
                <option value="">Select type...</option>
                {(Object.keys(TANK_TYPE_LABELS) as TankType[]).map(t => (
                  <option key={t} value={t}>{TANK_TYPE_LABELS[t]}</option>
                ))}
              </select>
            </div>

            {/* Volume */}
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Volume (gallons)</label>
              <input
                type="number"
                min="0"
                step="0.1"
                value={form.volume_gallons ?? ''}
                onChange={numericField('volume_gallons')}
                placeholder="e.g. 75"
                className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid"
              />
            </div>

            {/* Dimensions */}
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Length (in)</label>
              <input type="number" min="0" step="0.5" value={form.length_in ?? ''} onChange={numericField('length_in')} placeholder="48" className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid" />
            </div>
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Width (in)</label>
              <input type="number" min="0" step="0.5" value={form.width_in ?? ''} onChange={numericField('width_in')} placeholder="18" className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid" />
            </div>
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Height (in)</label>
              <input type="number" min="0" step="0.5" value={form.height_in ?? ''} onChange={numericField('height_in')} placeholder="20" className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid" />
            </div>

            {/* Manufacturer & model */}
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Manufacturer</label>
              <input type="text" value={form.manufacturer ?? ''} onChange={field('manufacturer')} placeholder="e.g. Red Sea" className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid" />
            </div>
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Model</label>
              <input type="text" value={form.model ?? ''} onChange={field('model')} placeholder="e.g. Reefer 350" className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid" />
            </div>

            {/* Setup date */}
            <div>
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Setup Date</label>
              <input type="date" value={form.setup_date ?? ''} onChange={field('setup_date')} className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface outline-none focus:ring-1 focus:ring-primary/40 transition-fluid" />
            </div>

            {/* Notes */}
            <div className="sm:col-span-2">
              <label className="text-xs text-on-surface-faint uppercase tracking-widest font-medium block mb-1">Notes</label>
              <textarea
                rows={2}
                value={form.notes ?? ''}
                onChange={field('notes')}
                placeholder="Any additional notes..."
                className="w-full bg-surface-container-highest/40 rounded-xl px-3 py-2 text-base text-on-surface placeholder:text-on-surface-faint outline-none focus:ring-1 focus:ring-primary/40 transition-fluid resize-none"
              />
            </div>
          </div>

          <div className="flex justify-end">
            <button
              onClick={handleSave}
              disabled={upsert.isPending}
              className="flex items-center gap-2 px-4 py-2 rounded-xl bg-primary/10 text-primary text-sm font-medium hover:bg-primary/20 transition-fluid disabled:opacity-50"
            >
              {saved ? <><Check size={14} /> Saved</> : upsert.isPending ? 'Saving...' : 'Save'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

export default function TankTab() {
  return (
    <div className="space-y-4 p-4">
      <p className="text-xs text-on-surface-faint uppercase tracking-widest font-medium px-1">Tank Profile</p>
      <TankProfileForm section="display" title="Display Tank" />
      <TankProfileForm section="sump" title="Sump" />
    </div>
  )
}

