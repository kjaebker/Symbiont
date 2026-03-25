import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  Flame,
  Waves,
  Sun,
  Filter,
  FlaskConical,
  Droplets,
  Droplet,
  Snowflake,
  Fan,
  Zap,
  Box,
  Power,
} from 'lucide-react'
import { getProbeHistory } from '@/api/client'
import { useSetOutlet } from '@/hooks/useOutlets'
import type { Device, Probe, Outlet, ProbeStatus } from '@/api/types'
import { cn } from '@/lib/utils'
import { Sparkline } from './Sparkline'

const deviceTypeIcons: Record<string, typeof Flame> = {
  heater: Flame,
  pump: Waves,
  wavemaker: Waves,
  light: Sun,
  skimmer: Filter,
  reactor: FlaskConical,
  doser: Droplets,
  ato: Droplet,
  chiller: Snowflake,
  fan: Fan,
  other: Box,
}

const statusColor = {
  normal: 'bg-secondary',
  warning: 'bg-amber-400',
  critical: 'bg-tertiary',
  unknown: 'bg-on-surface-faint',
} as const

const stateLabels: Record<string, string> = {
  ON: 'On',
  OFF: 'Off',
  AON: 'Auto',
  AOF: 'Auto Off',
  TBL: 'Schedule',
}

const stateColors: Record<string, string> = {
  ON: 'text-secondary',
  AON: 'text-primary',
  OFF: 'text-on-surface-dim',
  AOF: 'text-on-surface-dim',
  TBL: 'text-primary',
}

function worstProbeStatus(probes: Probe[]): ProbeStatus {
  if (probes.some((p) => p.status === 'critical')) return 'critical'
  if (probes.some((p) => p.status === 'warning')) return 'warning'
  if (probes.some((p) => p.status === 'normal')) return 'normal'
  return 'unknown'
}

/** Pick the primary probe — prefer the "monitoring" probe (temp, pH, ORP) over power probes. */
function pickPrimaryProbe(probes: Probe[]): Probe | undefined {
  const powerTypes = new Set(['pwr', 'Amps'])
  return (
    probes.find((p) => !powerTypes.has(p.type)) ??
    probes.find((p) => p.type === 'pwr') ??
    probes[0]
  )
}

interface DeviceCardProps {
  device: Device
  probes: Probe[]
  outlet: Outlet | undefined
}

export function DeviceCard({ device, probes, outlet }: DeviceCardProps) {
  const navigate = useNavigate()
  const mutation = useSetOutlet()

  const Icon = deviceTypeIcons[device.device_type ?? ''] ?? Zap
  const status = worstProbeStatus(probes)
  const primaryProbe = pickPrimaryProbe(probes)
  const secondaryProbes = probes.filter((p) => p !== primaryProbe)

  const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString()
  const { data: history } = useQuery({
    queryKey: ['probeSparkline', primaryProbe?.name],
    queryFn: () =>
      getProbeHistory(primaryProbe!.name, { from: twoHoursAgo, interval: '5m' }),
    staleTime: 60_000,
    enabled: !!primaryProbe,
  })

  const sparklineData = history?.data.map((d) => d.value) ?? []

  const isOn = outlet
    ? outlet.state === 'ON' || outlet.state === 'AON' || outlet.state === 'TBL'
    : false
  const isAuto = outlet
    ? outlet.state === 'AON' || outlet.state === 'AOF' || outlet.state === 'TBL'
    : false

  function handleControl(state: 'ON' | 'OFF' | 'AUTO') {
    if (outlet) mutation.mutate({ id: outlet.id, state })
  }

  return (
    <div className="bg-surface-container rounded-2xl p-5 transition-fluid flex flex-col">
      {/* Header: icon + name + type badge + status */}
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div
            className={cn(
              'w-10 h-10 rounded-xl flex items-center justify-center shrink-0',
              outlet && isOn ? 'bg-primary/10' : 'bg-surface-container-highest',
            )}
          >
            <Icon
              size={18}
              className={outlet && isOn ? 'text-primary' : 'text-on-surface-faint'}
            />
          </div>
          <div className="min-w-0">
            <div className="text-sm font-semibold text-on-surface truncate">
              {device.name}
            </div>
            {device.device_type && (
              <span className="text-xs text-primary/80 font-medium uppercase tracking-wider">
                {device.device_type}
              </span>
            )}
          </div>
        </div>
        <span
          className={cn(
            'h-2.5 w-2.5 rounded-full shrink-0 mt-1',
            statusColor[status],
            status === 'normal' && 'animate-bio-pulse',
          )}
        />
      </div>

      {/* Probe readings: primary large left, secondary compact right */}
      {probes.length > 0 && (
        <div className="mb-4">
          <div className="flex items-start justify-between mb-3">
            {/* Primary probe — large hero value */}
            {primaryProbe && (
              <button
                onClick={() =>
                  navigate(`/history?probe=${encodeURIComponent(primaryProbe.name)}`)
                }
                className="text-left group"
              >
                <div className="flex items-baseline gap-1.5">
                  <span className="text-4xl font-bold text-on-surface tracking-tight text-glow-primary group-hover:text-primary transition-fluid">
                    {primaryProbe.value.toFixed(primaryProbe.type === 'pH' ? 2 : 1)}
                  </span>
                  <span className="text-lg text-on-surface-dim font-light">
                    {primaryProbe.unit}
                  </span>
                </div>
              </button>
            )}

            {/* Secondary probes — compact stack */}
            {secondaryProbes.length > 0 && (
              <div className="space-y-1.5 text-right">
                {secondaryProbes.map((probe) => (
                  <button
                    key={probe.name}
                    onClick={() =>
                      navigate(`/history?probe=${encodeURIComponent(probe.name)}`)
                    }
                    className="flex items-center justify-end gap-2 w-full group"
                  >
                    <span className="text-sm font-bold text-on-surface group-hover:text-primary transition-fluid">
                      {probe.value.toFixed(probe.type === 'pH' ? 2 : 1)}
                    </span>
                    <span className="text-xs text-on-surface-dim">
                      {probe.unit}
                    </span>
                    <span
                      className={cn(
                        'h-1.5 w-1.5 rounded-full',
                        statusColor[probe.status],
                      )}
                    />
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Sparkline for primary probe */}
          {primaryProbe && (
            <div className="h-10">
              <Sparkline data={sparklineData} color="#3adffa" />
            </div>
          )}
        </div>
      )}

      {/* Spacer to push controls to bottom */}
      <div className="flex-1" />

      {/* Outlet controls */}
      {outlet && (
        <div>
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-1.5">
              {isOn ? (
                <Zap size={12} className="text-primary" />
              ) : (
                <Power size={12} className="text-on-surface-faint" />
              )}
              <span
                className={cn(
                  'text-xs font-medium uppercase tracking-wider',
                  stateColors[outlet.state] ?? 'text-on-surface-dim',
                )}
              >
                {stateLabels[outlet.state] ?? outlet.state}
              </span>
            </div>
          </div>
          <div className="flex gap-1.5">
            {(['OFF', 'ON', 'AUTO'] as const).map((s) => {
              const active =
                (s === 'OFF' && (outlet.state === 'OFF' || outlet.state === 'AOF')) ||
                (s === 'ON' && outlet.state === 'ON') ||
                (s === 'AUTO' && isAuto)

              return (
                <button
                  key={s}
                  onClick={() => handleControl(s)}
                  disabled={mutation.isPending}
                  className={cn(
                    'flex-1 py-1.5 rounded-lg text-xs font-semibold uppercase tracking-wider transition-fluid',
                    active
                      ? s === 'AUTO'
                        ? 'bg-primary text-on-primary'
                        : s === 'ON'
                          ? 'bg-secondary text-on-secondary'
                          : 'bg-surface-container-highest text-on-surface'
                      : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim',
                  )}
                >
                  {s}
                </button>
              )
            })}
          </div>
          {mutation.isError && (
            <p className="text-xs text-tertiary mt-2">
              Failed to set state. Try again.
            </p>
          )}
        </div>
      )}

      {/* Empty state when no probes and no outlet */}
      {probes.length === 0 && !outlet && (
        <p className="text-xs text-on-surface-faint text-center py-2">
          No probes or outlet linked
        </p>
      )}
    </div>
  )
}
