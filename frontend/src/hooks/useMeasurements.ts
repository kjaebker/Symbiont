import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getMeasurements,
  getMeasurementParameters,
  getKitCatalog,
  createMeasurement,
  updateMeasurement,
  deleteMeasurement,
} from '@/api/client'
import { qk } from '@/api/queryKeys'

export function useMeasurementParameters() {
  return useQuery({
    queryKey: qk.measurements.parameters,
    queryFn: getMeasurementParameters,
    staleTime: Infinity, // parameters are seeded at startup and rarely change
  })
}

export function useKitCatalog() {
  return useQuery({
    queryKey: qk.measurements.kits,
    queryFn: getKitCatalog,
    staleTime: Infinity, // kit catalog is static — embedded in the binary
  })
}

export function useMeasurements(params?: {
  parameter?: string
  from?: string
  to?: string
  limit?: number
}) {
  return useQuery({
    queryKey: qk.measurements.list(params),
    queryFn: () => getMeasurements(params),
    staleTime: 30_000,
  })
}

export function useCreateMeasurement() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: createMeasurement,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.measurements.list() })
    },
  })
}

export function useUpdateMeasurement() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      data,
    }: {
      id: number
      data: Parameters<typeof updateMeasurement>[1]
    }) => updateMeasurement(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.measurements.list() })
    },
  })
}

export function useDeleteMeasurement() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteMeasurement(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.measurements.list() })
    },
  })
}
