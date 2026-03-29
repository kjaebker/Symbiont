import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getFeedStatus, setFeedMode } from '@/api/client'

export function useFeedStatus() {
  return useQuery({
    queryKey: ['feed'],
    queryFn: getFeedStatus,
    staleTime: 10_000,
    refetchInterval: 15_000,
  })
}

export function useSetFeedMode() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ name, active }: { name: number; active: boolean }) =>
      setFeedMode(name, active),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['feed'] })
    },
  })
}
