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
    onSuccess: (_data, { name, active }) => {
      // Optimistically set the expected state immediately so the UI updates
      // without waiting for the Apex to reflect the change in its status poll.
      queryClient.setQueryData(['feed'], { name: active ? name : 0, active: active ? 1 : 0 })
      // Then confirm with the real state in the background.
      queryClient.invalidateQueries({ queryKey: ['feed'] })
    },
  })
}
