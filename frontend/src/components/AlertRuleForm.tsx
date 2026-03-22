import { useState, useEffect } from 'react'
import { X } from 'lucide-react'
import { useProbes } from '@/hooks/useProbes'
import type { AlertRule } from '@/api/types'
import { cn } from '@/lib/utils'

interface AlertRuleFormProps {
  rule?: AlertRule
  onSubmit: (rule: Omit<AlertRule, 'id' | 'created_at'>) => void
  onClose: () => void
}

const conditions = [
  { value: 'above', label: 'Above' },
  { value: 'below', label: 'Below' },
  { value: 'outside_range', label: 'Outside Range' },
] as const

const severities = [
  { value: 'warning', label: 'Warning' },
  { value: 'critical', label: 'Critical' },
] as const

export function AlertRuleForm({ rule, onSubmit, onClose }: AlertRuleFormProps) {
  const { data } = useProbes()
  const probes = data?.probes ?? []

  const [probeName, setProbeName] = useState(rule?.probe_name ?? '')
  const [condition, setCondition] = useState<AlertRule['condition']>(rule?.condition ?? 'above')
  const [thresholdLow, setThresholdLow] = useState(rule?.threshold_low?.toString() ?? '')
  const [thresholdHigh, setThresholdHigh] = useState(rule?.threshold_high?.toString() ?? '')
  const [severity, setSeverity] = useState<AlertRule['severity']>(rule?.severity ?? 'warning')
  const [cooldown, setCooldown] = useState(rule?.cooldown_minutes?.toString() ?? '15')
  const [enabled, setEnabled] = useState(rule?.enabled ?? true)
  const [error, setError] = useState('')

  // Default to first probe if none selected
  useEffect(() => {
    if (!probeName && probes.length > 0) {
      setProbeName(probes[0].name)
    }
  }, [probeName, probes])

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    if (!probeName) {
      setError('Select a probe')
      return
    }

    const low = thresholdLow ? parseFloat(thresholdLow) : null
    const high = thresholdHigh ? parseFloat(thresholdHigh) : null

    if (condition === 'above' && high == null) {
      setError('Threshold is required for "above" condition')
      return
    }
    if (condition === 'below' && low == null) {
      setError('Threshold is required for "below" condition')
      return
    }
    if (condition === 'outside_range' && (low == null || high == null)) {
      setError('Both low and high thresholds are required for "outside range"')
      return
    }

    onSubmit({
      probe_name: probeName,
      condition,
      threshold_low: low,
      threshold_high: high,
      severity,
      cooldown_minutes: parseInt(cooldown) || 15,
      enabled,
    })
  }

  const inputClass =
    'w-full bg-surface-container-high text-on-surface text-sm rounded-xl px-3 py-2 outline-none focus:ring-1 focus:ring-primary/30 transition-fluid'
  const labelClass = 'text-xs text-on-surface-faint uppercase tracking-widest font-medium mb-1.5 block'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
      <div className="bg-surface-container rounded-2xl p-6 w-full max-w-md shadow-abyss mx-4">
        <div className="flex items-center justify-between mb-5">
          <h2 className="text-lg font-semibold text-on-surface">
            {rule ? 'Edit Alert Rule' : 'New Alert Rule'}
          </h2>
          <button
            onClick={onClose}
            className="text-on-surface-faint hover:text-on-surface transition-fluid cursor-pointer"
          >
            <X size={18} />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Probe */}
          <div>
            <label className={labelClass}>Probe</label>
            <select
              value={probeName}
              onChange={(e) => setProbeName(e.target.value)}
              className={inputClass}
            >
              {probes.map((p) => (
                <option key={p.name} value={p.name}>
                  {p.display_name}
                </option>
              ))}
            </select>
          </div>

          {/* Condition */}
          <div>
            <label className={labelClass}>Condition</label>
            <div className="flex gap-1.5">
              {conditions.map((c) => (
                <button
                  key={c.value}
                  type="button"
                  onClick={() => setCondition(c.value)}
                  className={cn(
                    'flex-1 py-1.5 rounded-lg text-xs font-semibold uppercase tracking-wider transition-fluid cursor-pointer',
                    condition === c.value
                      ? 'bg-primary/20 text-primary'
                      : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim',
                  )}
                >
                  {c.label}
                </button>
              ))}
            </div>
          </div>

          {/* Thresholds */}
          <div className="grid grid-cols-2 gap-3">
            {(condition === 'below' || condition === 'outside_range') && (
              <div>
                <label className={labelClass}>
                  {condition === 'below' ? 'Threshold' : 'Low'}
                </label>
                <input
                  type="number"
                  step="any"
                  value={thresholdLow}
                  onChange={(e) => setThresholdLow(e.target.value)}
                  className={inputClass}
                  placeholder="0.0"
                />
              </div>
            )}
            {(condition === 'above' || condition === 'outside_range') && (
              <div>
                <label className={labelClass}>
                  {condition === 'above' ? 'Threshold' : 'High'}
                </label>
                <input
                  type="number"
                  step="any"
                  value={thresholdHigh}
                  onChange={(e) => setThresholdHigh(e.target.value)}
                  className={inputClass}
                  placeholder="0.0"
                />
              </div>
            )}
          </div>

          {/* Severity */}
          <div>
            <label className={labelClass}>Severity</label>
            <div className="flex gap-1.5">
              {severities.map((s) => (
                <button
                  key={s.value}
                  type="button"
                  onClick={() => setSeverity(s.value)}
                  className={cn(
                    'flex-1 py-1.5 rounded-lg text-xs font-semibold uppercase tracking-wider transition-fluid cursor-pointer',
                    severity === s.value
                      ? s.value === 'critical'
                        ? 'bg-tertiary/20 text-tertiary'
                        : 'bg-amber-400/20 text-amber-400'
                      : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim',
                  )}
                >
                  {s.label}
                </button>
              ))}
            </div>
          </div>

          {/* Cooldown */}
          <div>
            <label className={labelClass}>Cooldown (minutes)</label>
            <input
              type="number"
              min="1"
              value={cooldown}
              onChange={(e) => setCooldown(e.target.value)}
              className={inputClass}
            />
          </div>

          {/* Enabled */}
          <div className="flex items-center justify-between">
            <label className={labelClass + ' !mb-0'}>Enabled</label>
            <button
              type="button"
              onClick={() => setEnabled(!enabled)}
              className={cn(
                'relative w-10 h-5 rounded-full transition-fluid cursor-pointer',
                enabled ? 'bg-primary' : 'bg-surface-container-highest',
              )}
            >
              <span
                className={cn(
                  'absolute top-0.5 h-4 w-4 rounded-full bg-on-surface transition-fluid',
                  enabled ? 'left-5.5' : 'left-0.5',
                )}
              />
            </button>
          </div>

          {error && (
            <p className="text-xs text-tertiary">{error}</p>
          )}

          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 py-2 rounded-xl text-sm font-medium text-on-surface-dim bg-surface-container-high hover:bg-surface-container-highest transition-fluid cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="submit"
              className="flex-1 py-2 rounded-xl text-sm font-semibold text-on-primary bg-gradient-to-r from-primary to-primary-container hover:shadow-glow-primary transition-fluid cursor-pointer"
            >
              {rule ? 'Save Changes' : 'Create Rule'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
