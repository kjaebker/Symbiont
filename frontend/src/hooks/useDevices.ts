import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getDevices,
  createDevice,
  updateDevice,
  deleteDevice,
  setDeviceProbes,
  getDeviceSuggestions,
} from '@/api/client'
import type { Device } from '@/api/types'

export function useDevices() {
  return useQuery({
    queryKey: ['devices'],
    queryFn: getDevices,
    staleTime: 10_000,
  })
}

export function useCreateDevice() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (device: Omit<Device, 'id' | 'created_at' | 'updated_at'>) => createDevice(device),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] })
      queryClient.invalidateQueries({ queryKey: ['probeConfigs'] })
      queryClient.invalidateQueries({ queryKey: ['outletConfigs'] })
    },
  })
}

export function useUpdateDevice() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, device }: { id: number; device: Partial<Device> }) => updateDevice(id, device),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] })
      queryClient.invalidateQueries({ queryKey: ['probeConfigs'] })
      queryClient.invalidateQueries({ queryKey: ['outletConfigs'] })
      queryClient.invalidateQueries({ queryKey: ['probes'] })
      queryClient.invalidateQueries({ queryKey: ['outlets'] })
    },
  })
}

export function useDeleteDevice() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteDevice(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] })
    },
  })
}

export function useSetDeviceProbes() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, probeNames }: { id: number; probeNames: string[] }) => setDeviceProbes(id, probeNames),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['devices'] })
      queryClient.invalidateQueries({ queryKey: ['probeConfigs'] })
      queryClient.invalidateQueries({ queryKey: ['probes'] })
    },
  })
}

export function useDeviceSuggestions() {
  return useQuery({
    queryKey: ['devices', 'suggestions'],
    queryFn: getDeviceSuggestions,
    staleTime: 30_000,
    enabled: false, // Only fetch on demand.
  })
}
