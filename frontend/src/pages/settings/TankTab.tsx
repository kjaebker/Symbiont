import { useState,useEffect } from 'react'
import { cn } from '@/lib/utils'
import { Check,Waves } from 'lucide-react'
import { Button, Field, Input, Select, Textarea, labelClass } from '@/components/ui'
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
        <div className="px-4 pb-4 pt-1 space-y-3">
          <div className="pt-3 grid grid-cols-1 sm:grid-cols-2 gap-3">
            <Field label="Display Name" className="sm:col-span-2">
              <Input
                type="text"
                value={form.display_name ?? ''}
                onChange={field('display_name')}
                placeholder={title}
              />
            </Field>

            <Field label="Type">
              <Select value={form.tank_type ?? ''} onChange={field('tank_type')}>
                <option value="">Select type...</option>
                {(Object.keys(TANK_TYPE_LABELS) as TankType[]).map(t => (
                  <option key={t} value={t}>{TANK_TYPE_LABELS[t]}</option>
                ))}
              </Select>
            </Field>

            <Field label="Volume (gallons)">
              <Input
                type="number"
                min="0"
                step="0.1"
                value={form.volume_gallons ?? ''}
                onChange={numericField('volume_gallons')}
                placeholder="e.g. 75"
              />
            </Field>

            <Field label="Length (in)">
              <Input type="number" min="0" step="0.5" value={form.length_in ?? ''} onChange={numericField('length_in')} placeholder="48" />
            </Field>
            <Field label="Width (in)">
              <Input type="number" min="0" step="0.5" value={form.width_in ?? ''} onChange={numericField('width_in')} placeholder="18" />
            </Field>
            <Field label="Height (in)">
              <Input type="number" min="0" step="0.5" value={form.height_in ?? ''} onChange={numericField('height_in')} placeholder="20" />
            </Field>

            <Field label="Manufacturer">
              <Input type="text" value={form.manufacturer ?? ''} onChange={field('manufacturer')} placeholder="e.g. Red Sea" />
            </Field>
            <Field label="Model">
              <Input type="text" value={form.model ?? ''} onChange={field('model')} placeholder="e.g. Reefer 350" />
            </Field>

            <Field label="Setup Date">
              <Input type="date" value={form.setup_date ?? ''} onChange={field('setup_date')} />
            </Field>

            <Field label="Notes" className="sm:col-span-2">
              <Textarea
                rows={2}
                value={form.notes ?? ''}
                onChange={field('notes')}
                placeholder="Any additional notes..."
              />
            </Field>
          </div>

          <div className="flex justify-end">
            <Button
              variant="primary"
              onClick={handleSave}
              disabled={upsert.isPending}
            >
              {saved ? <><Check size={14} /> Saved</> : upsert.isPending ? 'Saving...' : 'Save'}
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}

export default function TankTab() {
  return (
    <div className="space-y-4 p-4">
      <p className={`${labelClass} px-1`}>Tank Profile</p>
      <TankProfileForm section="display" title="Display Tank" />
      <TankProfileForm section="sump" title="Sump" />
    </div>
  )
}

