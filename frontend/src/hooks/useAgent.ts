import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  getAgentSettings,
  updateAgentSettings,
  getAgentContext,
  getAgentSkills,
} from '@/api/client'
import type { AgentSettings } from '@/api/types'

export function useAgentSettings() {
  return useQuery({
    queryKey: ['agent-settings'],
    queryFn: getAgentSettings,
    staleTime: 60_000,
  })
}

export function useUpdateAgentSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (patch: Partial<AgentSettings>) => updateAgentSettings(patch),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['agent-settings'] })
      qc.invalidateQueries({ queryKey: ['agent-context'] })
    },
  })
}

export function useAgentContext() {
  return useQuery({
    queryKey: ['agent-context'],
    queryFn: getAgentContext,
    staleTime: 30_000,
  })
}

export function useAgentSkills() {
  return useQuery({
    queryKey: ['agent-skills'],
    queryFn: getAgentSkills,
    staleTime: 60_000,
  })
}
