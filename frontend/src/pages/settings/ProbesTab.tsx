import { cn } from '@/lib/utils'
import { Settings as SettingsIcon,ArrowRight } from 'lucide-react'
import { useProbeConfigs,useUpdateProbeConfig } from '@/hooks/useSettings'
import { useProbes } from '@/hooks/useProbes'
import type { ProbeConfig } from '@/api/types'
import { EditableCell,UnitSelect,getCategoryIcon,CategorySelect,LoadingState,EmptyState } from './_shared'

// =============================================================================
// Probes Tab — table for configuration (no drag-and-drop)
// =============================================================================

function useMergedProbeConfigs(
  probes: { name: string; display_name: string; type: string; unit: string; input_category?: string; on_label?: string | null; off_label?: string | null; is_binary?: boolean; hidden?: boolean }[] | undefined,
  configs: ProbeConfig[] | undefined,
): ProbeConfig[] {
  if (!probes) return configs ?? []

  const configMap = new Map((configs ?? []).map((c) => [c.probe_name, c]))

  return probes.map((p) => {
    const existing = configMap.get(p.name)
    return {
      probe_name: p.name,
      display_name: existing?.display_name ?? p.display_name ?? p.name,
      unit_override: existing?.unit_override ?? p.unit ?? '',
      min_normal: existing?.min_normal ?? null,
      max_normal: existing?.max_normal ?? null,
      min_warning: existing?.min_warning ?? null,
      max_warning: existing?.max_warning ?? null,
      device_id: existing?.device_id ?? null,
      input_category: (existing?.input_category ?? p.input_category ?? 'probe') as ProbeConfig['input_category'],
      on_label: existing?.on_label ?? p.on_label ?? null,
      off_label: existing?.off_label ?? p.off_label ?? null,
      ok_value: existing?.ok_value ?? null,
      is_binary: existing?.is_binary ?? p.is_binary ?? false,
      hidden: existing?.hidden ?? p.hidden ?? false,
    }
  })
}

export default function ProbesTab() {
  const { data: configData, isLoading: configsLoading } = useProbeConfigs()
  const { data: probesData, isLoading: probesLoading } = useProbes()
  const updateMutation = useUpdateProbeConfig()

  const isLoading = configsLoading || probesLoading
  const allItems = useMergedProbeConfigs(probesData?.probes, configData?.configs)

  // Split into analog probes and binary inputs.
  const analogProbes = allItems.filter((c) => !c.is_binary)
  const binaryInputs = allItems.filter((c) => c.is_binary)

  // Map probe name → raw type so we can identify mis-categorized digital probes.
  const probeTypeMap = new Map((probesData?.probes ?? []).map((p) => [p.name, p.type]))

  function handleUpdate(name: string, field: keyof ProbeConfig, raw: string | boolean | null) {
    const numericFields: (keyof ProbeConfig)[] = ['min_normal', 'max_normal', 'min_warning', 'max_warning']
    const value =
      typeof raw === 'boolean' ? raw
      : numericFields.includes(field) ? (raw === '' || raw === null ? null : Number(raw))
      : raw
    updateMutation.mutate({ name, config: { [field]: value } })
  }

  if (isLoading) return <LoadingState label="Loading probe configs..." />

  if (allItems.length === 0) {
    return (
      <EmptyState
        icon={<SettingsIcon size={32} />}
        message="No probes found. Probes will appear here after the first poll."
      />
    )
  }

  return (
    <div className="space-y-6 pt-5">
      {analogProbes.length > 0 && (
        <div>
          <p className="text-xs text-on-surface-dim uppercase tracking-widest font-semibold sm:px-5 mb-3">Analog Probes</p>
          {/* Mobile: individual cards */}
          <div className="sm:hidden space-y-3">
            {analogProbes.map((c) => {
              const isDigital = probeTypeMap.get(c.probe_name) === 'digital'
              return (
                <div key={c.probe_name} className="bg-surface-container-high rounded-2xl p-4 space-y-4">
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-sm font-semibold text-on-surface">{c.probe_name}</span>
                    {isDigital && (
                      <button
                        onClick={() => handleUpdate(c.probe_name, 'is_binary', true)}
                        className="inline-flex items-center gap-1 text-xs text-on-surface-faint hover:text-primary transition-fluid whitespace-nowrap"
                        title="Move to Binary Inputs"
                      >
                        <ArrowRight size={11} />
                        <span>Binary</span>
                      </button>
                    )}
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <p className="text-xs text-on-surface-faint uppercase tracking-wider mb-1">Display Name</p>
                      <EditableCell value={c.display_name} onSave={(v) => handleUpdate(c.probe_name, 'display_name', v)} />
                    </div>
                    <div>
                      <p className="text-xs text-on-surface-faint uppercase tracking-wider mb-1">Unit</p>
                      <UnitSelect value={c.unit_override} onSave={(v) => handleUpdate(c.probe_name, 'unit_override', v)} />
                    </div>
                  </div>
                  <div className="space-y-2.5">
                    <div>
                      <p className="text-xs text-on-surface-faint uppercase tracking-wider mb-1">Normal Range</p>
                      <div className="flex items-center gap-2">
                        <div className="flex-1"><EditableCell value={c.min_normal} type="number" onSave={(v) => handleUpdate(c.probe_name, 'min_normal', v)} /></div>
                        <span className="text-on-surface-faint text-xs shrink-0">to</span>
                        <div className="flex-1"><EditableCell value={c.max_normal} type="number" onSave={(v) => handleUpdate(c.probe_name, 'max_normal', v)} /></div>
                      </div>
                    </div>
                    <div>
                      <p className="text-xs text-on-surface-faint uppercase tracking-wider mb-1">Warn Range</p>
                      <div className="flex items-center gap-2">
                        <div className="flex-1"><EditableCell value={c.min_warning} type="number" onSave={(v) => handleUpdate(c.probe_name, 'min_warning', v)} /></div>
                        <span className="text-on-surface-faint text-xs shrink-0">to</span>
                        <div className="flex-1"><EditableCell value={c.max_warning} type="number" onSave={(v) => handleUpdate(c.probe_name, 'max_warning', v)} /></div>
                      </div>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
          {/* Desktop table */}
          <div className="hidden sm:block overflow-x-auto">
          <table className="w-full min-w-[750px]">
            <thead>
              <tr className="bg-surface-container-high">
                {['Probe', 'Display Name', 'Unit', 'Min Normal', 'Max Normal', 'Min Warn', 'Max Warn', ''].map((h) => (
                  <th key={h} className="text-left py-3.5 px-5 text-xs font-semibold text-on-surface-dim uppercase tracking-widest">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {analogProbes.map((c) => {
                const isDigital = probeTypeMap.get(c.probe_name) === 'digital'
                return (
                  <tr key={c.probe_name} className="transition-fluid hover:bg-surface-container-high/50">
                    <td className="py-2 px-5 text-sm font-medium text-on-surface">{c.probe_name}</td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.display_name} onSave={(v) => handleUpdate(c.probe_name, 'display_name', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <UnitSelect value={c.unit_override} onSave={(v) => handleUpdate(c.probe_name, 'unit_override', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.min_normal} type="number" onSave={(v) => handleUpdate(c.probe_name, 'min_normal', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.max_normal} type="number" onSave={(v) => handleUpdate(c.probe_name, 'max_normal', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.min_warning} type="number" onSave={(v) => handleUpdate(c.probe_name, 'min_warning', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.max_warning} type="number" onSave={(v) => handleUpdate(c.probe_name, 'max_warning', v)} />
                    </td>
                    <td className="py-2 px-5">
                      {isDigital && (
                        <button
                          onClick={() => handleUpdate(c.probe_name, 'is_binary', true)}
                          className="inline-flex items-center gap-1 text-xs text-on-surface-faint hover:text-primary transition-fluid whitespace-nowrap"
                          title="Move to Binary Inputs"
                        >
                          <ArrowRight size={11} />
                          <span>binary</span>
                        </button>
                      )}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
          </div>
        </div>
      )}

      {binaryInputs.length > 0 && (
        <div>
          <div className="flex items-baseline justify-between sm:px-5 mb-3">
            <p className="text-xs text-on-surface-dim uppercase tracking-widest font-semibold">Binary Inputs</p>
            <p className="text-xs text-on-surface-faint hidden sm:block">Click a cell to edit · Category controls dashboard icon &amp; alert behavior</p>
          </div>
          {/* Mobile: individual cards */}
          <div className="sm:hidden space-y-3">
            {binaryInputs.filter((c) => c.probe_name !== '').map((c) => {
              const { Icon, color, bg } = getCategoryIcon(c.input_category)
              return (
                <div key={c.probe_name} className="bg-surface-container-high rounded-2xl p-4 space-y-3">
                  <div className="flex items-center justify-between gap-2">
                    <div className="flex items-center gap-2.5">
                      <div className={cn('w-8 h-8 rounded-xl flex items-center justify-center shrink-0', bg)}>
                        <Icon size={14} className={color} />
                      </div>
                      <span className="text-sm font-semibold text-on-surface">{c.probe_name}</span>
                    </div>
                    <button
                      onClick={() => handleUpdate(c.probe_name, 'hidden', !c.hidden)}
                      className={cn(
                        'text-xs px-2.5 py-1 rounded-full transition-fluid shrink-0 uppercase tracking-wider font-medium',
                        c.hidden
                          ? 'bg-surface-container-highest text-on-surface-faint'
                          : 'bg-secondary/15 text-secondary',
                      )}
                    >
                      {c.hidden ? 'Hidden' : 'Visible'}
                    </button>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <p className="text-xs text-on-surface-faint uppercase tracking-wider mb-1">Category</p>
                      <CategorySelect value={c.input_category} onSave={(v) => handleUpdate(c.probe_name, 'input_category', v)} />
                    </div>
                    <div>
                      <p className="text-xs text-on-surface-faint uppercase tracking-wider mb-1">Display Name</p>
                      <EditableCell value={c.display_name} onSave={(v) => handleUpdate(c.probe_name, 'display_name', v)} />
                    </div>
                    <div>
                      <p className="text-xs text-on-surface-faint uppercase tracking-wider mb-1">On Label</p>
                      <EditableCell value={c.on_label ?? ''} onSave={(v) => handleUpdate(c.probe_name, 'on_label', v)} />
                    </div>
                    <div>
                      <p className="text-xs text-on-surface-faint uppercase tracking-wider mb-1">Off Label</p>
                      <EditableCell value={c.off_label ?? ''} onSave={(v) => handleUpdate(c.probe_name, 'off_label', v)} />
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
          {/* Desktop table */}
          <div className="hidden sm:block overflow-x-auto">
          <table className="w-full min-w-[640px]">
            <thead>
              <tr className="bg-surface-container-high">
                {['Input', 'Category', 'Display Name', 'On Label', 'Off Label', 'Hidden'].map((h) => (
                  <th key={h} className="text-left py-3.5 px-5 text-xs font-semibold text-on-surface-dim uppercase tracking-widest">
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {binaryInputs.filter((c) => c.probe_name !== '').map((c) => {
                const { Icon, color, bg } = getCategoryIcon(c.input_category)
                return (
                  <tr key={c.probe_name} className="transition-fluid hover:bg-surface-container-high/50">
                    <td className="py-2 px-5">
                      <div className="flex items-center gap-2.5">
                        <div className={cn('w-7 h-7 rounded-lg flex items-center justify-center shrink-0', bg)}>
                          <Icon size={13} className={color} />
                        </div>
                        <span className="text-sm font-medium text-on-surface">{c.probe_name}</span>
                      </div>
                    </td>
                    <td className="py-2 px-5">
                      <CategorySelect value={c.input_category} onSave={(v) => handleUpdate(c.probe_name, 'input_category', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.display_name} onSave={(v) => handleUpdate(c.probe_name, 'display_name', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.on_label ?? ''} onSave={(v) => handleUpdate(c.probe_name, 'on_label', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <EditableCell value={c.off_label ?? ''} onSave={(v) => handleUpdate(c.probe_name, 'off_label', v)} />
                    </td>
                    <td className="py-2 px-5">
                      <button
                        onClick={() => handleUpdate(c.probe_name, 'hidden', !c.hidden)}
                        className={cn(
                          'text-xs px-2 py-1 rounded-full transition-fluid',
                          c.hidden
                            ? 'bg-surface-container-high text-on-surface-faint'
                            : 'bg-secondary/15 text-secondary',
                        )}
                      >
                        {c.hidden ? 'Hidden' : 'Visible'}
                      </button>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
          </div>
        </div>
      )}
    </div>
  )
}

