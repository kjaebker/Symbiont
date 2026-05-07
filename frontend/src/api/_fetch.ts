import { getToken, clearToken } from './storage'

export class APIRequestError extends Error {
  code: string
  status: number

  constructor(message: string, code: string, status: number) {
    super(message)
    this.name = 'APIRequestError'
    this.code = code
    this.status = status
  }
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    ...((init?.headers as Record<string, string>) ?? {}),
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  headers['X-Source'] = 'ui'

  if (init?.body && typeof init.body === 'string') {
    headers['Content-Type'] = 'application/json'
  }

  const res = await fetch(path, { ...init, headers, cache: 'no-store' })

  if (res.status === 401) {
    clearToken()
    window.location.href = '/login'
    throw new APIRequestError('Unauthorized', 'unauthorized', 401)
  }

  if (res.status === 204) {
    return undefined as T
  }

  const body = await res.json()

  if (!res.ok) {
    throw new APIRequestError(
      body.error ?? 'Unknown error',
      body.code ?? 'unknown',
      res.status,
    )
  }

  return body as T
}

/**
 * Multipart upload helper. The standard fetch path can't be used because we
 * must NOT set Content-Type — the browser sets it with the multipart boundary.
 */
export async function uploadMultipart<T>(path: string, file: File, fieldName = 'image'): Promise<T> {
  const token = getToken()
  const form = new FormData()
  form.append(fieldName, file)
  const res = await fetch(path, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: form,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'upload failed', code: 'upload_error' }))
    throw new Error(err.error ?? 'upload failed')
  }
  return res.json() as Promise<T>
}
