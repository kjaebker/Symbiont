import { useQuery } from '@tanstack/react-query'
import { getPrograms } from '@/api/client'

export function usePrograms() {
  return useQuery({
    queryKey: ['programs'],
    queryFn: getPrograms,
    staleTime: 5 * 60 * 1000,
    refetchInterval: false,
  })
}

export function useProgramByDID(did: string) {
  return useQuery({
    queryKey: ['programs'],
    queryFn: getPrograms,
    staleTime: 5 * 60 * 1000,
    refetchInterval: false,
    select: (data) => data.programs.find((p) => p.did === did) ?? null,
  })
}
