import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getTankProfile, upsertTankProfile } from '@/api/client'
import type { TankSection, TankProfileInput } from '@/api/client'

export function useTankProfile() {
  return useQuery({
    queryKey: ['tank-profile'],
    queryFn: getTankProfile,
    staleTime: 60_000,
  })
}

export function useUpsertTankProfile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ section, data }: { section: TankSection; data: Partial<TankProfileInput> }) =>
      upsertTankProfile(section, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tank-profile'] })
    },
  })
}
