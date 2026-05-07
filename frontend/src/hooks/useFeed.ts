import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getFeedStatus, setFeedMode } from '@/api/client'
import { qk } from '@/api/queryKeys'

export function useFeedStatus() {
  return useQuery({
    queryKey: qk.feed.current,
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
      queryClient.setQueryData(qk.feed.current, { name: active ? name : 0, active: active ? 1 : 0 })
      // For cancel, delay the refetch — the Apex takes a moment to reflect the
      // change and an immediate refetch can overwrite the optimistic clear.
      setTimeout(
        () => queryClient.invalidateQueries({ queryKey: qk.feed.current }),
        active ? 0 : 15_000,
      )
    },
  })
}
