import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getOutlets, setOutletState, getOutletEvents } from '@/api/client'
import type { Outlet } from '@/api/types'

export function useOutlets() {
  return useQuery({
    queryKey: ['outlets'],
    queryFn: getOutlets,
    staleTime: 10_000,
    refetchInterval: false,
  })
}

export function useSetOutlet() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, state }: { id: string; state: 'ON' | 'OFF' | 'AUTO' }) =>
      setOutletState(id, state),
    onMutate: async ({ id, state }) => {
      await queryClient.cancelQueries({ queryKey: ['outlets'] })
      const previous = queryClient.getQueryData<{ outlets: Outlet[] }>(['outlets'])

      if (previous) {
        const mapped: Record<string, string> = { ON: 'ON', OFF: 'OFF', AUTO: 'AON' }
        queryClient.setQueryData<{ outlets: Outlet[] }>(['outlets'], {
          outlets: previous.outlets.map((o) =>
            o.id === id ? { ...o, state: (mapped[state] ?? state) as Outlet['state'] } : o,
          ),
        })
      }

      return { previous }
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(['outlets'], context.previous)
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ['outlets'] })
      queryClient.invalidateQueries({ queryKey: ['outletEvents'] })
    },
  })
}

export function useOutletEvents(params?: { outlet_id?: string; limit?: number }) {
  return useQuery({
    queryKey: ['outletEvents', params],
    queryFn: () => getOutletEvents(params),
    staleTime: 10_000,
  })
}
