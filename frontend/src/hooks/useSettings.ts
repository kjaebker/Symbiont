import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getProbeConfigs,
  updateProbeConfig,
  getOutletConfigs,
  updateOutletConfig,
  listTokens,
  createToken,
  revokeToken,
  getBackups,
  triggerBackup,
} from '@/api/client'
import type { ProbeConfig, OutletConfig } from '@/api/types'

// Probe configs

export function useProbeConfigs() {
  return useQuery({
    queryKey: ['probeConfigs'],
    queryFn: getProbeConfigs,
    staleTime: 10_000,
  })
}

export function useUpdateProbeConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ name, config }: { name: string; config: Partial<ProbeConfig> }) =>
      updateProbeConfig(name, config),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['probeConfigs'] })
      queryClient.invalidateQueries({ queryKey: ['probes'] })
    },
  })
}

// Outlet configs

export function useOutletConfigs() {
  return useQuery({
    queryKey: ['outletConfigs'],
    queryFn: getOutletConfigs,
    staleTime: 10_000,
  })
}

export function useUpdateOutletConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, config }: { id: string; config: Partial<OutletConfig> }) =>
      updateOutletConfig(id, config),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['outletConfigs'] })
      queryClient.invalidateQueries({ queryKey: ['outlets'] })
    },
  })
}

// Tokens

export function useTokens() {
  return useQuery({
    queryKey: ['tokens'],
    queryFn: listTokens,
    staleTime: 10_000,
  })
}

export function useCreateToken() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (label: string) => createToken(label),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tokens'] })
    },
  })
}

export function useRevokeToken() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => revokeToken(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tokens'] })
    },
  })
}

// Backups

export function useBackups() {
  return useQuery({
    queryKey: ['backups'],
    queryFn: getBackups,
    staleTime: 30_000,
  })
}

export function useTriggerBackup() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: triggerBackup,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['backups'] })
    },
  })
}
