import { apiFetch } from './_fetch'
import type { FeedStatus } from './types'

export function getFeedStatus() {
  return apiFetch<FeedStatus>('/api/feed')
}

export function setFeedMode(name: number, active: boolean) {
  return apiFetch<FeedStatus>('/api/feed', {
    method: 'PUT',
    body: JSON.stringify({ name, active }),
  })
}
