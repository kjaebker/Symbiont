import { apiFetch } from './_fetch'
import type { DailyPrompt } from './types'

export type { DailyPrompt }

export function getDailyPrompt() {
  return apiFetch<{ prompt: DailyPrompt | null }>('/api/daily-prompt')
}

export function respondToPrompt(question: string, response: string) {
  return apiFetch<{ ok: boolean }>('/api/daily-prompt/respond', {
    method: 'POST',
    body: JSON.stringify({ question, response }),
  })
}
