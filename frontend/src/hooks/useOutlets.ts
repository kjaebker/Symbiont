import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getOutlets, setOutletState, getOutletEvents } from '@/api/client'
import type { Outlet } from '@/api/types'

// How long to keep optimistic state after a mutation before trusting
// the server again. Must be longer than the poller interval (10s).
const MUTATION_GUARD_MS = 30_000

// Track which outlets have pending optimistic overrides and what state
// they should show until the guard window expires.
type OutletOverride = { state: Outlet['state']; until: number }
const overrides = new Map<string, OutletOverride>()

function applyOverrides(outlets: Outlet[]): Outlet[] {
  const now = Date.now()
  for (const [id, override] of overrides) {
    if (now > override.until) {
      overrides.delete(id)
    }
  }
  if (overrides.size === 0) return outlets
  return outlets.map((o) => {
    const ov = overrides.get(o.id)
    return ov ? { ...o, state: ov.state } : o
  })
}

export function useOutlets() {
  return useQuery({
    queryKey: ['outlets'],
    queryFn: getOutlets,
    staleTime: 10_000,
    refetchInterval: 15_000,
    select: (data) => ({
      ...data,
      outlets: applyOverrides(data.outlets),
    }),
  })
}

export function useSetOutlet() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, state }: { id: string; state: 'ON' | 'OFF' | 'AUTO' }) =>
      setOutletState(id, state),
    onMutate: async ({ id, state }) => {
      await queryClient.cancelQueries({ queryKey: ['outlets'] })
      const mapped: Record<string, Outlet['state']> = { ON: 'ON', OFF: 'OFF', AUTO: 'AON' }
      const newState = mapped[state] ?? (state as Outlet['state'])

      // Set the override so all future refetches show this state
      // until the guard window expires.
      overrides.set(id, { state: newState, until: Date.now() + MUTATION_GUARD_MS })

      // Also update the cache immediately for instant feedback.
      const previous = queryClient.getQueryData<{ outlets: Outlet[] }>(['outlets'])
      if (previous) {
        queryClient.setQueryData<{ outlets: Outlet[] }>(['outlets'], {
          outlets: previous.outlets.map((o) =>
            o.id === id ? { ...o, state: newState } : o,
          ),
        })
      }

      return { previous }
    },
    onError: (_err, vars, context) => {
      // Remove the override on error so the real state shows.
      overrides.delete(vars.id)
      if (context?.previous) {
        queryClient.setQueryData(['outlets'], context.previous)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['outletEvents'] })
    },
  })
}

export function useOutletEvents(params?: { outlet_id?: string; initiated_by?: string; limit?: number }) {
  return useQuery({
    queryKey: ['outletEvents', params],
    queryFn: () => getOutletEvents(params),
    staleTime: 10_000,
  })
}
