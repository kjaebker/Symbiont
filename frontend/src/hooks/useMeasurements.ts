import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getMeasurements,
  getMeasurementParameters,
  createMeasurement,
  updateMeasurement,
  deleteMeasurement,
} from '@/api/client'

export function useMeasurementParameters() {
  return useQuery({
    queryKey: ['measurement-parameters'],
    queryFn: getMeasurementParameters,
    staleTime: Infinity, // parameters are seeded at startup and rarely change
  })
}

export function useMeasurements(params?: {
  parameter?: string
  from?: string
  to?: string
  limit?: number
}) {
  return useQuery({
    queryKey: ['measurements', params],
    queryFn: () => getMeasurements(params),
    staleTime: 30_000,
  })
}

export function useCreateMeasurement() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: createMeasurement,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['measurements'] })
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
      queryClient.invalidateQueries({ queryKey: ['measurements'] })
    },
  })
}

export function useDeleteMeasurement() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => deleteMeasurement(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['measurements'] })
    },
  })
}
