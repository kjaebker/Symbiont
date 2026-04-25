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
import { inferPersonality } from '@/lib/devicePersonality'
import { Sparkline } from './Sparkline'
import { BioDot } from './BioDot'
import { getCategory } from './ProbeCard'

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

const statusDotHex: Record<ProbeStatus, string> = {
  normal: '#6dfe9c',
  warning: '#fbbf24',
  critical: '#ff8796',
  unknown: '#5a6080',
}

const stateLabels: Record<string, string> = {
  ON: 'ON',
  OFF: 'OFF',
  AON: 'AUTO',
  AOF: 'AUTO OFF',
  TBL: 'SCHEDULE',
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

/** Pick the primary probe — prefer monitoring probes (temp, pH, binary sensors) over power probes. */
export function pickPrimaryProbe(probes: Probe[]): Probe | undefined {
  const powerTypes = new Set(['pwr', 'Amps'])
  return (
    probes.find((p) => !powerTypes.has(p.type)) ??
    probes.find((p) => p.type === 'pwr') ??
    probes[0]
  )
}

function getBinaryLabel(probe: Probe): string {
  return probe.value !== 0 ? (probe.on_label ?? 'On') : (probe.off_label ?? 'Off')
}

function getBinaryColor(probe: Probe): string {
  if (probe.status === 'critical') return 'text-tertiary'
  if (probe.status === 'warning') return 'text-amber-400'
  return 'text-secondary'
}

interface DeviceCardProps {
  device: Device
  probes: Probe[]
  outlet: Outlet | undefined
  controlsLocked?: boolean
}

export function DeviceCard({ device, probes, outlet, controlsLocked = false }: DeviceCardProps) {
  const navigate = useNavigate()
  const mutation = useSetOutlet()

  const Icon = deviceTypeIcons[device.device_type ?? ''] ?? Zap
  const personality = inferPersonality(device.name, device.device_type)
  const status = worstProbeStatus(probes)
  const primaryProbe = pickPrimaryProbe(probes)
  const secondaryProbes = probes.filter((p) => p !== primaryProbe)

  const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString()
  const { data: history } = useQuery({
    queryKey: ['probeSparkline', primaryProbe?.name],
    queryFn: () =>
      getProbeHistory(primaryProbe!.name, { from: twoHoursAgo, interval: '5m' }),
    staleTime: 60_000,
    enabled: !!primaryProbe && !primaryProbe.is_binary,
  })

  const sparklineData = history?.data.map((d) => d.value) ?? []

  const probeCategory = primaryProbe ? getCategory(primaryProbe.type) : 'power'
  const probeCategoryColors = {
    temperature: { value: 'text-tertiary', glow: 'text-glow-tertiary', sparkline: '#ff8796' },
    chemistry:   { value: 'text-secondary', glow: 'text-glow-secondary', sparkline: '#6dfe9c' },
    power:       { value: 'text-primary', glow: 'text-glow-primary', sparkline: '#3adffa' },
    digital:     { value: 'text-on-surface-dim', glow: '', sparkline: '#8a90a8' },
  }
  const probeColors = probeCategoryColors[probeCategory]

  const isOn = outlet
    ? outlet.state === 'ON' || outlet.state === 'AON' || outlet.state === 'TBL'
    : false
  const isAuto = outlet
    ? outlet.state === 'AON' || outlet.state === 'AOF' || outlet.state === 'TBL'
    : false

  function handleControl(state: 'ON' | 'OFF' | 'AUTO') {
    if (outlet && !controlsLocked) mutation.mutate({ id: outlet.id, state })
  }

  return (
    <div
      className="rounded-2xl p-5 transition-fluid flex flex-col relative overflow-hidden"
      style={{
        background: outlet && isOn
          ? `linear-gradient(145deg, var(--color-surface-container) 20%, ${personality.color}33)`
          : 'var(--color-surface-container)',
      }}
    >
      {/* Personality blob */}
      <div
        style={{
          position: 'absolute',
          top: -20,
          right: -20,
          width: 120,
          height: 120,
          borderRadius: personality.blob,
          background: personality.bg,
          filter: 'blur(20px)',
          pointerEvents: 'none',
        }}
      />

      {/* Header: icon + name + type badge + status */}
      <div className="flex items-start justify-between mb-4 relative">
        <div className="flex items-center gap-3">
          <div
            className="w-10 h-10 flex items-center justify-center shrink-0 transition-fluid"
            style={{
              borderRadius: personality.blob,
              background: outlet && isOn ? personality.bg : 'rgba(255,255,255,0.04)',
              boxShadow: outlet && isOn ? `0 0 14px ${personality.color}33` : 'none',
            }}
          >
            <Icon
              size={18}
              style={{ color: outlet && isOn ? personality.color : 'var(--color-on-surface-faint)' }}
            />
          </div>
          <div className="min-w-0">
            <div className="text-sm font-semibold text-on-surface truncate">
              {device.name}
            </div>
            <div
              className="text-[10px] uppercase tracking-[0.12em] font-semibold"
              style={{ color: personality.color, opacity: 0.75 }}
            >
              {personality.label}
            </div>
          </div>
        </div>
        <BioDot color={statusDotHex[status]} size={10} />
      </div>

      {/* Probe readings: primary large left, secondary compact right */}
      {probes.length > 0 && (
        <div className="mb-4">
          <div className="flex items-start justify-between mb-3">
            {/* Primary probe — large hero value */}
            {primaryProbe && (
              primaryProbe.is_binary ? (
                <div className={`text-4xl font-bold tracking-tight ${getBinaryColor(primaryProbe)}`}>
                  {getBinaryLabel(primaryProbe)}
                </div>
              ) : (
                <button
                  onClick={() =>
                    navigate(`/history?tab=telemetry&probe=${encodeURIComponent(primaryProbe.name)}`)
                  }
                  className="text-left group"
                >
                  <div className="flex items-baseline gap-1.5">
                    <span className={`text-4xl font-bold tracking-tight transition-fluid ${probeColors.value} ${probeColors.glow}`}>
                      {primaryProbe.value.toFixed(primaryProbe.type === 'pH' ? 2 : 1)}
                    </span>
                    <span className="text-lg text-on-surface-dim font-light">
                      {primaryProbe.unit}
                    </span>
                  </div>
                </button>
              )
            )}

            {/* Secondary probes — compact stack */}
            {secondaryProbes.length > 0 && (
              <div className="space-y-1.5 text-right">
                {secondaryProbes.map((probe) => (
                  <button
                    key={probe.name}
                    onClick={() =>
                      !probe.is_binary &&
                      navigate(`/history?tab=telemetry&probe=${encodeURIComponent(probe.name)}`)
                    }
                    className={cn(
                      'flex items-center justify-end gap-2 w-full',
                      !probe.is_binary && 'group',
                    )}
                  >
                    {probe.is_binary ? (
                      <span className={`text-sm font-bold ${getBinaryColor(probe)}`}>
                        {getBinaryLabel(probe)}
                      </span>
                    ) : (
                      <>
                        <span className="text-sm font-bold text-on-surface group-hover:text-primary transition-fluid">
                          {probe.value.toFixed(probe.type === 'pH' ? 2 : 1)}
                        </span>
                        <span className="text-xs text-on-surface-dim">
                          {probe.unit}
                        </span>
                      </>
                    )}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Sparkline for primary probe — omit for binary probes */}
          {primaryProbe && !primaryProbe.is_binary && (
            <div className="h-10">
              <Sparkline data={sparklineData} color={probeColors.sparkline} />
            </div>
          )}
        </div>
      )}

      {/* Spacer to push controls to bottom */}
      <div className="flex-1" />

      {/* Outlet controls */}
      {outlet && (
        <div>
          <div className="flex items-center gap-1.5 mb-1">
            {isOn ? (
              <Zap size={12} className="text-primary" />
            ) : (
              <Power size={12} className="text-on-surface-faint" />
            )}
            <span
              className={cn(
                'text-xs font-semibold uppercase tracking-wider',
                stateColors[outlet.state] ?? 'text-on-surface-dim',
              )}
            >
              {stateLabels[outlet.state] ?? outlet.state}
            </span>
          </div>

          {/* Controls — revealed when unlocked */}
          <div
            className="grid overflow-hidden"
            style={{
              gridTemplateRows: controlsLocked ? '0fr' : '1fr',
              transition: 'grid-template-rows 250ms cubic-bezier(0.65, 0, 0.35, 1)',
            }}
          >
            <div className="min-h-0">
              <div className="flex gap-1.5 pt-2">
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
                        'flex-1 py-2 rounded-xl text-xs font-semibold uppercase tracking-wider transition-fluid',
                        active
                          ? s === 'AUTO'
                            ? 'bg-primary text-on-primary'
                            : s === 'ON'
                              ? 'bg-secondary text-on-secondary'
                              : 'bg-surface-container-highest text-on-surface'
                          : 'bg-surface-container-high text-on-surface-faint hover:text-on-surface-dim hover:bg-surface-container-highest',
                      )}
                    >
                      {s}
                    </button>
                  )
                })}
              </div>
              {mutation.isError && (
                <p className="text-xs text-tertiary mt-2">Failed to set state. Try again.</p>
              )}
            </div>
          </div>
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
