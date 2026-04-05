import { createContext, useContext } from 'react'
import { useProbes } from '@/hooks/useProbes'
import { useOutlets } from '@/hooks/useOutlets'
import { useDevices } from '@/hooks/useDevices'
import type { Probe, Outlet, Device } from '@/api/types'

interface SystemDataContextValue {
  probesByName: Record<string, Probe>
  outletsById: Record<string, Outlet>
  devicesById: Record<number, Device>
}

const SystemDataContext = createContext<SystemDataContextValue>({
  probesByName: {},
  outletsById: {},
  devicesById: {},
})

export function useSystemData() {
  return useContext(SystemDataContext)
}

export function SystemDataProvider({ children }: { children: React.ReactNode }) {
  const { data: probesData } = useProbes()
  const { data: outletsData } = useOutlets()
  const { data: devicesData } = useDevices()

  const probesByName: Record<string, Probe> = {}
  for (const p of probesData?.probes ?? []) {
    probesByName[p.name] = p
  }

  const outletsById: Record<string, Outlet> = {}
  for (const o of outletsData?.outlets ?? []) {
    outletsById[o.id] = o
  }

  const devicesById: Record<number, Device> = {}
  for (const d of devicesData?.devices ?? []) {
    devicesById[d.id] = d
  }

  return (
    <SystemDataContext.Provider value={{ probesByName, outletsById, devicesById }}>
      {children}
    </SystemDataContext.Provider>
  )
}
