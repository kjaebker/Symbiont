import { apiFetch } from './_fetch'
import type { AgentSettings, AgentSkill } from './types'

export function getAgentSettings() {
  return apiFetch<AgentSettings>('/api/agent/settings')
}

export function updateAgentSettings(patch: Partial<AgentSettings>) {
  return apiFetch<AgentSettings>('/api/agent/settings', {
    method: 'PUT',
    body: JSON.stringify(patch),
  })
}

export function getAgentContext() {
  return apiFetch<{ context: string }>('/api/agent/context')
}

export function getAgentSkills() {
  return apiFetch<{ skills: AgentSkill[] }>('/api/agent/skills')
}
