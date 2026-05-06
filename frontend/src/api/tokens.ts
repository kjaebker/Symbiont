import { apiFetch } from './_fetch'
import type { AuthToken } from './types'

export function listTokens() {
  return apiFetch<{ tokens: AuthToken[] }>('/api/tokens')
}

export function createToken(label: string, scope: string = 'admin') {
  return apiFetch<{ id: number; token: string; label: string; scope: string }>('/api/tokens', {
    method: 'POST',
    body: JSON.stringify({ label, scope }),
  })
}

export function updateTokenScope(id: number, scope: string) {
  return apiFetch<AuthToken>(`/api/tokens/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ scope }),
  })
}

export function revokeToken(id: number) {
  return apiFetch<void>(`/api/tokens/${id}`, { method: 'DELETE' })
}
