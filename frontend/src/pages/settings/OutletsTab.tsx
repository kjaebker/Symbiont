import { Settings as SettingsIcon } from 'lucide-react'
import { useOutletConfigs,useUpdateOutletConfig } from '@/hooks/useSettings'
import { useOutlets } from '@/hooks/useOutlets'
import type { OutletConfig } from '@/api/types'
import { EditableCell,LoadingState,EmptyState } from './_shared'

// =============================================================================
// Outlets Tab — table for configuration (no drag-and-drop)
// =============================================================================

function useMergedOutletConfigs(
  outlets: { id: string; name: string; display_name: string; type: string }[] | undefined,
  configs: OutletConfig[] | undefined,
): (OutletConfig & { outletName: string })[] {
  if (!outlets) return []
  const configMap = new Map((configs ?? []).map((c) => [c.outlet_id, c]))

  return outlets
    .filter((o) => o.type === 'outlet' || o.type === 'virtual')
    .map((o) => {
      const existing = configMap.get(o.id)
      return {
        outlet_id: o.id,
        display_name: existing?.display_name ?? o.display_name ?? o.name,
        icon: existing?.icon ?? '',
        outletName: o.name,
      }
    })
}

export default function OutletsTab() {
  const { data: outletConfigData, isLoading: outletConfigsLoading } = useOutletConfigs()
  const { data: outletsData, isLoading: outletsLoading } = useOutlets()
  const updateOutletMutation = useUpdateOutletConfig()

  const isLoading = outletConfigsLoading || outletsLoading
  const items = useMergedOutletConfigs(outletsData?.outlets, outletConfigData?.configs)

  function handleUpdate(id: string, field: keyof OutletConfig, raw: string) {
    updateOutletMutation.mutate({ id, config: { [field]: raw } })
  }

  if (isLoading) return <LoadingState label="Loading outlet configs..." />

  if (items.length === 0) {
    return (
      <EmptyState
        icon={<SettingsIcon size={32} />}
        message="No outlets found. Outlets will appear here after the first poll."
      />
    )
  }

  return (
    <>
      {/* Mobile: individual cards */}
      <div className="sm:hidden space-y-3">
        {items.map((item) => (
          <div key={item.outlet_id} className="bg-surface-container-high rounded-2xl p-4 space-y-3">
            <div className="flex items-center justify-between gap-2">
              <span className="text-sm font-semibold text-on-surface">{item.outletName}</span>
              <span className="text-xs text-on-surface-faint font-mono">{item.outlet_id}</span>
            </div>
            <div>
              <p className="text-xs text-on-surface-faint uppercase tracking-wider mb-1">Display Name</p>
              <EditableCell value={item.display_name} onSave={(v) => handleUpdate(item.outlet_id, 'display_name', v)} />
            </div>
          </div>
        ))}
      </div>
      {/* Desktop table */}
      <div className="hidden sm:block overflow-x-auto">
        <table className="w-full">
          <thead>
            <tr className="bg-surface-container-high/50">
              {['Outlet', 'Display Name', 'ID'].map((h) => (
                <th key={h} className="text-left py-3 px-4 text-xs font-medium text-on-surface-faint uppercase tracking-widest">
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
                <tr key={item.outlet_id} className="transition-fluid hover:bg-surface-container-high/50">
                  <td className="py-2 px-4 text-sm font-medium text-on-surface">{item.outletName}</td>
                  <td className="py-2 px-4">
                    <EditableCell value={item.display_name} onSave={(v) => handleUpdate(item.outlet_id, 'display_name', v)} />
                  </td>
                  <td className="py-2 px-4 text-xs text-on-surface-faint font-mono">{item.outlet_id}</td>
                </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}

