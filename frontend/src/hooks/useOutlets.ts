import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { getOutlets, setOutletState, getOutletHistory } from '@/api/client'
import { qk } from '@/api/queryKeys'
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
    queryKey: qk.outlets.all,
    queryFn: getOutlets,
    staleTime: 10_000,
    refetchInterval: 15_000,
    notifyOnChangeProps: ['data', 'error'],
    select: (data) => ({
      ...data,
      outlets: applyOverrides(data.outlets),
    }),
  })
}

export function useOutletIntensityHistory(
  id: string | null,
  params?: { from?: string; to?: string; interval?: string },
) {
  return useQuery({
    queryKey: qk.outlets.intensityHistory(id ?? '', params),
    queryFn: () => getOutletHistory(id!, params),
    enabled: !!id,
    staleTime: 60_000,
  })
}

export function useSetOutlet() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ id, state }: { id: string; state: 'ON' | 'OFF' | 'AUTO' }) =>
      setOutletState(id, state),
    onMutate: async ({ id, state }) => {
      await queryClient.cancelQueries({ queryKey: qk.outlets.all })
      const mapped: Record<string, Outlet['state']> = { ON: 'ON', OFF: 'OFF', AUTO: 'AON' }
      const newState = mapped[state] ?? (state as Outlet['state'])

      // Set the override so all future refetches show this state
      // until the guard window expires.
      overrides.set(id, { state: newState, until: Date.now() + MUTATION_GUARD_MS })

      // Also update the cache immediately for instant feedback.
      const previous = queryClient.getQueryData<{ outlets: Outlet[] }>(qk.outlets.all)
      if (previous) {
        queryClient.setQueryData<{ outlets: Outlet[] }>(qk.outlets.all, {
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
        queryClient.setQueryData(qk.outlets.all, context.previous)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.events.audit() })
    },
  })
}
