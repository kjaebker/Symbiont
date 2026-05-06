import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getDevices,
  createDevice,
  updateDevice,
  deleteDevice,
  setDeviceProbes,
  setDeviceOutlets,
  getDeviceSuggestions,
} from '@/api/client'
import { qk } from '@/api/queryKeys'
import type { Device, DeviceOutlet } from '@/api/types'

export function useDevices() {
  return useQuery({
    queryKey: qk.devices.all,
    queryFn: getDevices,
    staleTime: 10_000,
    notifyOnChangeProps: ['data', 'error'],
  })
}

export function useCreateDevice() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (device: Omit<Device, 'id' | 'created_at' | 'updated_at'>) => createDevice(device),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.devices.all })
      queryClient.invalidateQueries({ queryKey: qk.probes.configs })
      queryClient.invalidateQueries({ queryKey: qk.outlets.configs })
    },
  })
}

export function useUpdateDevice() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, device }: { id: number; device: Partial<Device> }) => updateDevice(id, device),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.devices.all })
      queryClient.invalidateQueries({ queryKey: qk.probes.configs })
      queryClient.invalidateQueries({ queryKey: qk.outlets.configs })
      queryClient.invalidateQueries({ queryKey: qk.probes.all })
      queryClient.invalidateQueries({ queryKey: qk.outlets.all })
    },
  })
}

export function useDeleteDevice() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteDevice(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.devices.all })
    },
  })
}

export function useSetDeviceProbes() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, probeNames }: { id: number; probeNames: string[] }) => setDeviceProbes(id, probeNames),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.devices.all })
      queryClient.invalidateQueries({ queryKey: qk.probes.configs })
      queryClient.invalidateQueries({ queryKey: qk.probes.all })
    },
  })
}

export function useSetDeviceOutlets() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, outlets }: { id: number; outlets: DeviceOutlet[] }) => setDeviceOutlets(id, outlets),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.devices.all })
    },
  })
}

export function useDeviceSuggestions() {
  return useQuery({
    queryKey: qk.devices.suggestions,
    queryFn: getDeviceSuggestions,
    staleTime: 30_000,
    enabled: false, // Only fetch on demand.
  })
}
