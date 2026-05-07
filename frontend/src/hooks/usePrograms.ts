import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getPrograms, syncPrograms } from '@/api/client'
import { qk } from '@/api/queryKeys'

export function usePrograms() {
  return useQuery({
    queryKey: qk.programs.all,
    queryFn: getPrograms,
    staleTime: 60 * 60 * 1000, // 1 hour — data is DB-backed, changes only via sync
    refetchInterval: false,
  })
}

export function useProgramByDID(did: string) {
  return useQuery({
    queryKey: qk.programs.all,
    queryFn: getPrograms,
    staleTime: 60 * 60 * 1000,
    refetchInterval: false,
    select: (data) => data.programs.find((p) => p.did === did) ?? null,
  })
}

export function useSyncPrograms() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: syncPrograms,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.programs.all })
    },
  })
}
